package biz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"idea_arena/internal/pkg/llm"

	"github.com/go-kratos/kratos/v2/log"
)

// DiscoveryUsecase 话题发现业务用例
type DiscoveryUsecase struct {
	repo      DiscoveryRepo
	ideaRepo  IdeaRepo
	llmClient *llm.Client
	debateUc  *DebateUsecase
	log       *log.Helper

	providerManager *llm.ProviderManager
	providerRepo    ProviderRepository

	plugins   map[string]CrawlerPluginInterface
	pluginsMu sync.RWMutex

	// 调度控制
	stopCh  chan struct{}
	running bool
	mu      sync.Mutex
}

// NewDiscoveryUsecase 创建话题发现用例
func NewDiscoveryUsecase(
	repo DiscoveryRepo,
	ideaRepo IdeaRepo,
	llmClient *llm.Client,
	logger log.Logger,
) *DiscoveryUsecase {
	return &DiscoveryUsecase{
		repo:      repo,
		ideaRepo:  ideaRepo,
		llmClient: llmClient,
		log:       log.NewHelper(logger),
		plugins:   make(map[string]CrawlerPluginInterface),
		stopCh:    make(chan struct{}),
	}
}

// SetDebateUsecase 注入辩论引擎（避免循环依赖）
func (uc *DiscoveryUsecase) SetDebateUsecase(debateUc *DebateUsecase) {
	uc.debateUc = debateUc
}

// SetProviderManager 注入多服务商调用能力（用于插件 AI 扩展）
func (uc *DiscoveryUsecase) SetProviderManager(pm *llm.ProviderManager, providerRepo ProviderRepository) {
	uc.providerManager = pm
	uc.providerRepo = providerRepo
}

// RegisterPlugin 注册爬虫插件
func (uc *DiscoveryUsecase) RegisterPlugin(p CrawlerPluginInterface) {
	uc.pluginsMu.Lock()
	defer uc.pluginsMu.Unlock()
	uc.plugins[p.Name()] = p

	// 确保 DB 中有该插件记录，并同步最新 Label
	ctx := context.Background()
	existing, err := uc.repo.GetPlugin(ctx, p.Name())
	if err != nil {
		_ = uc.repo.UpsertPlugin(ctx, &CrawlerPlugin{
			Name:    p.Name(),
			Label:   p.Label(),
			Enabled: false,
			Config:  "{}",
		})
	} else if existing.Label != p.Label() {
		existing.Label = p.Label()
		_ = uc.repo.UpsertPlugin(ctx, existing)
	}
}

// ---- Settings ----

// GetSettings 获取系统设置
func (uc *DiscoveryUsecase) GetSettings(ctx context.Context) (*SystemSettings, error) {
	all, err := uc.repo.GetAllSettings(ctx)
	if err != nil {
		return DefaultSettings(), nil
	}

	s := DefaultSettings()
	if v, ok := all["debate_max_rounds"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.DebateMaxRounds = n
		}
	}
	if v, ok := all["debate_graduation_score"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			s.DebateGraduationScore = f
		}
	}
	if v, ok := all["debate_timeout"]; ok {
		s.DebateTimeout = v
	}
	if v, ok := all["discovery_interval"]; ok {
		s.DiscoveryInterval = v
	}
	if v, ok := all["discovery_enabled"]; ok {
		s.DiscoveryEnabled = v == "true"
	}
	if v, ok := all["topics_per_source"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.TopicsPerSource = n
		}
	}
	if v, ok := all["auto_submit_debate"]; ok {
		s.AutoSubmitDebate = v == "true"
	}
	return s, nil
}

// SaveSettings 保存系统设置
func (uc *DiscoveryUsecase) SaveSettings(ctx context.Context, s *SystemSettings) error {
	pairs := map[string]string{
		"debate_max_rounds":       strconv.Itoa(s.DebateMaxRounds),
		"debate_graduation_score": fmt.Sprintf("%.1f", s.DebateGraduationScore),
		"debate_timeout":          s.DebateTimeout,
		"discovery_interval":      s.DiscoveryInterval,
		"discovery_enabled":       fmt.Sprintf("%t", s.DiscoveryEnabled),
		"topics_per_source":       strconv.Itoa(s.TopicsPerSource),
		"auto_submit_debate":      fmt.Sprintf("%t", s.AutoSubmitDebate),
	}
	for k, v := range pairs {
		if err := uc.repo.SetSetting(ctx, k, v); err != nil {
			return fmt.Errorf("save setting %s: %w", k, err)
		}
	}
	return nil
}

// ---- Keywords ----

func (uc *DiscoveryUsecase) ListKeywords(ctx context.Context) ([]*SearchKeyword, error) {
	return uc.repo.ListKeywords(ctx)
}

func (uc *DiscoveryUsecase) CreateKeyword(ctx context.Context, keyword string) (*SearchKeyword, error) {
	return uc.repo.CreateKeyword(ctx, keyword)
}

func (uc *DiscoveryUsecase) UpdateKeyword(ctx context.Context, id int64, keyword string, enabled bool) error {
	return uc.repo.UpdateKeyword(ctx, id, keyword, enabled)
}

func (uc *DiscoveryUsecase) DeleteKeyword(ctx context.Context, id int64) error {
	return uc.repo.DeleteKeyword(ctx, id)
}

// GetEnabledKeywords 获取已启用的关键词列表（给 KeywordSearchPlugin 用）
func (uc *DiscoveryUsecase) GetEnabledKeywords() []string {
	ctx := context.Background()
	keywords, err := uc.repo.ListKeywords(ctx)
	if err != nil {
		return nil
	}
	var result []string
	for _, kw := range keywords {
		if kw.Enabled {
			result = append(result, kw.Keyword)
		}
	}
	return result
}

// ---- Tags ----

func (uc *DiscoveryUsecase) ListTags(ctx context.Context) ([]*DiscoveryTag, error) {
	return uc.repo.ListTags(ctx)
}

func (uc *DiscoveryUsecase) CreateTag(ctx context.Context, tag, category string) (*DiscoveryTag, error) {
	if category == "" {
		category = "constraint"
	}
	return uc.repo.CreateTag(ctx, tag, category)
}

func (uc *DiscoveryUsecase) UpdateTag(ctx context.Context, id int64, tag, category string, enabled bool) error {
	return uc.repo.UpdateTag(ctx, id, tag, category, enabled)
}

func (uc *DiscoveryUsecase) DeleteTag(ctx context.Context, id int64) error {
	return uc.repo.DeleteTag(ctx, id)
}

// GetEnabledConstraintTags 获取已启用的约束标签（给插件注入用）
func (uc *DiscoveryUsecase) GetEnabledConstraintTags() []string {
	ctx := context.Background()
	tags, err := uc.repo.ListTagsByCategory(ctx, "constraint")
	if err != nil {
		return nil
	}
	var result []string
	for _, t := range tags {
		result = append(result, t.Tag)
	}
	return result
}

