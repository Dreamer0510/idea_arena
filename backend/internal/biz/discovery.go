package biz

import (
	"context"
	"time"
)

// DiscoveredTopic 发现的话题业务实体
type DiscoveredTopic struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	Source            string    `json:"source"`
	SourceURL         string    `json:"source_url"`
	Popularity        int       `json:"popularity"`
	Replies           int       `json:"replies"`
	Snippet           string    `json:"snippet"`
	ContentHash       string    `json:"content_hash,omitempty"`
	Status            string    `json:"status"`
	Recommendation    string    `json:"recommendation"`
	PainScore         float64   `json:"pain_score"`
	TrendScore        float64   `json:"trend_score"`
	FeasibilityScore  float64   `json:"feasibility_score"`
	MonetizationScore float64   `json:"monetization_score"`
	NoveltyScore      float64   `json:"novelty_score"`
	RecommendScore    float64   `json:"recommend_score"`
	IdeaID            int64     `json:"idea_id"`
	DiscoveredAt      time.Time `json:"discovered_at"`
	BatchID           string    `json:"batch_id"`
}

// CrawlerPlugin 爬虫插件配置
type CrawlerPlugin struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Config  string `json:"config,omitempty"`
}

// SearchKeyword 搜索关键词
type SearchKeyword struct {
	ID      int64  `json:"id"`
	Keyword string `json:"keyword"`
	Enabled bool   `json:"enabled"`
}

// SystemSettings 系统可动态调整的设置
type SystemSettings struct {
	DebateMaxRounds       int     `json:"debate_max_rounds"`
	DebateGraduationScore float64 `json:"debate_graduation_score"`
	DebateTimeout         string  `json:"debate_timeout"`
	DiscoveryInterval     string  `json:"discovery_interval"` // 话题发现间隔，如 "3h"
	DiscoveryEnabled      bool    `json:"discovery_enabled"`  // 是否启用自动发现
	TopicsPerSource       int     `json:"topics_per_source"`  // 每个渠道获取的话题数
	AutoSubmitDebate      bool    `json:"auto_submit_debate"` // 是否自动提交辩论
}

// DefaultSettings 默认设置
func DefaultSettings() *SystemSettings {
	return &SystemSettings{
		DebateMaxRounds:       20,
		DebateGraduationScore: 8.0,
		DebateTimeout:         "1800s",
		DiscoveryInterval:     "40m",
		DiscoveryEnabled:      false,
		TopicsPerSource:       10,
		AutoSubmitDebate:      true,
	}
}

// DiscoveryTag 话题发现约束标签
type DiscoveryTag struct {
	ID       int64  `json:"id"`
	Tag      string `json:"tag"`
	Category string `json:"category"` // constraint, trend_query_cn, trend_query_en
	Enabled  bool   `json:"enabled"`
}

// ---- 统一话题数据结构（各渠道爬虫的标准输出）----

// RawTopic 各渠道爬虫产出的原始话题
type RawTopic struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	Source     string `json:"source"`     // 渠道标识
	Popularity int    `json:"popularity"` // 热度指标（浏览数/搜索热度等）
	Replies    int    `json:"replies"`    // 讨论度（回复数/评论数等）
	Snippet    string `json:"snippet"`    // 摘要/描述
}

// TopicAnalysis AI 分析结果
type TopicAnalysis struct {
	RecommendScore float64 `json:"recommend_score"` // 0-10
	Recommendation string  `json:"recommendation"`  // 推荐理由（含数据支撑）
	SuggestedTopic string  `json:"suggested_topic"` // 建议的辩论话题方向
}

// TopicSourceStat 来源命中率统计
type TopicSourceStat struct {
	Source           string  `json:"source"`
	TotalCount       int64   `json:"total_count"`
	RecommendedCount int64   `json:"recommended_count"`
	SubmittedCount   int64   `json:"submitted_count"`
	DismissedCount   int64   `json:"dismissed_count"`
	PendingCount     int64   `json:"pending_count"`
	HitRate          float64 `json:"hit_rate"`    // 推荐命中率（recommended+submitted）/total
	SubmitRate       float64 `json:"submit_rate"` // 提交率 submitted/total
}

// SourceFeedbackStat 来源反哺统计（用于评分权重微调）
type SourceFeedbackStat struct {
	Source      string  `json:"source"`
	SampleSize  int64   `json:"sample_size"`  // 已进入辩论样本数
	SuccessRate float64 `json:"success_rate"` // promising+graduated 占比
}

// ---- Crawler Plugin 接口（定义在 biz 层，crawler 包实现）----

// CrawlerPluginInterface 爬虫插件接口
type CrawlerPluginInterface interface {
	// Name 插件唯一标识
	Name() string
	// Label 插件显示名
	Label() string
	// Fetch 抓取话题，limit 为最大数量
	Fetch(ctx context.Context, limit int) ([]*RawTopic, error)
}

// ---- Repository 接口 ----

// DiscoveryRepo 话题发现数据仓储
type DiscoveryRepo interface {
	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetAllSettings(ctx context.Context) (map[string]string, error)

	// Keywords
	ListKeywords(ctx context.Context) ([]*SearchKeyword, error)
	CreateKeyword(ctx context.Context, keyword string) (*SearchKeyword, error)
	UpdateKeyword(ctx context.Context, id int64, keyword string, enabled bool) error
	DeleteKeyword(ctx context.Context, id int64) error

	// Plugins
	ListPlugins(ctx context.Context) ([]*CrawlerPlugin, error)
	GetPlugin(ctx context.Context, name string) (*CrawlerPlugin, error)
	UpsertPlugin(ctx context.Context, plugin *CrawlerPlugin) error
	TogglePlugin(ctx context.Context, name string, enabled bool) error

	// Tags
	ListTags(ctx context.Context) ([]*DiscoveryTag, error)
	ListTagsByCategory(ctx context.Context, category string) ([]*DiscoveryTag, error)
	CreateTag(ctx context.Context, tag string, category string) (*DiscoveryTag, error)
	UpdateTag(ctx context.Context, id int64, tag string, category string, enabled bool) error
	DeleteTag(ctx context.Context, id int64) error

	// Discovered Topics
	SaveTopics(ctx context.Context, topics []*DiscoveredTopic) error
	ListTopics(ctx context.Context, status string, page, pageSize int) ([]*DiscoveredTopic, int, error)
	ListTopicSourceStats(ctx context.Context, days int) ([]*TopicSourceStat, error)
	ListSourceFeedbackStats(ctx context.Context, days int) ([]*SourceFeedbackStat, error)
	GetTopic(ctx context.Context, id int64) (*DiscoveredTopic, error)
	UpdateTopicStatus(ctx context.Context, id int64, status string) error
	UpdateTopicAnalysis(ctx context.Context, id int64, recommendation string, score float64) error
	SetTopicIdeaID(ctx context.Context, id int64, ideaID int64) error
	ExistsByHash(ctx context.Context, hash string) (bool, error)
}
