package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// IdeaInfo 飞书通知需要的Idea信息（避免直接依赖biz包）
type IdeaInfo struct {
	ID               int64
	Topic            string
	ProductName      string
	OneLiner         string
	Status           string
	ScoreFeasibility float64
	ScoreEconomics   float64
	ScoreProfit      float64
	ScoreOverall     float64
	RoundCount       int
	Tags             []string
	Summary          string     // 辩论摘要
	FinalReport      string     // 深度可行性报告
	JudgeRefined     string     // 判官精修方案
	RoundScores      []float64  // 各轮综合评分（落地曲线）
}

// debateStats 辩论统计（用于定期汇总）
type debateStats struct {
	mu            sync.Mutex
	total         int
	graduated     int
	promising     int
	failed        int
	highScoreList []string // 高分项目摘要
	promisingList []string // 潜力项目摘要
	startTime     time.Time
}

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	conf   *conf.FeishuNotify
	client *http.Client
	log    *log.Helper
	stats  debateStats
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(c *conf.FeishuNotify, logger log.Logger) *FeishuNotifier {
	n := &FeishuNotifier{
		conf:   c,
		client: &http.Client{Timeout: 10 * time.Second},
		log:    log.NewHelper(logger),
	}
	n.stats.startTime = time.Now()
	return n
}

// IsEnabled 是否启用
func (n *FeishuNotifier) IsEnabled() bool {
	return n.conf != nil && n.conf.Enabled && n.conf.WebhookURL != ""
}

// SendDebateResult 发送辩论结果通知（丰富版：含价值分析、落地曲线、详情链接）
func (n *FeishuNotifier) SendDebateResult(ctx context.Context, idea *IdeaInfo) error {
	if !n.IsEnabled() {
		return nil
	}

	emoji := "❌"
	verdictText := "未通过"
	headerColor := "red"
	switch idea.Status {
	case "graduated":
		emoji = "🎉"
		verdictText = "通过毕业"
		headerColor = "green"
	case "promising":
		emoji = "💡"
		verdictText = "有潜力，值得深入"
		headerColor = "orange"
	default:
		emoji = "❌"
		verdictText = "未通过"
		headerColor = "red"
	}

	name := idea.ProductName
	if name == "" {
		name = fmt.Sprintf("Idea #%d", idea.ID)
	}

	title := fmt.Sprintf("%s %s — 辩论完成", emoji, name)

	// ── 基本信息 ──
	var lines []string
	lines = append(lines, fmt.Sprintf("**话题**: %s", idea.Topic))
	if idea.OneLiner != "" {
		lines = append(lines, fmt.Sprintf("**一句话**: %s", idea.OneLiner))
	}
	lines = append(lines, "")

	// ── 评分详情 ──
	lines = append(lines, fmt.Sprintf("📊 **综合评分: %.1f / 10**", idea.ScoreOverall))
	lines = append(lines, fmt.Sprintf("%s 可行性: **%.1f** | %s 经济性: **%.1f** | %s 盈利性: **%.1f**",
		scoreEmoji(idea.ScoreFeasibility), idea.ScoreFeasibility,
		scoreEmoji(idea.ScoreEconomics), idea.ScoreEconomics,
		scoreEmoji(idea.ScoreProfit), idea.ScoreProfit))
	lines = append(lines, "")

	// ── 落地曲线（各轮评分走势）──
	if len(idea.RoundScores) >= 2 {
		lines = append(lines, "📈 **评分走势（落地曲线）**")
		curve := ""
		for i, s := range idea.RoundScores {
			bar := scoreBar(s)
			if i < len(idea.RoundScores)-1 {
				curve += fmt.Sprintf("R%d %.1f %s → ", i+1, s, bar)
			} else {
				curve += fmt.Sprintf("R%d **%.1f** %s", i+1, s, bar)
			}
		}
		lines = append(lines, curve)
		// 趋势判断
		first, last := idea.RoundScores[0], idea.RoundScores[len(idea.RoundScores)-1]
		if last > first+0.5 {
			lines = append(lines, "🔺 评分呈上升趋势，越辩越有价值")
		} else if last < first-0.5 {
			lines = append(lines, "🔻 评分呈下降趋势，存在关键瓶颈")
		} else {
			lines = append(lines, "➡️ 评分稳定，多轮验证结果一致")
		}
		lines = append(lines, "")
	}

	// ── 价值分析摘要 ──
	if idea.Summary != "" {
		lines = append(lines, "💎 **价值分析**")
		summary := idea.Summary
		if len(summary) > 300 {
			summary = summary[:300] + "..."
		}
		lines = append(lines, summary)
		lines = append(lines, "")
	}

	lines = append(lines, fmt.Sprintf("🏷 **裁决**: %s", verdictText))
	lines = append(lines, fmt.Sprintf("🔄 **辩论轮次**: %d 轮", idea.RoundCount))
	if len(idea.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("🏷 **标签**: %s", strings.Join(idea.Tags, "、")))
	}

	content := strings.Join(lines, "\n")
	detailURL := fmt.Sprintf("https://ai-withme.com/ideas/%d", idea.ID)
	return n.sendCardWithActions(ctx, title, headerColor, content, detailURL)
}

