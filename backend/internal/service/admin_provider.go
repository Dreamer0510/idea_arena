package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"idea_arena/internal/data"
	"idea_arena/internal/pkg/llm"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AdminProviderService 服务商管理服务
type AdminProviderService struct {
	providerRepo    *data.ProviderRepo
	providerManager *llm.ProviderManager
	log             *log.Helper
}

// NewAdminProviderService 创建服务商管理服务
func NewAdminProviderService(
	providerRepo *data.ProviderRepo,
	providerManager *llm.ProviderManager,
	logger log.Logger,
) *AdminProviderService {
	return &AdminProviderService{
		providerRepo:    providerRepo,
		providerManager: providerManager,
		log:             log.NewHelper(logger),
	}
}

// RegisterHTTPRoutes 注册服务商管理路由
func (s *AdminProviderService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.GET("/api/v1/admin/providers", s.ListProviders)
	r.POST("/api/v1/admin/providers", s.CreateProvider)
	r.PUT("/api/v1/admin/providers/{id}", s.UpdateProvider)
	r.DELETE("/api/v1/admin/providers/{id}", s.DeleteProvider)
	r.POST("/api/v1/admin/providers/{id}/test", s.TestProvider)
	r.POST("/api/v1/admin/providers/{id}/models", s.FetchProviderModels)
	r.GET("/api/v1/admin/providers/{id}/models", s.GetProviderModels)
	// 测试未保存的服务商（用于创建前测试）
	r.POST("/api/v1/admin/providers/test", s.TestNewProvider)
}

// ========== List ==========

