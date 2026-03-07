package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ProviderInfo 服务商信息
type ProviderInfo struct {
	ID      int64
	Name    string
	BaseURL string
	APIKey  string
}

// ProviderManager 多服务商 LLM 客户端管理器
type ProviderManager struct {
	mu      sync.RWMutex
	clients map[int64]*providerClient
	log     *log.Helper
}

type providerClient struct {
	info   ProviderInfo
	client *http.Client
}

// NewProviderManager 创建 ProviderManager
func NewProviderManager(logger log.Logger) *ProviderManager {
	return &ProviderManager{
		clients: make(map[int64]*providerClient),
		log:     log.NewHelper(logger),
	}
}

// RegisterProvider 注册/更新一个服务商
func (pm *ProviderManager) RegisterProvider(info ProviderInfo) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.clients[info.ID] = &providerClient{
		info: info,
		client: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
	pm.log.Infof("[ProviderManager] Registered provider: id=%d name=%s base_url=%s", info.ID, info.Name, info.BaseURL)
}

// RemoveProvider 移除一个服务商
func (pm *ProviderManager) RemoveProvider(id int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.clients, id)
}

// CallWithProvider 使用指定服务商调用 LLM
func (pm *ProviderManager) CallWithProvider(ctx context.Context, providerID int64, model string, systemPrompt string, messages []Message, opts *CallOptions) (string, error) {
	pm.mu.RLock()
	pc, ok := pm.clients[providerID]
	pm.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("provider %d not found", providerID)
	}

	maxTokens := 4096
	temperature := 0.8
	if opts != nil {
		if opts.MaxTokens > 0 {
			maxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			temperature = opts.Temperature
		}
	}

	allMessages := make([]Message, 0, len(messages)+1)
	if systemPrompt != "" {
		allMessages = append(allMessages, Message{Role: "system", Content: systemPrompt})
	}
	allMessages = append(allMessages, messages...)

	req := &ChatRequest{
		Model:       model,
		Messages:    allMessages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	var lastErr error
	maxRetries := 4

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := pm.doProviderRequest(ctx, pc, req)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < maxRetries-1 {
			delay := time.Duration((attempt+1)*8) * time.Second
			if isRateLimit(err) {
				delay = time.Duration((attempt+1)*15) * time.Second
			}
			pm.log.Warnf("[ProviderManager] Call failed (provider=%s model=%s attempt=%d/%d): %v, retrying in %v...",
				pc.info.Name, model, attempt+1, maxRetries, err, delay)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return "", fmt.Errorf("provider %s call failed after %d retries: %w", pc.info.Name, maxRetries, lastErr)
}

// TestProviderResult 测试结果
type TestProviderResult struct {
	Response   string `json:"response"`
	WorkingURL string `json:"working_url"` // 实际可用的 base URL
}

// TestProvider 测试服务商连通性（多策略探测 URL）
func (pm *ProviderManager) TestProvider(ctx context.Context, info ProviderInfo, model string) (*TestProviderResult, error) {
	// 生成候选 URL 列表（去重保序）
	candidates := buildURLCandidates(info.BaseURL)

	client := &http.Client{Timeout: 15 * time.Second}

	// 1) 先用 /models 接口快速探测哪个 URL 可用
	var workingURL string
	for _, base := range candidates {
		url := base + "/models"
		httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+info.APIKey)
		resp, err := client.Do(httpReq)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var mr ModelsResponse
			if json.Unmarshal(body, &mr) == nil && len(mr.Data) > 0 {
				workingURL = base
				break
			}
		}
	}

	// 2) 如果 /models 探测全部失败，用 /chat/completions 再试一次（多模型名尝试）
	var probeModel string // 记录探测成功的模型名
	if workingURL == "" {
		testModels := []string{"qwen-plus", "qwen3-coder-plus", "gpt-4o-mini", "gpt-3.5-turbo", "deepseek-chat", "glm-4"}
		for _, base := range candidates {
			for _, m := range testModels {
				pc := &providerClient{
					info:   ProviderInfo{BaseURL: base, APIKey: info.APIKey},
					client: client,
				}
				testReq := &ChatRequest{
					Model:    m,
					Messages: []Message{{Role: "user", Content: "hi"}},
					MaxTokens: 5, Temperature: 0,
				}
				if _, err := pm.doProviderRequest(ctx, pc, testReq); err == nil {
					workingURL = base
					probeModel = m
					break
				}
			}
			if workingURL != "" {
				break
			}
		}
	}

	if workingURL == "" {
		return nil, fmt.Errorf("所有 URL 探测失败，请检查 Base URL 和 API Key")
	}

	// 3) 用 chat 接口做真实测试
	if model == "" {
		// 优先用 /models 推断，其次用探测成功的模型
		model = pm.pickTestModel(ctx, workingURL, info.APIKey)
		if model == "" && probeModel != "" {
			model = probeModel
		}
		if model == "" {
			model = "qwen-plus"
		}
	}

	pc := &providerClient{
		info:   ProviderInfo{BaseURL: workingURL, APIKey: info.APIKey},
		client: &http.Client{Timeout: 30 * time.Second},
	}
	req := &ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: "你是一个测试助手。"},
			{Role: "user", Content: "请回复 'OK' 两个字母，不要其他内容。"},
		},
		MaxTokens:   10,
		Temperature: 0,
	}

	result, err := pm.doProviderRequest(ctx, pc, req)
	if err != nil {
		// chat 失败但 models 通了，仍算部分成功
		return &TestProviderResult{
			Response:   fmt.Sprintf("API 连通（/models 正常），但 chat 测试失败: %v", err),
			WorkingURL: workingURL,
		}, nil
	}

	return &TestProviderResult{
		Response:   result,
		WorkingURL: workingURL,
	}, nil
}