// GetEnabledTrendQueries 获取已启用的趋势搜索词（给 trend_search 插件用）
func (uc *DiscoveryUsecase) GetEnabledTrendQueries() (cn []string, en []string) {
	ctx := context.Background()
	cnTags, _ := uc.repo.ListTagsByCategory(ctx, "trend_query_cn")
	for _, t := range cnTags {
		cn = append(cn, t.Tag)
	}
	enTags, _ := uc.repo.ListTagsByCategory(ctx, "trend_query_en")
	for _, t := range enTags {
		en = append(en, t.Tag)
	}
	return
}

// ---- Plugins ----

func (uc *DiscoveryUsecase) ListPlugins(ctx context.Context) ([]*CrawlerPlugin, error) {
	return uc.repo.ListPlugins(ctx)
}

func (uc *DiscoveryUsecase) TogglePlugin(ctx context.Context, name string, enabled bool) error {
	return uc.repo.TogglePlugin(ctx, name, enabled)
}

func (uc *DiscoveryUsecase) UpdatePluginConfig(ctx context.Context, name, config string) error {
	plugin, err := uc.repo.GetPlugin(ctx, name)
	if err != nil {
		return err
	}
	plugin.Config = config
	return uc.repo.UpsertPlugin(ctx, plugin)
}

func (uc *DiscoveryUsecase) GetPluginConfig(name string) string {
	plugin, err := uc.repo.GetPlugin(context.Background(), name)
	if err != nil || plugin == nil {
		return "{}"
	}
	if strings.TrimSpace(plugin.Config) == "" {
		return "{}"
	}
	return plugin.Config
}

func (uc *DiscoveryUsecase) ExpandKeywordByAI(
	ctx context.Context,
	keywords []string,
	providerID int64,
	model string,
	extraPerKeyword int,
) ([]string, error) {
	if len(keywords) == 0 {
		return nil, nil
	}
	if uc.providerManager == nil {
		return nil, fmt.Errorf("provider manager not initialized")
	}
	if extraPerKeyword <= 0 {
		extraPerKeyword = 2
	}
	if extraPerKeyword > 5 {
		extraPerKeyword = 5
	}

	if providerID <= 0 && uc.providerRepo != nil {
		if defaultProvider, err := uc.providerRepo.GetDefault(ctx); err == nil && defaultProvider != nil {
			providerID = defaultProvider.ID
		}
	}
	if providerID <= 0 {
		return nil, fmt.Errorf("no available provider")
	}
	if strings.TrimSpace(model) == "" {
		model = "qwen-plus"
	}

	baseKeywordsJSON, _ := json.Marshal(keywords)
	systemPrompt := `你是“搜索关键词扩展器”。
你的任务是基于给定关键词，生成可用于搜索引擎抓取的“场景化长尾关键词”。
请保证结果多样化、贴近真实行业场景，并带有一定随机领域覆盖。`

	userPrompt := fmt.Sprintf(`请基于以下关键词进行扩展：
%s

要求：
1) 每个原关键词扩展 %d 条，优先输出“关键词 + 行业/场景 + 实际应用”形式。
2) 覆盖尽量不同的随机领域，例如：医疗、教育、制造、物流、法务、财税、零售、农业、政务、跨境、电商、人力资源等。
3) 语言以中文为主，短语可直接用于搜索，不要写解释句。
4) 去重、避免空泛词（如“未来趋势”“全面分析”）。
5) 仅输出 JSON 数组，不要 markdown，不要额外说明。

输出示例：
["AI自动化办公 在外贸跟单中的实际应用","AI自动化办公 在口腔诊所预约管理中的实际应用"]`,
		string(baseKeywordsJSON), extraPerKeyword,
	)

	expandCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	response, err := uc.providerManager.CallWithProvider(
		expandCtx,
		providerID,
		model,
		systemPrompt,
		[]llm.Message{{Role: "user", Content: userPrompt}},
		&llm.CallOptions{Temperature: 1.1, MaxTokens: 600},
	)
	if err != nil {
		return nil, err
	}

	expanded := parseExpandedKeywordsFromAIResponse(response)
	if len(expanded) == 0 {
		// 轻量重试一次，降低温度并强化输出约束，减少偶发空结果
		retryResponse, retryErr := uc.providerManager.CallWithProvider(
			expandCtx,
			providerID,
			model,
			systemPrompt,
			[]llm.Message{{Role: "user", Content: userPrompt + "\n\n再次强调：仅返回 JSON 字符串数组，不要返回对象、markdown、解释。"}},
			&llm.CallOptions{Temperature: 0.7, MaxTokens: 600},
		)
		if retryErr == nil {
			expanded = parseExpandedKeywordsFromAIResponse(retryResponse)
		}
	}

	seen := make(map[string]struct{})
	baseSet := make(map[string]struct{})
	for _, kw := range keywords {
		baseSet[strings.ToLower(strings.TrimSpace(kw))] = struct{}{}
	}
	result := make([]string, 0, len(expanded))
	for _, kw := range expanded {
		candidate := strings.TrimSpace(strings.Trim(kw, `"`))
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if _, exists := baseSet[key]; exists {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result, nil
}

func parseExpandedKeywordsFromAIResponse(response string) []string {
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return nil
	}

	// 1) 直接是 JSON 数组
	var expanded []string
	if err := json.Unmarshal([]byte(cleaned), &expanded); err == nil && len(expanded) > 0 {
		return normalizeExpandedKeywords(expanded)
	}

	// 2) 兼容 JSON 对象结构
	var objectPayload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &objectPayload); err == nil && len(objectPayload) > 0 {
		for _, key := range []string{"keywords", "items", "data", "result"} {
			raw, ok := objectPayload[key]
			if !ok {
				continue
			}
			var arr []string
			if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
				return normalizeExpandedKeywords(arr)
			}
		}
	}

	// 3) 尝试提取首个 JSON 数组片段
	if start := strings.Index(cleaned, "["); start >= 0 {
		if end := strings.LastIndex(cleaned, "]"); end > start {
			var arr []string
			if err := json.Unmarshal([]byte(cleaned[start:end+1]), &arr); err == nil && len(arr) > 0 {
				return normalizeExpandedKeywords(arr)
			}
		}
	}

	// 4) 兼容逐行列表/编号列表
	lines := strings.Split(cleaned, "\n")
	fromLines := make([]string, 0, len(lines))
	bulletPrefix := regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)、])\s*`)
	for _, line := range lines {
		candidate := strings.TrimSpace(bulletPrefix.ReplaceAllString(line, ""))
		candidate = strings.Trim(candidate, `"'`)
		if candidate == "" {
			continue
		}
		fromLines = append(fromLines, candidate)
	}
	return normalizeExpandedKeywords(fromLines)
}

