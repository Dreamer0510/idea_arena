package data

import (
	"time"

	"gorm.io/gorm"
)

// IdeaModel 创业点子数据模型
type IdeaModel struct {
	ID               int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Topic            string         `gorm:"type:text;not null" json:"topic"`
	Status           string         `gorm:"type:varchar(20);default:pending;index:idx_ideas_status" json:"status"`
	Proposal         string         `gorm:"type:longtext" json:"proposal"`
	DebateLog        string         `gorm:"type:longtext" json:"debate_log"`
	FinalReport      string         `gorm:"type:longtext" json:"final_report"`
	ScoreFeasibility float64        `gorm:"type:real;default:0" json:"score_feasibility"`
	ScoreEconomics   float64        `gorm:"type:real;default:0" json:"score_economics"`
	ScoreProfit      float64        `gorm:"type:real;default:0" json:"score_profit"`
	ScoreOverall     float64        `gorm:"type:real;default:0;index:idx_ideas_status_score" json:"score_overall"`
	RoundCount       int            `gorm:"default:0" json:"round_count"`
	TechStack        string         `gorm:"type:longtext" json:"tech_stack"`
	DevPrompt        string         `gorm:"type:longtext" json:"dev_prompt"`
	Tags             string         `gorm:"type:text" json:"tags"`
	ProductName      string         `gorm:"type:varchar(255)" json:"product_name"`
	OneLiner         string         `gorm:"type:varchar(500)" json:"one_liner"`
	SearchData       string         `gorm:"type:longtext" json:"search_data"`
	JudgeRefined     string         `gorm:"type:longtext" json:"judge_refined"`
	VoteResult       string         `gorm:"type:longtext" json:"vote_result"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (IdeaModel) TableName() string { return "ideas" }

// UserModel 用户数据模型
type UserModel struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"type:varchar(100);uniqueIndex" json:"username"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	Role         string         `gorm:"type:varchar(20);default:user" json:"role"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserModel) TableName() string { return "users" }

// ManualQueueModel 手动提交队列
type ManualQueueModel struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Topic       string         `gorm:"type:text;not null" json:"topic"`
	Status      string         `gorm:"type:varchar(20);default:queued;index" json:"status"`
	SubmittedBy int64          `gorm:"default:0" json:"submitted_by"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ManualQueueModel) TableName() string { return "manual_queue" }

// SocialPostModel 社交推文数据模型
type SocialPostModel struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	IdeaID      int64          `gorm:"index" json:"idea_id"`
	Platform    string         `gorm:"type:varchar(50);not null" json:"platform"`
	Title       string         `gorm:"type:varchar(500)" json:"title"`
	Content     string         `gorm:"type:text" json:"content"`
	Status      string         `gorm:"type:varchar(20);default:draft;index" json:"status"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SocialPostModel) TableName() string { return "social_posts" }

// AgentConfigModel Agent 人设配置
type AgentConfigModel struct {
	ID           string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name         string         `gorm:"type:varchar(100)" json:"name"`
	Emoji        string         `gorm:"type:varchar(10)" json:"emoji"`
	Role         string         `gorm:"type:varchar(50)" json:"role"`
	SystemPrompt string         `gorm:"type:text" json:"system_prompt"`
	Temperature  float64        `gorm:"type:real;default:0.8" json:"temperature"`
	ProviderID   int64          `gorm:"default:0" json:"provider_id"`
	ModelName    string         `gorm:"type:varchar(200)" json:"model_name"`
	Enabled      bool           `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AgentConfigModel) TableName() string { return "agent_configs" }

// LLMProviderModel LLM 服务商
type LLMProviderModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	BaseURL   string         `gorm:"type:varchar(500);not null" json:"base_url"`
	APIKey    string         `gorm:"type:varchar(500);not null" json:"api_key"`
	ModelsURL string         `gorm:"type:varchar(500)" json:"models_url"`
	IsDefault bool           `gorm:"default:false" json:"is_default"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LLMProviderModel) TableName() string { return "llm_providers" }

// LLMProviderModelListModel 服务商支持的模型缓存
type LLMProviderModelListModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID int64     `gorm:"index;not null" json:"provider_id"`
	ModelID    string    `gorm:"type:varchar(200);not null" json:"model_id"`
	OwnedBy    string    `gorm:"type:varchar(100)" json:"owned_by"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (LLMProviderModelListModel) TableName() string { return "llm_provider_models" }
