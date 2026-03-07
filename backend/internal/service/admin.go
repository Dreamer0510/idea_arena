package service

import (
	"context"
	"net/http"
	"strconv"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AdminService 管理后台服务
type AdminService struct {
	discoveryUc *biz.DiscoveryUsecase
	debateUc    *biz.DebateUsecase
	log         *log.Helper
}

// NewAdminService 创建管理后台服务
func NewAdminService(discoveryUc *biz.DiscoveryUsecase, debateUc *biz.DebateUsecase, logger log.Logger) *AdminService {
	return &AdminService{
		discoveryUc: discoveryUc,
		debateUc:    debateUc,
		log:         log.NewHelper(logger),
	}
}

// RegisterHTTPRoutes 注册管理后台路由
func (s *AdminService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	// Settings
	r.GET("/api/v1/admin/settings", s.GetSettings)
	r.PUT("/api/v1/admin/settings", s.SaveSettings)

	// Keywords
	r.GET("/api/v1/admin/keywords", s.ListKeywords)
	r.POST("/api/v1/admin/keywords", s.CreateKeyword)
	r.PUT("/api/v1/admin/keywords/{id}", s.UpdateKeyword)
	r.DELETE("/api/v1/admin/keywords/{id}", s.DeleteKeyword)

	// Plugins
	r.GET("/api/v1/admin/plugins", s.ListPlugins)
	r.PUT("/api/v1/admin/plugins/{name}/toggle", s.TogglePlugin)

	// Tags
	r.GET("/api/v1/admin/tags", s.ListTags)
	r.POST("/api/v1/admin/tags", s.CreateTag)
	r.PUT("/api/v1/admin/tags/{id}", s.UpdateTag)
	r.DELETE("/api/v1/admin/tags/{id}", s.DeleteTag)

	// Discovery
	r.POST("/api/v1/admin/discovery/run", s.RunDiscovery)
	r.POST("/api/v1/admin/discovery/auto", s.RunAutoDiscovery)
	r.GET("/api/v1/admin/topics", s.ListTopics)
	r.POST("/api/v1/admin/topics/{id}/submit", s.SubmitTopic)
	r.POST("/api/v1/admin/topics/{id}/dismiss", s.DismissTopic)
}

// ---- Settings ----

func (s *AdminService) GetSettings(ctx kratoshttp.Context) error {
	settings, err := s.discoveryUc.GetSettings(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, settings)
}

func (s *AdminService) SaveSettings(ctx kratoshttp.Context) error {
	var req biz.SystemSettings
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.discoveryUc.SaveSettings(ctx, &req); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ---- Keywords ----

func (s *AdminService) ListKeywords(ctx kratoshttp.Context) error {
	keywords, err := s.discoveryUc.ListKeywords(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": keywords})
}

func (s *AdminService) CreateKeyword(ctx kratoshttp.Context) error {
	var req struct {
		Keyword string `json:"keyword"`
	}
	if err := ctx.Bind(&req); err != nil || req.Keyword == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "keyword is required"})
	}
	kw, err := s.discoveryUc.CreateKeyword(ctx, req.Keyword)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusCreated, kw)
}

func (s *AdminService) UpdateKeyword(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req struct {
		Keyword string `json:"keyword"`
		Enabled bool   `json:"enabled"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.discoveryUc.UpdateKeyword(ctx, id, req.Keyword, req.Enabled); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AdminService) DeleteKeyword(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := s.discoveryUc.DeleteKeyword(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ---- Plugins ----

func (s *AdminService) ListPlugins(ctx kratoshttp.Context) error {
	plugins, err := s.discoveryUc.ListPlugins(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": plugins})
}

func (s *AdminService) TogglePlugin(ctx kratoshttp.Context) error {
	name := ctx.Vars().Get("name")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.discoveryUc.TogglePlugin(ctx, name, req.Enabled); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ---- Tags ----

func (s *AdminService) ListTags(ctx kratoshttp.Context) error {
	tags, err := s.discoveryUc.ListTags(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": tags})
}

func (s *AdminService) CreateTag(ctx kratoshttp.Context) error {
	var req struct {
		Tag      string `json:"tag"`
		Category string `json:"category"`
	}
	if err := ctx.Bind(&req); err != nil || req.Tag == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "tag is required"})
	}
	tag, err := s.discoveryUc.CreateTag(ctx, req.Tag, req.Category)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusCreated, tag)
}

func (s *AdminService) UpdateTag(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req struct {
		Tag      string `json:"tag"`
		Category string `json:"category"`
		Enabled  bool   `json:"enabled"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.discoveryUc.UpdateTag(ctx, id, req.Tag, req.Category, req.Enabled); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AdminService) DeleteTag(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := s.discoveryUc.DeleteTag(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ---- Discovery ----

func (s *AdminService) RunDiscovery(ctx kratoshttp.Context) error {
	// 使用后台 context 避免 HTTP 请求超时
	count, err := s.discoveryUc.RunDiscovery(context.Background())
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success":    true,
		"new_topics": count,
	})
}

func (s *AdminService) RunAutoDiscovery(ctx kratoshttp.Context) error {
	// 全自动流程是长时间运行的，使用后台 context 异步执行
	go func() {
		count, err := s.discoveryUc.RunAutoDiscoveryAndDebate(context.Background())
		if err != nil {
			s.log.Errorf("[Admin] Auto discovery failed: %v", err)
		} else {
			s.log.Infof("[Admin] Auto discovery completed: %d debates started", count)
		}
	}()
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "全自动发现+辩论已在后台启动",
	})
}

func (s *AdminService) ListTopics(ctx kratoshttp.Context) error {
	r := ctx.Request()
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	topics, total, err := s.discoveryUc.ListTopics(ctx, status, page, pageSize)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"items": topics,
		"total": total,
		"page":  page,
	})
}

func (s *AdminService) SubmitTopic(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	idea, err := s.discoveryUc.SubmitToDebate(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 自动启动辩论（异步，不阻塞响应）
	eventCh := make(chan biz.DebateEvent, 100)
	go func() {
		if startErr := s.debateUc.StartDebate(context.Background(), idea.ID, eventCh); startErr != nil {
			s.log.Errorf("[Admin] Auto-start debate for idea %d failed: %v", idea.ID, startErr)
			return
		}
		// 消费事件（丢弃，前端通过 SSE 获取）
		for range eventCh {
		}
	}()

	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success": true,
		"idea_id": idea.ID,
	})
}

func (s *AdminService) DismissTopic(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := s.discoveryUc.DismissTopic(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}
