package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// Message LLM 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest OpenAI 兼容请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// ChatResponse OpenAI 兼容响应
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Client LLM 客户端
type Client struct {
	conf   *conf.LLM
	client *http.Client
	log    *log.Helper
}

// NewClient 创建 LLM 客户端
func NewClient(c *conf.LLM, logger log.Logger) *Client {
	helper := log.NewHelper(logger)
	helper.Infof("LLM client initialized: base_url=%s, default_model=%s, max_tokens=%d",
		c.BaseURL, c.DefaultModel, c.GetMaxTokens())
	if c.Models != nil {
		helper.Infof("LLM models: proposer=%s, opponent=%s, judge=%s, utility=%s",
			c.Models.Proposer, c.Models.Opponent, c.Models.Judge, c.Models.Utility)
	}
	return &Client{
		conf: c,
		client: &http.Client{
			Timeout: c.GetTimeout(),
		},
		log: helper,
	}
}

// CallOptions 调用选项
type CallOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
}

// Call 调用 LLM
func (c *Client) Call(ctx context.Context, systemPrompt string, messages []Message, opts *CallOptions) (string, error) {
	model := c.conf.DefaultModel
	maxTokens := c.conf.GetMaxTokens()
	temperature := 0.8

	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		if opts.MaxTokens > 0 {
			maxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			temperature = opts.Temperature
		}
	}

	allMessages := make([]Message, 0, len(messages)+1)
	allMessages = append(allMessages, Message{Role: "system", Content: systemPrompt})
	allMessages = append(allMessages, messages...)

	req := &ChatRequest{
		Model:       model,
		Messages:    allMessages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	var lastErr error
	maxRetries := c.conf.GetMaxRetries()

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := c.doRequest(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < maxRetries-1 {
			delay := time.Duration((attempt+1)*8) * time.Second
			if isRateLimit(err) {
				delay = time.Duration((attempt+1)*15) * time.Second
			}
			c.log.Warnf("LLM call failed (attempt %d/%d, model=%s): %v, retrying in %v...",
				attempt+1, maxRetries, model, err, delay)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return "", fmt.Errorf("LLM call failed after %d retries: %w", maxRetries, lastErr)
}

// CallWithRole 按角色调用 LLM（自动选择模型）
func (c *Client) CallWithRole(ctx context.Context, role string, systemPrompt string, messages []Message) (string, error) {
	model := c.conf.GetModel(role)
	return c.Call(ctx, systemPrompt, messages, &CallOptions{Model: model})
}

func (c *Client) doRequest(ctx context.Context, req *ChatRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.conf.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.conf.APIKey)

	resp, err := c.client.Do(httpReq)
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

func isRateLimit(err error) bool {
	if e, ok := err.(*LLMError); ok {
		return e.StatusCode == 429
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LLMError 可区分的 LLM 错误类型
type LLMError struct {
	StatusCode int
	Message    string
}

func (e *LLMError) Error() string {
	return e.Message
}
