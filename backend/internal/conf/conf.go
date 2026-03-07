package conf

import (
	"time"
)

// Bootstrap 应用配置根结构
type Bootstrap struct {
	Server   *Server   `json:"server" yaml:"server"`
	Data     *Data     `json:"data" yaml:"data"`
	Auth     *Auth     `json:"auth" yaml:"auth"`
	LLM      *LLM      `json:"llm" yaml:"llm"`
	Debate   *Debate   `json:"debate" yaml:"debate"`
	Search   *Search   `json:"search" yaml:"search"`
	Notify   *Notify   `json:"notify" yaml:"notify"`
	Registry *Registry `json:"registry" yaml:"registry"`
}

type Server struct {
	HTTP *ServerItem `json:"http" yaml:"http"`
	GRPC *ServerItem `json:"grpc" yaml:"grpc"`
}

type ServerItem struct {
	Addr    string `json:"addr" yaml:"addr"`
	Timeout string `json:"timeout" yaml:"timeout"`
}

func (s *ServerItem) GetTimeout() time.Duration {
	d, _ := time.ParseDuration(s.Timeout)
	if d == 0 {
		d = 10 * time.Second
	}
	return d
}

type Data struct {
	Database *Database `json:"database" yaml:"database"`
	Redis    *Redis    `json:"redis" yaml:"redis"`
}

type Database struct {
	Driver string `json:"driver" yaml:"driver"`
	Source string `json:"source" yaml:"source"`
}

type Redis struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

type Auth struct {
	JWTSecret string `json:"jwt_secret" yaml:"jwt_secret"`
	JWTExpire string `json:"jwt_expire" yaml:"jwt_expire"`
}

func (a *Auth) GetJWTExpire() time.Duration {
	d, _ := time.ParseDuration(a.JWTExpire)
	if d == 0 {
		d = 24 * time.Hour
	}
	return d
}

type LLM struct {
	BaseURL       string     `json:"base_url" yaml:"base_url"`
	APIKey        string     `json:"api_key" yaml:"api_key"`
	DefaultModel  string     `json:"default_model" yaml:"default_model"`
	MaxTokens     int        `json:"max_tokens" yaml:"max_tokens"`
	TimeoutSec    int        `json:"timeout_sec" yaml:"timeout_sec"`
	MaxRetries    int        `json:"max_retries" yaml:"max_retries"`
	Models        *LLMModels `json:"models" yaml:"models"`
}

type LLMModels struct {
	Proposer string `json:"proposer" yaml:"proposer"`
	Opponent string `json:"opponent" yaml:"opponent"`
	Judge    string `json:"judge" yaml:"judge"`
	Utility  string `json:"utility" yaml:"utility"`
}

func (l *LLM) GetModel(role string) string {
	if l.Models != nil {
		switch role {
		case "proposer":
			if l.Models.Proposer != "" {
				return l.Models.Proposer
			}
		case "opponent":
			if l.Models.Opponent != "" {
				return l.Models.Opponent
			}
		case "judge":
			if l.Models.Judge != "" {
				return l.Models.Judge
			}
		case "utility":
			if l.Models.Utility != "" {
				return l.Models.Utility
			}
		}
	}
	if l.DefaultModel != "" {
		return l.DefaultModel
	}
	return "qwen-plus"
}

func (l *LLM) GetTimeout() time.Duration {
	if l.TimeoutSec > 0 {
		return time.Duration(l.TimeoutSec) * time.Second
	}
	return 180 * time.Second
}

func (l *LLM) GetMaxRetries() int {
	if l.MaxRetries > 0 {
		return l.MaxRetries
	}
	return 4
}

func (l *LLM) GetMaxTokens() int {
	if l.MaxTokens > 0 {
		return l.MaxTokens
	}
	return 4096
}

type Debate struct {
	MaxRounds       int     `json:"max_rounds" yaml:"max_rounds"`
	MinRounds       int     `json:"min_rounds" yaml:"min_rounds"`
	MaxConcurrent   int     `json:"max_concurrent" yaml:"max_concurrent"`
	Timeout         string  `json:"timeout" yaml:"timeout"`
	RetryCount      int     `json:"retry_count" yaml:"retry_count"`
	GraduationScore float64 `json:"graduation_score" yaml:"graduation_score"`
}

func (d *Debate) GetTimeout() time.Duration {
	dur, _ := time.ParseDuration(d.Timeout)
	if dur == 0 {
		dur = 5 * time.Minute
	}
	return dur
}

type Search struct {
	Bing   *SearchProvider `json:"bing" yaml:"bing"`
	Baidu  *SearchProvider `json:"baidu" yaml:"baidu"`
	Tavily *SearchProvider `json:"tavily" yaml:"tavily"`
	Serper *SearchProvider `json:"serper" yaml:"serper"`
}

type SearchProvider struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	APIKey  string `json:"api_key" yaml:"api_key"`
}

type Notify struct {
	Feishu *FeishuNotify `json:"feishu" yaml:"feishu"`
}

type FeishuNotify struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
}

type Registry struct {
	ProjectID   string `json:"project_id" yaml:"project_id"`
	ProjectName string `json:"project_name" yaml:"project_name"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}
