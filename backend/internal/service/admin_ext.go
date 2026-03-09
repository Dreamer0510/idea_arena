package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"idea_arena/internal/biz"
	"idea_arena/internal/conf"
	"idea_arena/internal/data"
	"idea_arena/internal/pkg/llm"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AdminExtService 管理后台扩展服务（stats/models/agents/users/posts）
type AdminExtService struct {
	ideaRepo      biz.IdeaRepo
	userUc        *biz.UserUsecase
	llmConf       *conf.LLM
	llmClient     *llm.Client
	postRepo      *data.SocialPostRepo
	agentRepo     *data.AgentConfigRepo
	log           *log.Helper
}

// NewAdminExtService 创建扩展管理服务
func NewAdminExtService(
	ideaRepo biz.IdeaRepo,
	userUc *biz.UserUsecase,
	llmConf *conf.LLM,
	llmClient *llm.Client,
	postRepo *data.SocialPostRepo,
	agentRepo *data.AgentConfigRepo,
	logger log.Logger,
) *AdminExtService {
	return &AdminExtService{
		ideaRepo:  ideaRepo,
		userUc:    userUc,
		llmConf:   llmConf,
		llmClient: llmClient,
		postRepo:  postRepo,
		agentRepo: agentRepo,
		log:       log.NewHelper(logger),
	}
}

// RegisterHTTPRoutes 注册扩展管理路由
func (s *AdminExtService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	// Stats
	r.GET("/api/v1/admin/stats/overview", s.GetStatsOverview)

	// Models Config
	r.GET("/api/v1/admin/models/config", s.GetModelsConfig)
	r.PUT("/api/v1/admin/models/config", s.SaveModelsConfig)
	r.POST("/api/v1/admin/models/test", s.TestModelConnection)

	// Agents
	r.GET("/api/v1/admin/agents", s.ListAgents)
	r.PUT("/api/v1/admin/agents", s.SaveAgents)

	// Users
	r.GET("/api/v1/admin/users", s.ListUsers)
	r.POST("/api/v1/admin/users", s.CreateUser)
	r.PUT("/api/v1/admin/users/{id}", s.UpdateUser)
	r.DELETE("/api/v1/admin/users/{id}", s.DeleteUser)

	// Posts
	r.GET("/api/v1/admin/posts", s.ListPosts)
	r.POST("/api/v1/admin/posts/generate", s.GeneratePost)
	r.PUT("/api/v1/admin/posts/{id}", s.UpdatePost)
	r.DELETE("/api/v1/admin/posts/{id}", s.DeletePost)
}

// ========== Stats ==========

func (s *AdminExtService) GetStatsOverview(ctx kratoshttp.Context) error {
	pending, _ := s.ideaRepo.CountByStatus(ctx, "pending")
	debating, _ := s.ideaRepo.CountByStatus(ctx, "debating")
	graduated, _ := s.ideaRepo.CountByStatus(ctx, "graduated")
	failed, _ := s.ideaRepo.CountByStatus(ctx, "failed")
	promising, _ := s.ideaRepo.CountByStatus(ctx, "promising")

	total := pending + debating + graduated + failed + promising
	completed := graduated + failed
	passRate := int64(0)
	if completed > 0 {
		passRate = graduated * 100 / completed
	}

	// 计算平均分和今日产出
	avgScore := 0.0
	todayCount := int64(0)
	result, err := s.ideaRepo.List(ctx, &biz.IdeaListQuery{Page: 1, PageSize: 10000})
	if err == nil && len(result.Items) > 0 {
		var scoreSum float64
		var scoreCount int
		today := time.Now().Format("2006-01-02")
		for _, idea := range result.Items {
			if idea.ScoreOverall > 0 {
				scoreSum += idea.ScoreOverall
				scoreCount++
			}
			if idea.CreatedAt.Format("2006-01-02") == today {
				todayCount++
			}
		}
		if scoreCount > 0 {
			avgScore = float64(int(scoreSum/float64(scoreCount)*10)) / 10
		}
	}

	return ctx.Result(http.StatusOK, map[string]interface{}{
		"total_ideas":     total,
		"pending_count":   pending,
		"debating_count":  debating,
		"graduated_count": graduated,
		"failed_count":    failed,
		"avg_score":       avgScore,
		"today_count":     todayCount,
		"pass_rate":       passRate,
	})
}