func (s *AdminProviderService) ListProviders(ctx kratoshttp.Context) error {
	providers, err := s.providerRepo.List(ctx)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	// mask API keys
	items := make([]map[string]interface{}, len(providers))
	for i, p := range providers {
		items[i] = map[string]interface{}{
			"id":         p.ID,
			"name":       p.Name,
			"base_url":   p.BaseURL,
			"api_key":    maskProviderKey(p.APIKey),
			"models_url": p.ModelsURL,
			"is_default": p.IsDefault,
			"enabled":    p.Enabled,
		}
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": items})
}

// ========== Create ==========

func (s *AdminProviderService) CreateProvider(ctx kratoshttp.Context) error {
	var req struct {
		Name      string `json:"name"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		ModelsURL string `json:"models_url"`
		IsDefault bool   `json:"is_default"`
	}
	if err := ctx.Bind(&req); err != nil || req.Name == "" || req.BaseURL == "" || req.APIKey == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "name, base_url, api_key required"})
	}

	provider, err := s.providerRepo.Create(ctx, &data.LLMProvider{
		Name:      req.Name,
		BaseURL:   strings.TrimSuffix(req.BaseURL, "/"),
		APIKey:    req.APIKey,
		ModelsURL: strings.TrimSuffix(req.ModelsURL, "/"),
		IsDefault: req.IsDefault,
		Enabled:   true,
	})
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 注册到 ProviderManager
	s.providerManager.RegisterProvider(llm.ProviderInfo{
		ID:      provider.ID,
		Name:    provider.Name,
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
	})

	s.log.Infof("[Admin] Provider created: id=%d name=%s", provider.ID, provider.Name)
	return ctx.Result(http.StatusCreated, map[string]interface{}{
		"id":   provider.ID,
		"name": provider.Name,
	})
}

// ========== Update ==========

func (s *AdminProviderService) UpdateProvider(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	existing, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "provider not found"})
	}

	var req struct {
		Name      string `json:"name"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		ModelsURL string `json:"models_url"`
		IsDefault bool   `json:"is_default"`
		Enabled   bool   `json:"enabled"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.BaseURL != "" {
		existing.BaseURL = strings.TrimSuffix(req.BaseURL, "/")
	}
	if req.APIKey != "" && !strings.HasPrefix(req.APIKey, "sk-...") && !strings.Contains(req.APIKey, "••") {
		existing.APIKey = req.APIKey
	}
	existing.ModelsURL = strings.TrimSuffix(req.ModelsURL, "/")
	existing.IsDefault = req.IsDefault
	existing.Enabled = req.Enabled

	if err := s.providerRepo.Update(ctx, existing); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 更新 ProviderManager
	if existing.Enabled {
		s.providerManager.RegisterProvider(llm.ProviderInfo{
			ID:      existing.ID,
			Name:    existing.Name,
			BaseURL: existing.BaseURL,
			APIKey:  existing.APIKey,
		})
	} else {
		s.providerManager.RemoveProvider(existing.ID)
	}

	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ========== Delete ==========

func (s *AdminProviderService) DeleteProvider(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := s.providerRepo.Delete(ctx, id); err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	s.providerManager.RemoveProvider(id)
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

// ========== Test ==========

func (s *AdminProviderService) TestProvider(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "provider not found"})
	}

	var req struct {
		Model string `json:"model"`
	}
	_ = ctx.Bind(&req)

	result, testErr := s.providerManager.TestProvider(context.Background(), llm.ProviderInfo{
		ID:      provider.ID,
		Name:    provider.Name,
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
	}, req.Model)

	if testErr != nil {
		return ctx.Result(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   testErr.Error(),
		})
	}

	// 如果探测到的 URL 与存储的不同，回写到数据库
	if result.WorkingURL != "" && result.WorkingURL != provider.BaseURL {
		provider.BaseURL = result.WorkingURL
		_ = s.providerRepo.Update(ctx, provider)
		// 更新 ProviderManager 中的注册信息
		s.providerManager.RegisterProvider(llm.ProviderInfo{
			ID: provider.ID, Name: provider.Name,
			BaseURL: result.WorkingURL, APIKey: provider.APIKey,
		})
		s.log.Infof("[Admin] Provider %d URL auto-corrected: %s -> %s", provider.ID, provider.BaseURL, result.WorkingURL)
	}

	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success":     true,
		"response":    result.Response,
		"working_url": result.WorkingURL,
	})
}

func (s *AdminProviderService) TestNewProvider(ctx kratoshttp.Context) error {
	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Model   string `json:"model"`
	}
	if err := ctx.Bind(&req); err != nil || req.BaseURL == "" || req.APIKey == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "base_url and api_key required"})
	}

	result, testErr := s.providerManager.TestProvider(context.Background(), llm.ProviderInfo{
		BaseURL: strings.TrimSuffix(req.BaseURL, "/"),
		APIKey:  req.APIKey,
	}, req.Model)

	if testErr != nil {
		return ctx.Result(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   testErr.Error(),
		})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success":     true,
		"response":    result.Response,
		"working_url": result.WorkingURL,
	})
}

// ========== Models Detection ==========

func (s *AdminProviderService) FetchProviderModels(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "provider not found"})
	}

	models, fetchErr := s.providerManager.FetchModels(context.Background(), llm.ProviderInfo{
		ID:      provider.ID,
		Name:    provider.Name,
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
	}, provider.ModelsURL)
	if fetchErr != nil {
		return ctx.Result(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fetchErr.Error(),
		})
	}

	// 过滤出聊天模型（排除 image/video/audio/embedding 等）
	chatModels := filterChatModels(models)

	// 保存到缓存
	modelInfos := make([]data.ProviderModelInfo, len(chatModels))
	for i, m := range chatModels {
		modelInfos[i] = data.ProviderModelInfo{ModelID: m.ID, OwnedBy: m.OwnedBy}
	}
	if err := s.providerRepo.SaveModels(ctx, id, modelInfos); err != nil {
		s.log.Warnf("Failed to save model cache: %v", err)
	}

	s.log.Infof("[Admin] Fetched %d models (filtered to %d chat models) for provider %d", len(models), len(chatModels), id)
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"success":     true,
		"total":       len(models),
		"chat_models": len(chatModels),
		"items":       modelInfos,
	})
}

func (s *AdminProviderService) GetProviderModels(ctx kratoshttp.Context) error {
	id, err := strconv.ParseInt(ctx.Vars().Get("id"), 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	models, err := s.providerRepo.GetModels(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"items": models})
}

// ========== Helpers ==========

func maskProviderKey(key string) string {
	if len(key) <= 8 {
		return "sk-..."
	}
	return key[:3] + "..." + key[len(key)-4:]
}

func filterChatModels(models []llm.ModelInfo) []llm.ModelInfo {
	excludeKeywords := []string{
		"flux", "dall-e", "embed", "tts", "whisper", "image", "video",
		"rembg", "animation", "voice", "audio", "rerank", "seedream",
		"seedance", "seededit", "wan2", "vidu", "mj_", "aigc-",
		"stable-diffusion", "midjourney", "nano-banana", "vectorize",
		"file-upload", "voice-clone", "transcribe", "realtime",
	}

	var result []llm.ModelInfo
	for _, m := range models {
		lower := strings.ToLower(m.ID)
		skip := false
		for _, kw := range excludeKeywords {
			if strings.Contains(lower, kw) {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, m)
		}
	}
	return result
}
