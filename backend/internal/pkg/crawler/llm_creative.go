package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"idea_arena/internal/biz"
	"idea_arena/internal/pkg/llm"

	"github.com/go-kratos/kratos/v2/log"
)

// ── 跨域碰撞素材 ──

var domainA = []string{
	"AI", "区块链", "量子计算", "脑机接口", "基因编辑", "合成生物",
	"AR/VR", "具身智能", "数字孪生", "边缘计算", "纳米技术",
	"MCP协议", "多模态AI", "端侧大模型", "AI Agent",
}

var domainB = []string{
	"殡葬", "宗教", "梦境", "考古", "深海", "太空殖民", "情感",
	"记忆", "衰老", "孤独", "动物语言", "意识", "时间", "死亡",
	"味觉", "直觉", "运气", "美", "信任", "创造力", "童年",
	"城市废墟", "声音景观", "气味", "触觉", "方言消亡", "手工艺消亡",
	"睡眠", "冥想", "宠物", "二次元", "非遗", "乡村振兴",
}

var humanProblems = []string{
	"如何消除教育不公平", "如何解决全球孤独感流行",
	"如何让每个人都有好医生", "如何消除语言隔阂",
	"如何让决策不再焦虑", "如何对抗信息过载",
	"如何让老年人不孤单", "如何拯救正在消亡的文化",
	"如何让所有人都能做创造性工作", "如何预防而不是治疗疾病",
	"如何让城市不再拥堵", "如何消除算法偏见",
	"如何让科研效率提升10倍", "如何解决睡眠危机",
	"如何让终身学习真正可行", "如何让远程协作像面对面一样高效",
}

const creativeGenPromptTemplate = `你是一个极具创造力的独立开发者和连续创业者。请生成 8 个前所未有的AI创业方向。

## 核心约束条件（必须严格遵守）
%s

## 要求
- 至少 3 个是"实际落地"型：基于最新AI技术（Agent、MCP、端侧模型等）的具体产品方向，个人开发者能独立实现
- 至少 2 个是"跨界碰撞"型：把AI和一个不常见的领域结合，但要确保个人开发者能切入
- 至少 1 个是"市场验证"型：已有初步市场信号、有付费用户群体的方向
- 其余的自由发挥，越具体越好
- 每个方向用一句话描述（10-30字），要具体到能直接立项
- 不要泛泛的"AI教育"，而是"AI驱动的自适应编程教学平台"这样具体
- 不要超出约束条件范围的天马行空想法（如纳米机器人、脑机接口等个人开发者不可能做的）

## 输出格式（严格JSON数组，不要其他文字）
["方向1", "方向2", "方向3", ...]`

// LLMCreativePlugin 基于LLM创意生成的话题发现插件
type LLMCreativePlugin struct {
	llmClient      *llm.Client
	log            *log.Helper
	getConstraints func() []string // 获取约束标签
}

var _ biz.CrawlerPluginInterface = (*LLMCreativePlugin)(nil)

func NewLLMCreativePlugin(llmClient *llm.Client, logger log.Logger, getConstraints func() []string) *LLMCreativePlugin {
	return &LLMCreativePlugin{
		llmClient:      llmClient,
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *LLMCreativePlugin) Name() string  { return "llm_creative" }
func (p *LLMCreativePlugin) Label() string { return "AI创意话题生成（LLM跨域碰撞+发散）" }

func (p *LLMCreativePlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 读取约束标签
	constraintBlock := "- 适合个人开发者或小团队\n- 低成本可启动\n- 有明确的商业变现路径"
	if p.getConstraints != nil {
		constraints := p.getConstraints()
		if len(constraints) > 0 {
			var lines []string
			for _, c := range constraints {
				lines = append(lines, "- "+c)
			}
			constraintBlock = strings.Join(lines, "\n")
		}
	}

	// 构建带约束条件的 prompt
	systemPrompt := fmt.Sprintf(creativeGenPromptTemplate, constraintBlock)

	// 随机注入碰撞素材作为灵感
	shuffledA := shuffleStrings(domainA)
	shuffledB := shuffleStrings(domainB)
	shuffledProblems := shuffleStrings(humanProblems)

	domA := shuffledA[:min(2, len(shuffledA))]
	domB := shuffledB[:min(3, len(shuffledB))]
	problem := shuffledProblems[0]

	hint := fmt.Sprintf("灵感素材（可用可不用）：技术[%s] × 领域[%s] | 人类难题[%s]",
		strings.Join(domA, "、"), strings.Join(domB, "、"), problem)

	response, err := p.llmClient.CallWithRole(ctx, "utility", systemPrompt, []llm.Message{
		{Role: "user", Content: hint},
	})
	if err != nil {
		p.log.Warnf("[LLMCreative] LLM call failed: %v", err)
		// 降级：用随机跨域碰撞生成话题
		return p.fallbackCollision(r, limit), nil
	}

	// 解析 JSON 数组
	cleaned := response
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var topics []string
	if err := json.Unmarshal([]byte(cleaned), &topics); err != nil {
		p.log.Warnf("[LLMCreative] JSON parse failed: %v, raw: %s", err, truncateStr(response, 300))
		return p.fallbackCollision(r, limit), nil
	}

	var result []*biz.RawTopic
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		result = append(result, &biz.RawTopic{
			Title:   t,
			Source:  "llm_creative",
			Snippet: fmt.Sprintf("AI创意生成 | 灵感: %s × %s", domA[0], domB[0]),
		})
	}

	// 额外补充几个纯随机碰撞话题
	collisions := p.fallbackCollision(r, 3)
	result = append(result, collisions...)

	if len(result) > limit {
		result = result[:limit]
	}

	p.log.Infof("[LLMCreative] Generated %d creative topics", len(result))
	return result, nil
}

// fallbackCollision 随机跨域碰撞生成话题（零成本降级方案）
func (p *LLMCreativePlugin) fallbackCollision(r *rand.Rand, count int) []*biz.RawTopic {
	var topics []*biz.RawTopic
	used := make(map[string]bool)

	for i := 0; i < count*3 && len(topics) < count; i++ {
		a := domainA[r.Intn(len(domainA))]
		b := domainB[r.Intn(len(domainB))]
		key := a + "×" + b
		if used[key] {
			continue
		}
		used[key] = true
		topics = append(topics, &biz.RawTopic{
			Title:   fmt.Sprintf("%s × %s", a, b),
			Source:  "collision",
			Snippet: "跨域碰撞随机生成",
		})
	}
	return topics
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