// ========== Models Config ==========

type modelsConfigResponse struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	DefaultModel string `json:"default_model"`
	MaxTokens    int    `json:"max_tokens"`
	TimeoutSec   int    `json:"timeout_sec"`
	MaxRetries   int    `json:"max_retries"`
	Models       struct {
		Proposer string `json:"proposer"`
		Opponent string `json:"opponent"`
		Judge    string `json:"judge"`
		Utility  string `json:"utility"`
	} `json:"models"`
}

func (s *AdminExtService) GetModelsConfig(ctx kratoshttp.Context) error {
	resp := modelsConfigResponse{
		BaseURL:      s.llmConf.BaseURL,
		APIKey:       maskAPIKey(s.llmConf.APIKey),
		DefaultModel: s.llmConf.DefaultModel,
		MaxTokens:    s.llmConf.GetMaxTokens(),
		TimeoutSec:   s.llmConf.TimeoutSec,
		MaxRetries:   s.llmConf.GetMaxRetries(),
	}
	if s.llmConf.Models != nil {
		resp.Models.Proposer = s.llmConf.Models.Proposer
		resp.Models.Opponent = s.llmConf.Models.Opponent
		resp.Models.Judge = s.llmConf.Models.Judge
		resp.Models.Utility = s.llmConf.Models.Utility
	}
	return ctx.Result(http.StatusOK, resp)
}

