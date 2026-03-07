package data

import "time"

// IdeaModel 数据库模型（已存在于其他文件中，此处不重复定义）

// SettingModel 系统设置 K-V 存储
type SettingModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Key       string    `gorm:"uniqueIndex;size:128;not null"`
	Value     string    `gorm:"type:text"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SettingModel) TableName() string { return "settings" }

// SearchKeywordModel 预设搜索关键词
type SearchKeywordModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Keyword   string    `gorm:"size:256;not null"`
	Enabled   bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SearchKeywordModel) TableName() string { return "search_keywords" }

// DiscoveredTopicModel 发现的话题
type DiscoveredTopicModel struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	Title           string    `gorm:"size:512;not null"`
	Source          string    `gorm:"size:64;not null"`  // 来源渠道：52pojie, bing, baidu, v2ex ...
	SourceURL       string    `gorm:"size:1024"`         // 原始链接
	Popularity      int       `gorm:"default:0"`         // 热度（浏览/点赞等）
	Replies         int       `gorm:"default:0"`         // 回复/讨论数
	Snippet         string    `gorm:"type:text"`         // 摘要或描述
	ContentHash     string    `gorm:"size:64;index"`     // 内容哈希，用于去重
	Status          string    `gorm:"size:32;default:'pending'"` // pending, recommended, submitted, dismissed
	Recommendation  string    `gorm:"type:text"`         // AI 推荐理由
	RecommendScore  float64   `gorm:"default:0"`         // AI 推荐分数 (0-10)
	IdeaID          int64     `gorm:"default:0"`         // 关联的 Idea ID（如果已提交辩论）
	DiscoveredAt    time.Time `gorm:"autoCreateTime"`    // 发现时间
	BatchID         string    `gorm:"size:64;index"`     // 批次 ID，同一轮发现的归为一批
}

func (DiscoveredTopicModel) TableName() string { return "discovered_topics" }

// CrawlerPluginModel 爬虫插件配置
type CrawlerPluginModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"uniqueIndex;size:64;not null"` // 插件名：52pojie, bing_keyword, baidu_keyword, v2ex ...
	Label     string    `gorm:"size:128"`                     // 显示名
	Enabled   bool      `gorm:"default:false"`
	Config    string    `gorm:"type:text"`                    // JSON 格式的插件配置
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (CrawlerPluginModel) TableName() string { return "crawler_plugins" }

// DiscoveryTagModel 话题发现约束标签（如：个人开发者、低成本启动）
type DiscoveryTagModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Tag       string    `gorm:"size:256;not null"`
	Category  string    `gorm:"size:64;default:'constraint'"` // constraint=约束条件, trend_query_cn=中文趋势搜索词, trend_query_en=英文趋势搜索词
	Enabled   bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (DiscoveryTagModel) TableName() string { return "discovery_tags" }