func normalizeExpandedKeywords(keywords []string) []string {
	seen := make(map[string]struct{}, len(keywords))
	result := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		candidate := strings.TrimSpace(strings.Trim(keyword, `"'`))
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

// TestPlugin 测试单个插件抓取能力（不入库）
func (uc *DiscoveryUsecase) TestPlugin(ctx context.Context, name string, limit int) ([]*RawTopic, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	uc.pluginsMu.RLock()
	plugin, ok := uc.plugins[name]
	uc.pluginsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	topics, err := plugin.Fetch(ctx, limit)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

// ---- Topics ----

func (uc *DiscoveryUsecase) ListTopics(ctx context.Context, status string, page, pageSize int) ([]*DiscoveredTopic, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return uc.repo.ListTopics(ctx, status, page, pageSize)
}

// ListTopicSourceStats 来源命中率统计
func (uc *DiscoveryUsecase) ListTopicSourceStats(ctx context.Context, days int) ([]*TopicSourceStat, error) {
	if days <= 0 {
		days = 30
	}

	stats, err := uc.repo.ListTopicSourceStats(ctx, days)
	if err != nil {
		return nil, err
	}

	for _, stat := range stats {
		if stat == nil || stat.TotalCount <= 0 {
			continue
		}
		hitCount := stat.RecommendedCount + stat.SubmittedCount
		stat.HitRate = float64(hitCount) / float64(stat.TotalCount)
		stat.SubmitRate = float64(stat.SubmittedCount) / float64(stat.TotalCount)
	}

	return stats, nil
}

func (uc *DiscoveryUsecase) DismissTopic(ctx context.Context, id int64) error {
	return uc.repo.UpdateTopicStatus(ctx, id, "dismissed")
}

// SubmitToDebate 将发现的话题提交为 Idea 进行辩论
func (uc *DiscoveryUsecase) SubmitToDebate(ctx context.Context, topicID int64) (*Idea, error) {
	topic, err := uc.repo.GetTopic(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// 用 suggested_topic 或原始标题创建 Idea
	topicText := topic.Title
	if topic.Recommendation != "" {
		// 尝试从推荐理由中提取 suggested_topic
		topicText = extractSuggestedTopic(topic.Recommendation, topic.Title)
	}

	idea, err := uc.ideaRepo.Create(ctx, &Idea{
		Topic:  topicText,
		Status: "pending",
	})
	if err != nil {
		return nil, fmt.Errorf("create idea failed: %w", err)
	}

	_ = uc.repo.SetTopicIdeaID(ctx, topicID, idea.ID)
	return idea, nil
}

// ---- Discovery Execution ----

// RunDiscovery 手动触发一次话题发现（仅发现，不自动辩论）
func (uc *DiscoveryUsecase) RunDiscovery(ctx context.Context) (int, error) {
	settings, _ := uc.GetSettings(ctx)
	limit := 10
	if settings != nil && settings.TopicsPerSource > 0 {
		limit = settings.TopicsPerSource
	}
	topics, err := uc.fetchAndDedupWithLimit(ctx, limit)
	if err != nil {
		return 0, err
	}
	if len(topics) == 0 {
		return 0, nil
	}

	// AI 分析推荐（异步）
	go uc.analyzeTopics(context.Background(), topics)

	return len(topics), nil
}

// 背压 & 生命周期常量
const (
	backpressureSkipThreshold = 50 // pending > 50 → 跳过抓取
	backpressureHalfThreshold = 30 // pending 30-50 → 减半抓取
	staleTopicDays            = 7  // pending 超过 7 天 → 自动 dismiss
	lowScoreDismissThreshold  = 3.0
	pendingTopicCap           = 50 // pending 上限
)

// RunAutoDiscoveryAndDebate 全自动流程：清理 → 背压发现 → 全量精选 → 创建Idea → 启动辩论 → 消化存量
func (uc *DiscoveryUsecase) RunAutoDiscoveryAndDebate(ctx context.Context) (int, error) {
	uc.log.Info("[AutoDiscovery] ========== 全自动发现+辩论流程开始 ==========")

	// Step 0: 话题生命周期清理
	uc.cleanupStaleTopics(ctx)

	// Step 1: 背压检查 — 根据 pending 数量决定是否抓取
	pendingCount, _ := uc.repo.CountTopicsByStatus(ctx, "pending")
	uc.log.Infof("[AutoDiscovery] Step1: 当前 pending topics = %d", pendingCount)

	settings, _ := uc.GetSettings(ctx)
	if settings == nil {
		settings = DefaultSettings()
	}
	limit := settings.TopicsPerSource
	if limit <= 0 {
		limit = 10
	}

	switch {
	case pendingCount > int64(backpressureSkipThreshold):
		uc.log.Infof("[AutoDiscovery] 背压触发：pending=%d > %d，跳过本轮抓取，专注消化存量", pendingCount, backpressureSkipThreshold)
		// 不抓取，直接进入精选和辩论
	case pendingCount > int64(backpressureHalfThreshold):
		halfLimit := limit / 2
		if halfLimit < 1 {
			halfLimit = 1
		}
		uc.log.Infof("[AutoDiscovery] 背压减半：pending=%d，topics_per_source %d → %d", pendingCount, limit, halfLimit)
		newTopics, err := uc.fetchAndDedupWithLimit(ctx, halfLimit)
		if err != nil {
			uc.log.Warnf("[AutoDiscovery] fetch error: %v", err)
		} else {
			uc.log.Infof("[AutoDiscovery] 减半抓取发现 %d 个新话题", len(newTopics))
		}
	default:
		newTopics, err := uc.fetchAndDedupWithLimit(ctx, limit)
		if err != nil {
			uc.log.Warnf("[AutoDiscovery] fetch error: %v", err)
		} else {
			uc.log.Infof("[AutoDiscovery] 正常抓取发现 %d 个新话题", len(newTopics))
		}
	}

	// Step 2: 获取已有项目摘要用于语义去重
	existingSummaries := uc.buildExistingSummaries(ctx)
	uc.log.Infof("[AutoDiscovery] Step2: 已有 %d 个项目用于语义去重", len(existingSummaries))

	// Step 3: 从全量 pending 池中取 top-N 候选（不只新批次）
	availableSlots := 0
	if uc.debateUc != nil {
		availableSlots = uc.debateUc.AvailableSlots()
	}
	if availableSlots <= 0 {
		uc.log.Info("[AutoDiscovery] 无可用辩论槽位，跳过精选。尝试消化存量 pending ideas")
		drained := uc.drainPendingIdeas(ctx)
		return drained, nil
	}

	// 动态配额：可用槽位数即为精选上限，候选池取 3 倍
	selectionQuota := availableSlots
	candidatePoolSize := selectionQuota * 3
	if candidatePoolSize < 10 {
		candidatePoolSize = 10
	}

	candidates, err := uc.repo.ListTopPendingTopics(ctx, candidatePoolSize)
	if err != nil || len(candidates) == 0 {
		uc.log.Info("[AutoDiscovery] 无 pending 候选话题")
		drained := uc.drainPendingIdeas(ctx)
		return drained, nil
	}
	uc.log.Infof("[AutoDiscovery] Step3: 从全量 pending 池取 %d 个候选（可用槽位=%d，配额=%d）", len(candidates), availableSlots, selectionQuota)

	// Step 4: 语义去重阈值门禁
	filteredTopics, reviewCount, dismissedCount := uc.applySemanticDedupThreshold(ctx, candidates, existingSummaries)
	uc.log.Infof("[AutoDiscovery] Step4: 语义过滤后保留 %d，人工复核 %d，自动过滤 %d", len(filteredTopics), reviewCount, dismissedCount)
	if len(filteredTopics) == 0 {
		uc.log.Info("[AutoDiscovery] 语义过滤后无候选")
		drained := uc.drainPendingIdeas(ctx)
		return drained, nil
	}

	// Step 5: AI 统一决策 — 动态配额精选
	selected, err := uc.aiSelectTopicsWithQuota(ctx, filteredTopics, existingSummaries, selectionQuota)
	if err != nil {
		uc.log.Warnf("[AutoDiscovery] AI selection failed: %v, using top topics by score", err)
		selected = uc.fallbackSelect(filteredTopics, selectionQuota)
	}

	if len(selected) == 0 {
		uc.log.Info("[AutoDiscovery] AI 未精选任何话题")
		drained := uc.drainPendingIdeas(ctx)
		return drained, nil
	}

	uc.log.Infof("[AutoDiscovery] Step5: AI精选了 %d 个话题进行辩论", len(selected))

	// Step 6: 为精选话题创建 Idea 并启动辩论
	debateCount := uc.startDebatesForSelected(ctx, selected)

	// Step 7: 消化存量 pending ideas（如果还有空闲槽位）
	drained := uc.drainPendingIdeas(ctx)
	totalStarted := debateCount + drained

	uc.log.Infof("[AutoDiscovery] ========== 完成：精选 %d + 存量消化 %d = 共启动 %d 场辩论 ==========", debateCount, drained, totalStarted)
	return totalStarted, nil
}

// buildExistingSummaries 构建已有项目摘要列表
func (uc *DiscoveryUsecase) buildExistingSummaries(ctx context.Context) []existingIdeaSummary {
	existingIdeas, _ := uc.ideaRepo.List(ctx, &IdeaListQuery{Page: 1, PageSize: 200, SortBy: "created_at", SortOrder: "desc"})
	var summaries []existingIdeaSummary
	if existingIdeas != nil {
		for _, idea := range existingIdeas.Items {
			summaries = append(summaries, existingIdeaSummary{
				ProductName: idea.ProductName,
				Topic:       idea.Topic,
				OneLiner:    idea.OneLiner,
				Tags:        idea.Tags,
				Score:       idea.ScoreOverall,
				Status:      idea.Status,
			})
		}
	}
	return summaries
}

// cleanupStaleTopics 话题生命周期清理
func (uc *DiscoveryUsecase) cleanupStaleTopics(ctx context.Context) {
	// 1. 超过 7 天的 pending → dismiss
	staleTime := time.Now().AddDate(0, 0, -staleTopicDays)
	staleCount, err := uc.repo.DismissStaleTopics(ctx, staleTime)
	if err == nil && staleCount > 0 {
		uc.log.Infof("[Cleanup] 清理过期话题: %d 条（>%d天）", staleCount, staleTopicDays)
	}

	// 2. 低分 pending → dismiss
	lowCount, err := uc.repo.DismissLowScoreTopics(ctx, lowScoreDismissThreshold)
	if err == nil && lowCount > 0 {
		uc.log.Infof("[Cleanup] 清理低分话题: %d 条（score<%.1f）", lowCount, lowScoreDismissThreshold)
	}

	// 3. pending 上限裁剪
	trimCount, err := uc.repo.TrimPendingTopics(ctx, pendingTopicCap)
	if err == nil && trimCount > 0 {
		uc.log.Infof("[Cleanup] 裁剪超限话题: %d 条（保留 top %d）", trimCount, pendingTopicCap)
	}
}

// fetchAndDedupWithLimit 带自定义 limit 的抓取去重
func (uc *DiscoveryUsecase) fetchAndDedupWithLimit(ctx context.Context, limit int) ([]*DiscoveredTopic, error) {
	// 获取启用的插件
	dbPlugins, err := uc.repo.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}
	enabledPlugins := make(map[string]bool)
	for _, p := range dbPlugins {
		enabledPlugins[p.Name] = p.Enabled
	}

	batchID := fmt.Sprintf("batch_%d", time.Now().Unix())
	var allRawTopics []*RawTopic

	uc.pluginsMu.RLock()
	for name, plugin := range uc.plugins {
		if !enabledPlugins[name] {
			continue
		}
		uc.log.Infof("[Discovery] Fetching from plugin: %s (limit=%d)", name, limit)
		topics, fetchErr := plugin.Fetch(ctx, limit)
		if fetchErr != nil {
			uc.log.Warnf("[Discovery] Plugin %s fetch error: %v", name, fetchErr)
			continue
		}
		uc.log.Infof("[Discovery] Plugin %s returned %d topics", name, len(topics))
		allRawTopics = append(allRawTopics, topics...)
	}
	uc.pluginsMu.RUnlock()

	if len(allRawTopics) == 0 {
		return nil, nil
	}

	sourceFactors := uc.buildSourceFeedbackFactors(ctx)
	var newTopics []*DiscoveredTopic
	for _, rt := range allRawTopics {
		hash := contentHash(rt.Title, rt.Source)
		exists, err := uc.repo.ExistsByHash(ctx, hash)
		if err != nil || exists {
			continue
		}
		painScore, trendScore, feasibilityScore, monetizationScore, noveltyScore, totalScore := calculateRuleScores(rt)
		totalScore = applySourceFeedbackFactor(totalScore, sourceFactors[rt.Source])
		newTopics = append(newTopics, &DiscoveredTopic{
			Title:             rt.Title,
			Source:            rt.Source,
			SourceURL:         rt.URL,
			Popularity:        rt.Popularity,
			Replies:           rt.Replies,
			Snippet:           rt.Snippet,
			ContentHash:       hash,
			Status:            "pending",
			PainScore:         painScore,
			TrendScore:        trendScore,
			FeasibilityScore:  feasibilityScore,
			MonetizationScore: monetizationScore,
			NoveltyScore:      noveltyScore,
			RecommendScore:    totalScore,
			BatchID:           batchID,
		})
	}

	if len(newTopics) == 0 {
		return nil, nil
	}

	if err := uc.repo.SaveTopics(ctx, newTopics); err != nil {
		return nil, fmt.Errorf("save topics: %w", err)
	}
	uc.log.Infof("[Discovery] Saved %d new topics", len(newTopics))
	return newTopics, nil
}

// aiSelectTopicsWithQuota AI精选（动态配额）
func (uc *DiscoveryUsecase) aiSelectTopicsWithQuota(ctx context.Context, topics []*DiscoveredTopic, existingIdeas []existingIdeaSummary, quota int) ([]SelectedTopic, error) {
	if quota <= 0 {
		quota = 1
	}
	if quota > 6 {
		quota = 6
	}

	// 复用已有的 aiSelectTopics，但修改上限为 quota
	selected, err := uc.aiSelectTopics(ctx, topics, existingIdeas)
	if err != nil {
		return nil, err
	}
	if len(selected) > quota {
		selected = selected[:quota]
	}
	return selected, nil
}

// startDebatesForSelected 为精选话题创建 Idea 并启动辩论
func (uc *DiscoveryUsecase) startDebatesForSelected(ctx context.Context, selected []SelectedTopic) int {
	debateCount := 0
	for _, sel := range selected {
		idea, err := uc.ideaRepo.Create(ctx, &Idea{
			Topic:  sel.Topic,
			Status: "pending",
		})
		if err != nil {
			uc.log.Warnf("[AutoDiscovery] Create idea failed for '%s': %v", sel.Topic, err)
			continue
		}

		if sel.DiscoveredTopicID > 0 {
			_ = uc.repo.SetTopicIdeaID(ctx, sel.DiscoveredTopicID, idea.ID)
		}

		uc.log.Infof("[AutoDiscovery] Created Idea #%d: %s (reason: %s)", idea.ID, sel.Topic, sel.Reason)

		if uc.debateUc != nil {
			eventCh := make(chan DebateEvent, 100)
			go func(ch <-chan DebateEvent, ideaID int64) {
				for range ch {
				}
				uc.log.Infof("[AutoDiscovery] Idea #%d debate finished", ideaID)
			}(eventCh, idea.ID)

			if err := uc.debateUc.StartDebate(context.Background(), idea.ID, eventCh); err != nil {
				uc.log.Warnf("[AutoDiscovery] Start debate for Idea #%d failed: %v", idea.ID, err)
				continue
			}
			debateCount++
			uc.log.Infof("[AutoDiscovery] Started debate for Idea #%d: %s", idea.ID, sel.Topic)
		}
	}
	return debateCount
}

// drainPendingIdeas 消化存量 pending ideas — 自动启动等待中的辩论
func (uc *DiscoveryUsecase) drainPendingIdeas(ctx context.Context) int {
	if uc.debateUc == nil {
		return 0
	}

	availableSlots := uc.debateUc.AvailableSlots()
	if availableSlots <= 0 {
		return 0
	}

	// 查找 pending ideas
	result, err := uc.ideaRepo.List(ctx, &IdeaListQuery{
		Page: 1, PageSize: availableSlots, Status: "pending", SortBy: "created_at", SortOrder: "asc",
	})
	if err != nil || result == nil || len(result.Items) == 0 {
		return 0
	}

	started := 0
	for _, idea := range result.Items {
		if uc.debateUc.AvailableSlots() <= 0 {
			break
		}

		eventCh := make(chan DebateEvent, 100)
		go func(ch <-chan DebateEvent, ideaID int64) {
			for range ch {
			}
			uc.log.Infof("[DrainPending] Idea #%d debate finished", ideaID)
		}(eventCh, idea.ID)

		if err := uc.debateUc.StartDebate(context.Background(), idea.ID, eventCh); err != nil {
			uc.log.Warnf("[DrainPending] Start debate for Idea #%d failed: %v", idea.ID, err)
			continue
		}
		started++
		uc.log.Infof("[DrainPending] Started debate for pending Idea #%d: %s", idea.ID, idea.Topic)
	}

	if started > 0 {
		uc.log.Infof("[DrainPending] 消化了 %d 个存量 pending ideas", started)
	}
	return started
}

// existingIdeaSummary 已有项目摘要（用于语义去重）
type existingIdeaSummary struct {
	ProductName string   `json:"product_name,omitempty"`
	Topic       string   `json:"topic"`
	OneLiner    string   `json:"one_liner,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Score       float64  `json:"score,omitempty"`
	Status      string   `json:"status"`
}

// SelectedTopic AI决策精选的话题
type SelectedTopic struct {
	Topic             string `json:"topic"`
	Reason            string `json:"reason"`
	DiscoveredTopicID int64  `json:"discovered_topic_id,omitempty"`
}

const (
	semanticDedupDismissThreshold = 0.85
	semanticDedupReviewThreshold  = 0.75
	sourceFeedbackLookbackDays    = 30
	sourceFeedbackMinSample       = 5
)


// aiSelectTopics AI统一决策：从所有发现的话题中精选最佳1-3个进行辩论（严格语义去重）
func (uc *DiscoveryUsecase) aiSelectTopics(ctx context.Context, topics []*DiscoveredTopic, existingIdeas []existingIdeaSummary) ([]SelectedTopic, error) {
	if len(topics) == 0 {
		return nil, nil
	}

	type topicInput struct {
		Index      int    `json:"index"`
		Title      string `json:"title"`
		Source     string `json:"source"`
		Popularity int    `json:"popularity"`
		Replies    int    `json:"replies"`
		Snippet    string `json:"snippet"`
	}

	inputs := make([]topicInput, len(topics))
	for i, t := range topics {
		inputs[i] = topicInput{
			Index:      i,
			Title:      t.Title,
			Source:     t.Source,
			Popularity: t.Popularity,
			Replies:    t.Replies,
			Snippet:    truncateText(t.Snippet, 200),
		}
	}

	inputJSON, _ := json.Marshal(inputs)

	// 构建已有项目的丰富摘要（用于语义去重）
	existingStr := ""
	if len(existingIdeas) > 0 {
		existingJSON, _ := json.Marshal(existingIdeas)
		existingStr = fmt.Sprintf("\n\n## 已有项目列表（共 %d 个，必须严格避免语义重复）\n```json\n%s\n```", len(existingIdeas), string(existingJSON))
	}

	// 读取约束标签
	constraintStr := ""
	constraintTags, _ := uc.repo.ListTagsByCategory(ctx, "constraint")
	if len(constraintTags) > 0 {
		var lines []string
		for _, t := range constraintTags {
			lines = append(lines, "- "+t.Tag)
		}
		constraintStr = fmt.Sprintf("\n\n## 核心约束条件（必须满足）\n%s", strings.Join(lines, "\n"))
	}

	selectorPrompt := `你是 AI 创业话题精选决策器。你的核心职责是从候选话题中精选出**真正有新意、与已有项目不重复**的创业方向。

## ⚠️ 最高优先级：严格语义去重
你会收到一份"已有项目列表"，包含每个项目的产品名、话题、一句话描述和标签。
你必须逐个对比候选话题与已有项目，判断是否存在**语义重复**：
- **同一赛道的类似产品** = 重复（如"AI写作助手"和"AI内容生成工具"）
- **换了个名字但本质相同** = 重复（如"AI声音克隆"和"AI语音合成"）  
- **同一目标用户的同类解决方案** = 重复（如"老年人健康监测"和"银发族智能健康管家"）
- **仅细分方向不同但核心技术和商业模式相同** = 重复

如果候选话题与任何已有项目存在语义重复，**直接排除，不要选它**。
宁可本轮一个都不选，也不要选出与已有项目重复的话题。

## 精选原则（在去重基础上）
1. 优先选择**具体的、可落地的**创业方向
2. 优先选择**有市场验证信号**的（高热度、多讨论、有融资新闻）
3. 话题间要有**多样性**（彼此之间也不能相似）
4. 如果原始话题不够具体，请**重新提炼**为更具体的创业方向
5. **严格过滤**不满足约束条件的话题
6. 数量：宁缺毋滥，通常精选 1-2 个，质量极高时最多 3 个

## 输出格式（严格JSON数组，不要其他文字）
[{"topic": "精选的具体创业方向", "reason": "选择理由（必须说明与已有项目的差异点）", "source_index": 0}]

如果本批候选全部与已有项目重复或质量不够，输出空数组 []。
source_index 是来源话题在输入列表中的索引号，如果是你重新提炼的话题就填 -1。`

	prompt := fmt.Sprintf("以下是从多个渠道发现的 %d 个话题候选：\n\n%s%s%s", len(inputs), string(inputJSON), constraintStr, existingStr)

	response, err := uc.llmClient.CallWithRole(ctx, "utility", selectorPrompt, []llm.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析 JSON
	cleaned := response
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	type selectionResult struct {
		Topic       string `json:"topic"`
		Reason      string `json:"reason"`
		SourceIndex int    `json:"source_index"`
	}

	var results []selectionResult
	if err := json.Unmarshal([]byte(cleaned), &results); err != nil {
		return nil, fmt.Errorf("parse AI selection: %w, raw: %s", err, truncateText(response, 300))
	}

	// 上限3个
	if len(results) > 3 {
		results = results[:3]
	}

	var selected []SelectedTopic
	for _, r := range results {
		sel := SelectedTopic{
			Topic:  r.Topic,
			Reason: r.Reason,
		}
		// 关联原始 discovered_topic ID
		if r.SourceIndex >= 0 && r.SourceIndex < len(topics) {
			sel.DiscoveredTopicID = topics[r.SourceIndex].ID
			// 更新话题分析
			_ = uc.repo.UpdateTopicAnalysis(ctx, topics[r.SourceIndex].ID, fmt.Sprintf("[已精选] %s", r.Reason), 9.0)
		}
		selected = append(selected, sel)
	}

	return selected, nil
}

// fallbackSelect 降级精选：直接取前N个热门话题
func (uc *DiscoveryUsecase) fallbackSelect(topics []*DiscoveredTopic, count int) []SelectedTopic {
	if len(topics) == 0 {
		return nil
	}
	if count > len(topics) {
		count = len(topics)
	}

	var selected []SelectedTopic
	for i := 0; i < count; i++ {
		selected = append(selected, SelectedTopic{
			Topic:             topics[i].Title,
			Reason:            "降级模式：直接选取",
			DiscoveredTopicID: topics[i].ID,
		})
	}
	return selected
}

// applySemanticDedupThreshold 在自动提交流程前执行语义去重阈值门禁
// 规则：
// 1) 相似度 > 0.85：自动过滤（标记 dismissed）
// 2) 相似度 0.75~0.85：进入人工复核队列（保留 pending，不进入自动辩论）
// 3) 相似度 < 0.75：进入自动辩论精选候选
func (uc *DiscoveryUsecase) applySemanticDedupThreshold(
	ctx context.Context,
	topics []*DiscoveredTopic,
	existingIdeas []existingIdeaSummary,
) (kept []*DiscoveredTopic, reviewCount int, dismissedCount int) {
	if len(topics) == 0 || len(existingIdeas) == 0 {
		return topics, 0, 0
	}

	existingTexts := make([]string, 0, len(existingIdeas))
	for _, idea := range existingIdeas {
		text := buildExistingIdeaSemanticText(idea)
		if text != "" {
			existingTexts = append(existingTexts, text)
		}
	}
	if len(existingTexts) == 0 {
		return topics, 0, 0
	}

	kept = make([]*DiscoveredTopic, 0, len(topics))
	for _, topic := range topics {
		candidateText := buildTopicSemanticText(topic)
		if candidateText == "" {
			kept = append(kept, topic)
			continue
		}

		maxSimilarity := 0.0
		for _, existingText := range existingTexts {
			similarity := semanticSimilarity(candidateText, existingText)
			if similarity > maxSimilarity {
				maxSimilarity = similarity
			}
		}

		switch {
		case maxSimilarity > semanticDedupDismissThreshold:
			dismissedCount++
			if err := uc.repo.UpdateTopicStatus(ctx, topic.ID, "dismissed"); err != nil {
				uc.log.Warnf("[AutoDiscovery] semantic dismiss failed for topic %d: %v", topic.ID, err)
			}
			uc.log.Infof("[AutoDiscovery] semantic dismiss topic=%d similarity=%.2f title=%s", topic.ID, maxSimilarity, truncateText(topic.Title, 80))
		case maxSimilarity >= semanticDedupReviewThreshold:
			reviewCount++
			uc.log.Infof("[AutoDiscovery] semantic review topic=%d similarity=%.2f title=%s", topic.ID, maxSimilarity, truncateText(topic.Title, 80))
		default:
			kept = append(kept, topic)
		}
	}

	return kept, reviewCount, dismissedCount
}

// analyzeTopics 用 AI 分析话题推荐度（用于手动发现流程）
func (uc *DiscoveryUsecase) analyzeTopics(ctx context.Context, topics []*DiscoveredTopic) {
	type topicInput struct {
		Index      int    `json:"index"`
		Title      string `json:"title"`
		Source     string `json:"source"`
		Popularity int    `json:"popularity"`
		Replies    int    `json:"replies"`
		Snippet    string `json:"snippet"`
	}

	inputs := make([]topicInput, len(topics))
	for i, t := range topics {
		inputs[i] = topicInput{
			Index:      i,
			Title:      t.Title,
			Source:     t.Source,
			Popularity: t.Popularity,
			Replies:    t.Replies,
			Snippet:    truncateText(t.Snippet, 200),
		}
	}

	inputJSON, err := json.Marshal(inputs)
	if err != nil {
		uc.log.Warnf("[Discovery] Marshal topics for analysis failed: %v", err)
		return
	}

	prompt := fmt.Sprintf("以下是从多个渠道收集到的 %d 个话题，请逐一分析并给出推荐评分和理由：\n\n%s", len(inputs), string(inputJSON))

	response, err := uc.llmClient.CallWithRole(ctx, "utility", llm.AgentTopicAnalyst.SystemPrompt, []llm.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		uc.log.Warnf("[Discovery] AI analysis failed: %v", err)
		return
	}

	type analysisResult struct {
		Index          int     `json:"index"`
		RecommendScore float64 `json:"recommend_score"`
		Recommendation string  `json:"recommendation"`
		SuggestedTopic string  `json:"suggested_topic"`
	}

	cleaned := response
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var results []analysisResult
	if err := json.Unmarshal([]byte(cleaned), &results); err != nil {
		uc.log.Warnf("[Discovery] Parse AI analysis failed: %v, raw: %s", err, truncateText(response, 300))
		return
	}

	for _, r := range results {
		if r.Index < 0 || r.Index >= len(topics) {
			continue
		}
		topic := topics[r.Index]
		ruleScore := clampScore(topic.RecommendScore)
		llmScore := clampScore(r.RecommendScore)
		hybridScore := clampScore(ruleScore*0.60 + llmScore*0.40)

		recommendation := fmt.Sprintf(
			"[混合评分] 规则 %.1f × 0.60 + LLM %.1f × 0.40 = %.1f\n[规则分项] 痛点 %.1f / 趋势 %.1f / 可行 %.1f / 变现 %.1f / 新颖 %.1f\n\n%s",
			ruleScore, llmScore, hybridScore,
			topic.PainScore, topic.TrendScore, topic.FeasibilityScore, topic.MonetizationScore, topic.NoveltyScore,
			r.Recommendation,
		)
		if r.SuggestedTopic != "" {
			recommendation = fmt.Sprintf("[建议话题] %s\n\n%s", r.SuggestedTopic, recommendation)
		}
		_ = uc.repo.UpdateTopicAnalysis(ctx, topic.ID, recommendation, hybridScore)
	}

	uc.log.Infof("[Discovery] AI analysis completed for %d topics", len(results))
}

// ---- Scheduler ----

// StartScheduler 启动全自动话题发现+辩论调度器
func (uc *DiscoveryUsecase) StartScheduler(ctx context.Context) {
	uc.mu.Lock()
	if uc.running {
		uc.mu.Unlock()
		return
	}
	uc.running = true
	uc.stopCh = make(chan struct{})
	uc.mu.Unlock()

	go func() {
		uc.log.Info("[Scheduler] ========== 全自动调度器已启动 ==========")

		// 启动后等10秒，恢复被中断的辩论
		select {
		case <-time.After(10 * time.Second):
		case <-uc.stopCh:
			return
		}
		uc.resumeStalledDebates()

		// 再等20秒后执行首次发现（给系统启动留时间）
		select {
		case <-time.After(20 * time.Second):
		case <-uc.stopCh:
			return
		}

		for {
			settings, _ := uc.GetSettings(context.Background())
			if settings == nil || !settings.DiscoveryEnabled {
				// 未启用，每分钟检查一次
				uc.log.Info("[Scheduler] 自动发现未启用，等待中...")
				select {
				case <-time.After(1 * time.Minute):
					continue
				case <-uc.stopCh:
					uc.log.Info("[Scheduler] 调度器已停止")
					return
				}
			}

			interval, err := time.ParseDuration(settings.DiscoveryInterval)
			if err != nil || interval < 10*time.Minute {
				interval = 40 * time.Minute // 默认40分钟
			}

			uc.log.Infof("[Scheduler] ⏰ 开始执行全自动发现+辩论...")

			// 执行全自动流程
			count, runErr := uc.RunAutoDiscoveryAndDebate(context.Background())
			if runErr != nil {
				uc.log.Warnf("[Scheduler] 全自动流程出错: %v", runErr)
			} else {
				uc.log.Infof("[Scheduler] ✅ 全自动流程完成，启动了 %d 场辩论", count)
			}

			// 自适应间隔：根据 pending 数量动态调整
			adaptiveInterval := uc.computeAdaptiveInterval(interval)
			uc.log.Infof("[Scheduler] 下次执行: %s 后", adaptiveInterval)
			select {
			case <-time.After(adaptiveInterval):
			case <-uc.stopCh:
				uc.log.Info("[Scheduler] 调度器已停止")
				return
			}
		}
	}()
}

// computeAdaptiveInterval 根据 pending 数量动态调整调度间隔
func (uc *DiscoveryUsecase) computeAdaptiveInterval(baseInterval time.Duration) time.Duration {
	pendingCount, _ := uc.repo.CountTopicsByStatus(context.Background(), "pending")

	switch {
	case pendingCount > int64(backpressureSkipThreshold):
		// pending 很多，缩短间隔加速消化（最低 10 分钟）
		short := 10 * time.Minute
		uc.log.Infof("[Scheduler] 自适应: pending=%d > %d，间隔缩短至 %s", pendingCount, backpressureSkipThreshold, short)
		return short
	case pendingCount > int64(backpressureHalfThreshold):
		// pending 中等，适度缩短
		medium := baseInterval * 3 / 4
		if medium < 15*time.Minute {
			medium = 15 * time.Minute
		}
		uc.log.Infof("[Scheduler] 自适应: pending=%d，间隔调整为 %s", pendingCount, medium)
		return medium
	case pendingCount < 20:
		// pending 很少，延长间隔节省资源
		long := baseInterval * 3 / 2
		if long > 90*time.Minute {
			long = 90 * time.Minute
		}
		uc.log.Infof("[Scheduler] 自适应: pending=%d < 20，间隔延长至 %s", pendingCount, long)
		return long
	default:
		return baseInterval
	}
}

// resumeStalledDebates 恢复被服务重启中断的辩论
func (uc *DiscoveryUsecase) resumeStalledDebates() {
	if uc.debateUc == nil {
		return
	}

	ctx := context.Background()
	// 查找所有状态为 debating 的 ideas（说明上次服务中断时正在辩论）
	result, err := uc.ideaRepo.List(ctx, &IdeaListQuery{
		Page: 1, PageSize: 20, Status: "debating", SortBy: "created_at", SortOrder: "desc",
	})
	if err != nil || result == nil || len(result.Items) == 0 {
		uc.log.Info("[Scheduler] 无需恢复中断的辩论")
		return
	}

	uc.log.Infof("[Scheduler] 发现 %d 个被中断的辩论，开始恢复...", len(result.Items))

	for _, idea := range result.Items {
		// 先将状态重置为 pending，否则 StartDebate 会拒绝（认为已在辩论中）
		_ = uc.ideaRepo.UpdateStatus(ctx, idea.ID, "pending")

		eventCh := make(chan DebateEvent, 100)
		go func(ch <-chan DebateEvent, id int64) {
			for range ch {
			}
			uc.log.Infof("[Scheduler] 恢复的辩论 Idea #%d 已完成", id)
		}(eventCh, idea.ID)

		if err := uc.debateUc.StartDebate(ctx, idea.ID, eventCh); err != nil {
			uc.log.Warnf("[Scheduler] 恢复辩论 Idea #%d 失败: %v", idea.ID, err)
			continue
		}
		uc.log.Infof("[Scheduler] ✅ 已恢复辩论 Idea #%d: %s", idea.ID, idea.Topic)
	}
}

// StopScheduler 停止定时调度
func (uc *DiscoveryUsecase) StopScheduler() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.running {
		close(uc.stopCh)
		uc.running = false
	}
}

// ---- Helpers ----

func contentHash(title, source string) string {
	h := sha256.Sum256([]byte(source + "|" + title))
	return fmt.Sprintf("%x", h[:16])
}

func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func extractSuggestedTopic(recommendation, fallback string) string {
	// 从推荐理由中提取 [建议话题] 后面的内容
	re := regexp.MustCompile(`\[建议话题\]\s*(.+?)[\n]`)
	matches := re.FindStringSubmatch(recommendation)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return fallback
}

func calculateRuleScores(rt *RawTopic) (pain, trend, feasibility, monetization, novelty, total float64) {
	text := strings.ToLower(rt.Title + " " + rt.Snippet)

	painKeywords := []string{"痛点", "吐槽", "抱怨", "求助", "避雷", "效率低", "麻烦", "难", "贵", "问题", "痛苦", "frustrat", "pain", "struggle"}
	monetizationKeywords := []string{"saas", "订阅", "收费", "变现", "客单价", "企业", "商家", "b2b", "降本", "增效", "roi", "pay", "pricing", "revenue"}
	feasibleNegativeKeywords := []string{"量子", "脑机", "核聚变", "纳米机器人", "太空殖民", "永生", "意识上传"}

	painHits := keywordHits(text, painKeywords)
	monetizationHits := keywordHits(text, monetizationKeywords)
	feasibleNegativeHits := keywordHits(text, feasibleNegativeKeywords)

	pain = clampScore(4.0 + float64(painHits)*1.2 + normalizeMetric(rt.Replies, 100)*0.3)
	trend = clampScore(normalizeMetric(rt.Popularity, 800)*0.7 + normalizeMetric(rt.Replies, 120)*0.3)
	feasibility = clampScore(7.0 - float64(feasibleNegativeHits)*2.5 + normalizeMetric(rt.Replies, 80)*0.2)
	monetization = clampScore(4.5 + float64(monetizationHits)*1.3 + normalizeMetric(rt.Popularity, 1000)*0.2)
	novelty = clampScore(5.5 + sourceNoveltyBoost(rt.Source) + noveltyPatternBoost(rt.Title))

	total = clampScore(
		pain*0.30 +
			trend*0.20 +
			feasibility*0.20 +
			monetization*0.20 +
			novelty*0.10,
	)
	return pain, trend, feasibility, monetization, novelty, total
}

func keywordHits(text string, keywords []string) int {
	hits := 0
	for _, kw := range keywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			hits++
		}
	}
	return hits
}

func normalizeMetric(value, cap int) float64 {
	if value <= 0 || cap <= 0 {
		return 0
	}
	if value >= cap {
		return 10
	}
	return float64(value) * 10 / float64(cap)
}

func sourceNoveltyBoost(source string) float64 {
	switch source {
	case "llm_creative", "collision":
		return 2.0
	case "bing_trend_en":
		return 1.0
	default:
		return 0.4
	}
}

func noveltyPatternBoost(title string) float64 {
	if strings.Contains(title, "×") || strings.Contains(title, " x ") {
		return 1.2
	}
	return 0
}

func (uc *DiscoveryUsecase) buildSourceFeedbackFactors(ctx context.Context) map[string]float64 {
	result := make(map[string]float64)
	stats, err := uc.repo.ListSourceFeedbackStats(ctx, sourceFeedbackLookbackDays)
	if err != nil {
		uc.log.Warnf("[Discovery] load source feedback stats failed: %v", err)
		return result
	}

	for _, stat := range stats {
		if stat == nil || stat.Source == "" || stat.SampleSize < sourceFeedbackMinSample {
			continue
		}
		// 反哺因子范围：0.85 ~ 1.15
		factor := 0.85 + stat.SuccessRate*0.30
		if factor < 0.85 {
			factor = 0.85
		}
		if factor > 1.15 {
			factor = 1.15
		}
		result[stat.Source] = factor
	}
	return result
}

func applySourceFeedbackFactor(score float64, factor float64) float64 {
	if factor <= 0 {
		return clampScore(score)
	}
	return clampScore(score * factor)
}

func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 10 {
		return 10
	}
	return score
}

func buildExistingIdeaSemanticText(idea existingIdeaSummary) string {
	return normalizeSemanticText(strings.Join([]string{
		idea.ProductName,
		idea.Topic,
		idea.OneLiner,
		strings.Join(idea.Tags, " "),
	}, " "))
}

func buildTopicSemanticText(topic *DiscoveredTopic) string {
	if topic == nil {
		return ""
	}
	return normalizeSemanticText(strings.Join([]string{
		topic.Title,
		topic.Snippet,
	}, " "))
}

func semanticSimilarity(left, right string) float64 {
	leftSet := semanticTokenSet(left)
	rightSet := semanticTokenSet(right)
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}

	intersection := 0
	for token := range leftSet {
		if _, exists := rightSet[token]; exists {
			intersection++
		}
	}
	union := len(leftSet) + len(rightSet) - intersection
	if union <= 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func normalizeSemanticText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	invalidChars := regexp.MustCompile(`[^\p{Han}a-z0-9]+`)
	text = invalidChars.ReplaceAllString(text, " ")
	return strings.TrimSpace(strings.Join(strings.Fields(text), " "))
}

func semanticTokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	if text == "" {
		return set
	}

	for _, token := range strings.Fields(text) {
		if token != "" {
			set["w:"+token] = struct{}{}
		}
	}

	compact := strings.ReplaceAll(text, " ", "")
	runes := []rune(compact)
	if len(runes) == 1 {
		set["g:"+string(runes[0])] = struct{}{}
		return set
	}

	for index := 0; index < len(runes)-1; index++ {
		bigram := string(runes[index : index+2])
		set["g:"+bigram] = struct{}{}
	}
	return set
}