// SendHighScoreAlert 发送高分提醒（含价值分析和详情链接）
func (n *FeishuNotifier) SendHighScoreAlert(ctx context.Context, idea *IdeaInfo) error {
	if !n.IsEnabled() {
		return nil
	}

	name := idea.ProductName
	if name == "" {
		name = fmt.Sprintf("Idea #%d", idea.ID)
	}

	title := fmt.Sprintf("🏆 高分项目: %s (%.1f分)", name, idea.ScoreOverall)

	var lines []string
	lines = append(lines, fmt.Sprintf("**话题**: %s", idea.Topic))
	if idea.OneLiner != "" {
		lines = append(lines, fmt.Sprintf("**一句话**: %s", idea.OneLiner))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("📊 **综合评分: %.1f / 10** 🔥🔥🔥", idea.ScoreOverall))
	lines = append(lines, fmt.Sprintf("%s 可行性: **%.1f** | %s 经济性: **%.1f** | %s 盈利性: **%.1f**",
		scoreEmoji(idea.ScoreFeasibility), idea.ScoreFeasibility,
		scoreEmoji(idea.ScoreEconomics), idea.ScoreEconomics,
		scoreEmoji(idea.ScoreProfit), idea.ScoreProfit))
	lines = append(lines, "")

	// 价值分析
	if idea.Summary != "" {
		lines = append(lines, "💎 **价值分析**")
		summary := idea.Summary
		if len(summary) > 400 {
			summary = summary[:400] + "..."
		}
		lines = append(lines, summary)
		lines = append(lines, "")
	}

	// 落地曲线
	if len(idea.RoundScores) >= 2 {
		lines = append(lines, "📈 **评分走势**")
		curve := ""
		for i, s := range idea.RoundScores {
			bar := scoreBar(s)
			if i < len(idea.RoundScores)-1 {
				curve += fmt.Sprintf("R%d %.1f %s → ", i+1, s, bar)
			} else {
				curve += fmt.Sprintf("R%d **%.1f** %s", i+1, s, bar)
			}
		}
		lines = append(lines, curve)
		lines = append(lines, "")
	}

	lines = append(lines, "⚡ **该项目评分超过 8 分，值得重点关注和落地！**")
	if len(idea.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("🏷 **标签**: %s", strings.Join(idea.Tags, "、")))
	}

	content := strings.Join(lines, "\n")
	detailURL := fmt.Sprintf("https://ai-withme.com/ideas/%d", idea.ID)
	return n.sendCardWithActions(ctx, title, "red", content, detailURL)
}

// RecordDebateResult 记录辩论结果到统计（线程安全）
func (n *FeishuNotifier) RecordDebateResult(idea *IdeaInfo) {
	n.stats.mu.Lock()
	defer n.stats.mu.Unlock()

	n.stats.total++
	name := idea.ProductName
	if name == "" {
		name = fmt.Sprintf("Idea #%d", idea.ID)
	}

	switch idea.Status {
	case "graduated":
		n.stats.graduated++
		n.stats.highScoreList = append(n.stats.highScoreList, fmt.Sprintf("🎉 %s（%.1f分）", name, idea.ScoreOverall))
	case "promising":
		n.stats.promising++
		n.stats.promisingList = append(n.stats.promisingList, fmt.Sprintf("💡 %s（%.1f分）", name, idea.ScoreOverall))
	default:
		n.stats.failed++
	}
}