// pickTestModel 从 /models 获取第一个可用的聊天模型名
func (pm *ProviderManager) pickTestModel(ctx context.Context, baseURL, apiKey string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	url := baseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ""
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(httpReq)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var mr ModelsResponse
	if json.Unmarshal(body, &mr) != nil || len(mr.Data) == 0 {
		return ""
	}
	// 优先选 chat 类模型
	preferred := []string{"qwen", "gpt", "claude", "deepseek", "glm", "yi-"}
	for _, p := range preferred {
		for _, m := range mr.Data {
			if strings.Contains(strings.ToLower(m.ID), p) {
				return m.ID
			}
		}
	}
	return mr.Data[0].ID
}

// buildURLCandidates 生成候选 URL（去重保序）
func buildURLCandidates(rawURL string) []string {
	u := strings.TrimSuffix(rawURL, "/")
	seen := map[string]bool{}
	var result []string
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	// 原始 URL 优先
	add(u)
	// 如果没有版本路径，追加 /v1
	parts := strings.Split(u, "/")
	last := parts[len(parts)-1]
	hasVersion := len(last) >= 2 && last[0] == 'v' && last[1] >= '0' && last[1] <= '9'
	if !hasVersion {
		add(u + "/v1")
	} else {
		// 如果已有版本路径，也试试去掉
		add(strings.TrimSuffix(u, "/"+last))
	}
	return result
}

// FetchModels 获取服务商支持的模型列表（支持自定义模型接口 URL）
func (pm *ProviderManager) FetchModels(ctx context.Context, info ProviderInfo, modelsURL string) ([]ModelInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	url := normalizeBaseURL(info.BaseURL) + "/models"
	if modelsURL != "" {
		url = modelsURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+info.APIKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return modelsResp.Data, nil
}

func (pm *ProviderManager) doProviderRequest(ctx context.Context, pc *providerClient, req *ChatRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := normalizeBaseURL(pc.info.BaseURL) + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+pc.info.APIKey)

	resp, err := pc.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", &LLMError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("LLM API error: %d %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)])),
		}
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// normalizeBaseURL 确保 base URL 以 /v1 结尾（OpenAI 兼容格式）
func normalizeBaseURL(baseURL string) string {
	u := strings.TrimSuffix(baseURL, "/")
	// 已有版本路径如 /v1, /v2 等
	parts := strings.Split(u, "/")
	last := parts[len(parts)-1]
	if len(last) >= 2 && last[0] == 'v' && last[1] >= '0' && last[1] <= '9' {
		return u
	}
	return u + "/v1"
}

// ModelInfo 模型信息（来自 /v1/models 接口）
type ModelInfo struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse /v1/models 响应
type ModelsResponse struct {
	Data []ModelInfo `json:"data"`
}