func (s *AdminExtService) SaveModelsConfig(ctx kratoshttp.Context) error {
	var req struct {
		BaseURL      string `json:"base_url"`
		APIKey       string `json:"api_key"`
		DefaultModel string `json:"default_model"`
		MaxTokens    int    `json:"max_tokens"`
		TimeoutSec   int    `json:"timeout_sec"`
		MaxRetries   int    `json:"max_retries"`
		Models       struct {
			Proposer string `json:"proposer"`
			Opponent string `json:"opponent"`
			Judge    string `json:"judge"`
			Utility  string `json:"utility"`
		} `json:"models"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// 热更新配置（内存级别）
	if req.BaseURL != "" {
		s.llmConf.BaseURL = req.BaseURL
	}
	if req.APIKey != "" && !strings.HasPrefix(req.APIKey, "sk-...") {
		s.llmConf.APIKey = req.APIKey
	}
	if req.DefaultModel != "" {
		s.llmConf.DefaultModel = req.DefaultModel
	}
	if req.MaxTokens > 0 {
		s.llmConf.MaxTokens = req.MaxTokens
	}
	if req.TimeoutSec > 0 {
		s.llmConf.TimeoutSec = req.TimeoutSec
	}
	if req.MaxRetries > 0 {
		s.llmConf.MaxRetries = req.MaxRetries
	}

	if s.llmConf.Models == nil {
		s.llmConf.Models = &conf.LLMModels{}
	}
	s.llmConf.Models.Proposer = req.Models.Proposer
	s.llmConf.Models.Opponent = req.Models.Opponent
	s.llmConf.Models.Judge = req.Models.Judge
	s.llmConf.Models.Utility = req.Models.Utility

	s.log.Infof("[Admin] Models config updated: base_url=%s, default=%s, proposer=%s, opponent=%s, judge=%s, utility=%s",
		s.llmConf.BaseURL, s.llmConf.DefaultModel,
		s.llmConf.Models.Proposer, s.llmConf.Models.Opponent,
		s.llmConf.Models.Judge, s.llmConf.Models.Utility)

	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AdminExtService) TestModelConnection(ctx kratoshttp.Context) error {
	result, err := s.llmClient.Call(context.Background(), "你是一个测试助手。", []llm.Message{
		{Role: "user", Content: "请回复 'OK' 两个字母，不要其他内容。"},
	}, &llm.CallOptions{MaxTokens: 10, Temperature: 0})
	if err != nil {
		return ctx.Result(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success":  true,
		"response": result,
	})
}

// ========== Agents ==========

func (s *AdminExtService) ListAgents(ctx kratoshttp.Context) error {
	agents, err := s.agentRepo.List(ctx)
	if err != nil || len(agents) == 0 {
		// 返回默认 Agent 列表
		defaults := getDefaultAgents()
		agents = defaults
	}
	// 附加默认提示词
	promptMap := defaultPromptMap()
	items := make([]map[string]interface{}, len(agents))
	for i, a := range agents {
		items[i] = map[string]interface{}{
			"id":             a.ID,
			"name":           a.Name,
			"emoji":          a.Emoji,
			"role":           a.Role,
			"system_prompt":  a.SystemPrompt,
			"temperature":    a.Temperature,
			"provider_id":    a.ProviderID,
			"model_name":     a.ModelName,
			"enabled":        a.Enabled,
			"default_prompt": promptMap[a.ID],
		}
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": items})
}

func (s *AdminExtService) SaveAgents(ctx kratoshttp.Context) error {
	var req struct {
		Items []*data.AgentConfig `json:"items"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.agentRepo.BatchUpsert(ctx, req.Items); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ========== Users ==========

func (s *AdminExtService) ListUsers(ctx kratoshttp.Context) error {
	users, err := s.userUc.List(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	// 不返回密码
	items := make([]map[string]interface{}, len(users))
	for i, u := range users {
		items[i] = map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"role":       u.Role,
			"created_at": u.CreatedAt,
		}
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": items})
}

func (s *AdminExtService) CreateUser(ctx kratoshttp.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := ctx.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "username and password required"})
	}
	if req.Role == "" {
		req.Role = "user"
	}
	user, err := s.userUc.CreateUser(ctx, req.Username, req.Password, req.Role)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (s *AdminExtService) UpdateUser(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := ctx.Bind(&req); err != nil || req.Role == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "role is required"})
	}
	if err := s.userUc.UpdateRole(ctx, id, req.Role); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AdminExtService) DeleteUser(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if id == 1 {
		return ctx.Result(http.StatusForbidden, map[string]string{"error": "cannot delete default admin"})
	}
	if err := s.userUc.DeleteUser(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ========== Posts ==========

func (s *AdminExtService) ListPosts(ctx kratoshttp.Context) error {
	status := ctx.Request().URL.Query().Get("status")
	posts, err := s.postRepo.List(ctx, status)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": posts})
}

func (s *AdminExtService) GeneratePost(ctx kratoshttp.Context) error {
	var req struct {
		IdeaID   int64   `json:"idea_id"`
		IdeaIDs  []int64 `json:"idea_ids"`
		Platform string  `json:"platform"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	// 支持单个 idea_id 或多个 idea_ids
	if req.IdeaID > 0 && len(req.IdeaIDs) == 0 {
		req.IdeaIDs = []int64{req.IdeaID}
	}
	if len(req.IdeaIDs) == 0 {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "idea_id or idea_ids required"})
	}
	if req.Platform == "" {
		req.Platform = "xiaohongshu"
	}

	// 收集 idea 信息
	var ideaInfos []string
	for _, id := range req.IdeaIDs {
		idea, err := s.ideaRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		info := fmt.Sprintf("【%s】(ID:%d) %s", idea.ProductName, idea.ID, idea.OneLiner)
		if idea.FinalReport != "" {
			// 截取前 500 字
			report := idea.FinalReport
			if len(report) > 500 {
				report = report[:500] + "..."
			}
			info += "\n报告摘要: " + report
		}
		ideaInfos = append(ideaInfos, info)
	}

	if len(ideaInfos) == 0 {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "no valid ideas found"})
	}

	platformGuide := getPlatformGuide(req.Platform)
	prompt := fmt.Sprintf(`基于以下创业点子信息，为 %s 平台生成一篇推文。

%s

## 创业点子信息：
%s

请直接输出推文内容，格式：
第一行：标题
空一行
正文内容`, req.Platform, platformGuide, strings.Join(ideaInfos, "\n\n"))

	content, err := s.llmClient.Call(context.Background(),
		"你是一个专业的社交媒体文案写手，擅长将技术创业内容转化为引人入胜的社交媒体帖子。",
		[]llm.Message{{Role: "user", Content: prompt}},
		&llm.CallOptions{Temperature: 0.9},
	)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": "AI generation failed: " + err.Error()})
	}

	// 解析标题和内容
	title, body := parsePostContent(content)

	// 保存到数据库
	post, err := s.postRepo.Create(ctx, &data.SocialPost{
		IdeaID:   req.IdeaIDs[0],
		Platform: req.Platform,
		Title:    title,
		Content:  body,
	})
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.Result(http.StatusOK, post)
}

func (s *AdminExtService) UpdatePost(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.postRepo.Update(ctx, &data.SocialPost{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
		Status:  req.Status,
	}); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AdminExtService) DeletePost(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := s.postRepo.Delete(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ========== Helpers ==========

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "sk-..."
	}
	return key[:3] + "..." + key[len(key)-4:]
}

func defaultPromptMap() map[string]string {
	agents := []*llm.Agent{
		llm.AgentProposer,
		llm.AgentOpponent,
		llm.AgentReferee,
		llm.AgentJudge,
		llm.AgentMetaExtractor,
		llm.AgentDebateSummarizer,
		llm.AgentTopicStrategist,
	}
	m := make(map[string]string, len(agents))
	for _, a := range agents {
		m[a.ID] = a.SystemPrompt
	}
	return m
}

func getDefaultAgents() []*data.AgentConfig {
	agents := []*llm.Agent{
		llm.AgentProposer,
		llm.AgentOpponent,
		llm.AgentReferee,
		llm.AgentJudge,
		llm.AgentMetaExtractor,
		llm.AgentDebateSummarizer,
		llm.AgentTopicStrategist,
	}
	result := make([]*data.AgentConfig, len(agents))
	for i, a := range agents {
		result[i] = &data.AgentConfig{
			ID:           a.ID,
			Name:         a.Name,
			Emoji:        a.Emoji,
			Role:         a.Role,
			SystemPrompt: "",
			Temperature:  0.8,
			Enabled:      true,
		}
	}
	return result
}

func getPlatformGuide(platform string) string {
	switch platform {
	case "xiaohongshu", "小红书":
		return `## 小红书风格指南：
- 标题用 emoji 开头，吸引眼球，15-25字
- 正文分段清晰，每段 2-3 句
- 多用 emoji 点缀（但不过度）
- 口语化、亲切感，像和朋友聊天
- 结尾加话题标签 #AI创业 #科技创新 等
- 总字数 300-600 字`
	case "weixin", "wechat", "公众号":
		return `## 公众号风格指南：
- 标题有吸引力，20-35字
- 正文结构清晰，有小标题
- 专业但不晦涩，有深度分析
- 可适当使用数据和案例
- 结尾有总结和引导关注
- 总字数 800-1500 字`
	case "twitter":
		return `## Twitter/X 风格指南：
- 简洁有力，每条推文不超过280字
- 可以用线程形式展开
- 英文为主，关键概念中英对照
- 用 hashtag 增加曝光
- 直接切入重点，不要客套`
	default:
		return `## 通用风格指南：
- 标题吸引人，正文清晰
- 语言通俗易懂
- 突出创新点和商业价值
- 300-800 字`
	}
}

func parsePostContent(content string) (title, body string) {
	content = strings.TrimSpace(content)
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) >= 2 {
		title = strings.TrimSpace(lines[0])
		body = strings.TrimSpace(lines[1])
	} else {
		title = content
		if len(title) > 100 {
			title = title[:100]
		}
		body = content
	}
	// 去掉标题前缀如 "标题：" "Title:"
	for _, prefix := range []string{"标题：", "标题:", "Title:", "# "} {
		title = strings.TrimPrefix(title, prefix)
	}
	return
}