// SendSummary 发送定期汇总通知并重置统计
func (n *FeishuNotifier) SendSummary(ctx context.Context) error {
	if !n.IsEnabled() {
		return nil
	}

	n.stats.mu.Lock()
	total := n.stats.total
	graduated := n.stats.graduated
	promising := n.stats.promising
	failed := n.stats.failed
	highScoreList := n.stats.highScoreList
	promisingList := n.stats.promisingList
	startTime := n.stats.startTime
	// 重置统计
	n.stats.total = 0
	n.stats.graduated = 0
	n.stats.promising = 0
	n.stats.failed = 0
	n.stats.highScoreList = nil
	n.stats.promisingList = nil
	n.stats.startTime = time.Now()
	n.stats.mu.Unlock()

	if total == 0 {
		n.log.Infof("[Feishu] 汇总周期内无辩论，跳过通知")
		return nil
	}

	// 计算通过率
	passRate := float64(graduated+promising) / float64(total) * 100

	title := fmt.Sprintf("📋 辩论竞技场 — %s 汇总", startTime.Format("01/02 15:04")+" ~ "+time.Now().Format("15:04"))

	var lines []string
	lines = append(lines, fmt.Sprintf("🔢 **总辩论数**: %d 场", total))
	lines = append(lines, fmt.Sprintf("✅ **通过率**: %.0f%%（高分 %d + 潜力 %d）/ 总 %d", passRate, graduated, promising, total))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("🎉 **毕业项目**: %d 个", graduated))
	lines = append(lines, fmt.Sprintf("💡 **潜力项目**: %d 个", promising))
	lines = append(lines, fmt.Sprintf("❌ **未通过**: %d 个", failed))

	if len(highScoreList) > 0 {
		lines = append(lines, "")
		lines = append(lines, "---")
		lines = append(lines, "**🏆 高分项目明细**")
		for _, item := range highScoreList {
			lines = append(lines, "- "+item)
		}
	}

	if len(promisingList) > 0 {
		lines = append(lines, "")
		if len(highScoreList) == 0 {
			lines = append(lines, "---")
		}
		lines = append(lines, "**💡 潜力项目明细**")
		for _, item := range promisingList {
			lines = append(lines, "- "+item)
		}
	}

	content := strings.Join(lines, "\n")
	return n.sendCard(ctx, title, content)
}

// StartSummaryScheduler 启动定期汇总通知调度器（每3小时）
func (n *FeishuNotifier) StartSummaryScheduler(ctx context.Context) {
	if !n.IsEnabled() {
		return
	}
	n.log.Infof("[Feishu] 汇总通知调度器已启动，每3小时发送一次")
	go func() {
		ticker := time.NewTicker(3 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := n.SendSummary(context.Background()); err != nil {
					n.log.Warnf("[Feishu] 汇总通知发送失败: %v", err)
				}
			}
		}
	}()
}

// scoreEmoji 根据分数返回对应 emoji
func scoreEmoji(score float64) string {
	if score >= 8 {
		return "🟢"
	} else if score >= 6 {
		return "🟡"
	} else if score >= 4 {
		return "🟠"
	}
	return "🔴"
}

// scoreBar 根据分数生成可视化条形
func scoreBar(score float64) string {
	filled := int(score)
	if filled > 10 {
		filled = 10
	}
	empty := 10 - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// sendCard 发送飞书卡片消息（无按钮）
func (n *FeishuNotifier) sendCard(ctx context.Context, title, content string) error {
	return n.sendCardWithActions(ctx, title, "blue", content, "")
}

// sendCardWithActions 发送飞书卡片消息（带可选操作按钮）
func (n *FeishuNotifier) sendCardWithActions(ctx context.Context, title, headerColor, content, detailURL string) error {
	elements := []map[string]interface{}{
		{
			"tag":     "markdown",
			"content": content,
		},
	}

	// 添加分割线和操作按钮
	if detailURL != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "action",
			"actions": []map[string]interface{}{
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "📋 查看完整报告",
					},
					"type":      "primary",
					"multi_url": map[string]string{"url": detailURL, "pc_url": detailURL, "android_url": detailURL, "ios_url": detailURL},
				},
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "🏠 排行榜",
					},
					"type":      "default",
					"multi_url": map[string]string{"url": "https://ai-withme.com/ideas", "pc_url": "https://ai-withme.com/ideas", "android_url": "https://ai-withme.com/ideas", "ios_url": "https://ai-withme.com/ideas"},
				},
			},
		})
	}

	msg := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": title,
				},
				"template": headerColor,
			},
			"elements": elements,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.conf.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create feishu request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}

	n.log.Infof("Feishu notification sent: %s", title)
	return nil
}

// sendText 发送纯文本消息（备用）
func (n *FeishuNotifier) sendText(ctx context.Context, text string) error {
	msg := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.conf.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
