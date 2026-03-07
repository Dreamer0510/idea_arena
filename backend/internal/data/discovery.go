package data

import (
	"context"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type discoveryRepo struct {
	data *Data
	log  *log.Helper
}

// NewDiscoveryRepo 创建 DiscoveryRepo 实现
func NewDiscoveryRepo(data *Data, logger log.Logger) biz.DiscoveryRepo {
	return &discoveryRepo{data: data, log: log.NewHelper(logger)}
}

// ---- Settings ----

func (r *discoveryRepo) GetSetting(ctx context.Context, key string) (string, error) {
	var m SettingModel
	if err := r.data.db.WithContext(ctx).Where("`key` = ?", key).First(&m).Error; err != nil {
		return "", err
	}
	return m.Value, nil
}

func (r *discoveryRepo) SetSetting(ctx context.Context, key, value string) error {
	var m SettingModel
	result := r.data.db.WithContext(ctx).Where("`key` = ?", key).First(&m)
	if result.Error != nil {
		// 不存在则创建
		return r.data.db.WithContext(ctx).Create(&SettingModel{Key: key, Value: value}).Error
	}
	return r.data.db.WithContext(ctx).Model(&m).Update("value", value).Error
}

func (r *discoveryRepo) GetAllSettings(ctx context.Context) (map[string]string, error) {
	var models []SettingModel
	if err := r.data.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	settings := make(map[string]string, len(models))
	for _, m := range models {
		settings[m.Key] = m.Value
	}
	return settings, nil
}

// ---- Keywords ----

func (r *discoveryRepo) ListKeywords(ctx context.Context) ([]*biz.SearchKeyword, error) {
	var models []SearchKeywordModel
	if err := r.data.db.WithContext(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.SearchKeyword, len(models))
	for i, m := range models {
		result[i] = &biz.SearchKeyword{ID: m.ID, Keyword: m.Keyword, Enabled: m.Enabled}
	}
	return result, nil
}

func (r *discoveryRepo) CreateKeyword(ctx context.Context, keyword string) (*biz.SearchKeyword, error) {
	m := &SearchKeywordModel{Keyword: keyword, Enabled: true}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return &biz.SearchKeyword{ID: m.ID, Keyword: m.Keyword, Enabled: m.Enabled}, nil
}

func (r *discoveryRepo) UpdateKeyword(ctx context.Context, id int64, keyword string, enabled bool) error {
	return r.data.db.WithContext(ctx).Model(&SearchKeywordModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"keyword": keyword, "enabled": enabled}).Error
}

func (r *discoveryRepo) DeleteKeyword(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&SearchKeywordModel{}, id).Error
}

// ---- Plugins ----

func (r *discoveryRepo) ListPlugins(ctx context.Context) ([]*biz.CrawlerPlugin, error) {
	var models []CrawlerPluginModel
	if err := r.data.db.WithContext(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.CrawlerPlugin, len(models))
	for i, m := range models {
		result[i] = &biz.CrawlerPlugin{ID: m.ID, Name: m.Name, Label: m.Label, Enabled: m.Enabled, Config: m.Config}
	}
	return result, nil
}

func (r *discoveryRepo) GetPlugin(ctx context.Context, name string) (*biz.CrawlerPlugin, error) {
	var m CrawlerPluginModel
	if err := r.data.db.WithContext(ctx).Where("name = ?", name).First(&m).Error; err != nil {
		return nil, err
	}
	return &biz.CrawlerPlugin{ID: m.ID, Name: m.Name, Label: m.Label, Enabled: m.Enabled, Config: m.Config}, nil
}

func (r *discoveryRepo) UpsertPlugin(ctx context.Context, plugin *biz.CrawlerPlugin) error {
	var existing CrawlerPluginModel
	result := r.data.db.WithContext(ctx).Where("name = ?", plugin.Name).First(&existing)
	if result.Error != nil {
		// 不存在，创建
		return r.data.db.WithContext(ctx).Create(&CrawlerPluginModel{
			Name:    plugin.Name,
			Label:   plugin.Label,
			Enabled: plugin.Enabled,
			Config:  plugin.Config,
		}).Error
	}
	// 存在，更新
	return r.data.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"label":   plugin.Label,
		"enabled": plugin.Enabled,
		"config":  plugin.Config,
	}).Error
}

func (r *discoveryRepo) TogglePlugin(ctx context.Context, name string, enabled bool) error {
	return r.data.db.WithContext(ctx).Model(&CrawlerPluginModel{}).Where("name = ?", name).
		Update("enabled", enabled).Error
}

// ---- Tags ----

func (r *discoveryRepo) ListTags(ctx context.Context) ([]*biz.DiscoveryTag, error) {
	var models []DiscoveryTagModel
	if err := r.data.db.WithContext(ctx).Order("category ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.DiscoveryTag, len(models))
	for i, m := range models {
		result[i] = &biz.DiscoveryTag{ID: m.ID, Tag: m.Tag, Category: m.Category, Enabled: m.Enabled}
	}
	return result, nil
}

func (r *discoveryRepo) ListTagsByCategory(ctx context.Context, category string) ([]*biz.DiscoveryTag, error) {
	var models []DiscoveryTagModel
	if err := r.data.db.WithContext(ctx).Where("category = ? AND enabled = ?", category, true).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.DiscoveryTag, len(models))
	for i, m := range models {
		result[i] = &biz.DiscoveryTag{ID: m.ID, Tag: m.Tag, Category: m.Category, Enabled: m.Enabled}
	}
	return result, nil
}

func (r *discoveryRepo) CreateTag(ctx context.Context, tag string, category string) (*biz.DiscoveryTag, error) {
	m := &DiscoveryTagModel{Tag: tag, Category: category, Enabled: true}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return &biz.DiscoveryTag{ID: m.ID, Tag: m.Tag, Category: m.Category, Enabled: m.Enabled}, nil
}

func (r *discoveryRepo) UpdateTag(ctx context.Context, id int64, tag string, category string, enabled bool) error {
	return r.data.db.WithContext(ctx).Model(&DiscoveryTagModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"tag": tag, "category": category, "enabled": enabled}).Error
}

func (r *discoveryRepo) DeleteTag(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&DiscoveryTagModel{}, id).Error
}

// ---- Discovered Topics ----

func (r *discoveryRepo) SaveTopics(ctx context.Context, topics []*biz.DiscoveredTopic) error {
	if len(topics) == 0 {
		return nil
	}
	models := make([]DiscoveredTopicModel, len(topics))
	for i, t := range topics {
		models[i] = DiscoveredTopicModel{
			Title:             t.Title,
			Source:            t.Source,
			SourceURL:         t.SourceURL,
			Popularity:        t.Popularity,
			Replies:           t.Replies,
			Snippet:           t.Snippet,
			ContentHash:       t.ContentHash,
			Status:            t.Status,
			Recommendation:    t.Recommendation,
			PainScore:         t.PainScore,
			TrendScore:        t.TrendScore,
			FeasibilityScore:  t.FeasibilityScore,
			MonetizationScore: t.MonetizationScore,
			NoveltyScore:      t.NoveltyScore,
			RecommendScore:    t.RecommendScore,
			BatchID:           t.BatchID,
		}
	}
	if err := r.data.db.WithContext(ctx).Create(&models).Error; err != nil {
		return err
	}
	// 回写自增 ID 到 biz 实体
	for i := range models {
		topics[i].ID = models[i].ID
	}
	return nil
}

func (r *discoveryRepo) ListTopics(ctx context.Context, status string, page, pageSize int) ([]*biz.DiscoveredTopic, int, error) {
	db := r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{})
	if status != "" {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []DiscoveredTopicModel
	offset := (page - 1) * pageSize
	if err := db.Order("recommend_score DESC, discovered_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.DiscoveredTopic, len(models))
	for i, m := range models {
		result[i] = toDiscoveredTopicBiz(&m)
	}
	return result, int(total), nil
}

func (r *discoveryRepo) ListTopicSourceStats(ctx context.Context, days int) ([]*biz.TopicSourceStat, error) {
	type sourceStatRow struct {
		Source           string `gorm:"column:source"`
		TotalCount       int64  `gorm:"column:total_count"`
		RecommendedCount int64  `gorm:"column:recommended_count"`
		SubmittedCount   int64  `gorm:"column:submitted_count"`
		DismissedCount   int64  `gorm:"column:dismissed_count"`
		PendingCount     int64  `gorm:"column:pending_count"`
	}

	query := r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{})
	if days > 0 {
		since := time.Now().AddDate(0, 0, -days)
		query = query.Where("discovered_at >= ?", since)
	}

	var rows []sourceStatRow
	if err := query.
		Select(`
			source,
			COUNT(*) AS total_count,
			SUM(CASE WHEN status = 'recommended' THEN 1 ELSE 0 END) AS recommended_count,
			SUM(CASE WHEN status = 'submitted' THEN 1 ELSE 0 END) AS submitted_count,
			SUM(CASE WHEN status = 'dismissed' THEN 1 ELSE 0 END) AS dismissed_count,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_count
		`).
		Group("source").
		Order("total_count DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.TopicSourceStat, 0, len(rows))
	for _, row := range rows {
		result = append(result, &biz.TopicSourceStat{
			Source:           row.Source,
			TotalCount:       row.TotalCount,
			RecommendedCount: row.RecommendedCount,
			SubmittedCount:   row.SubmittedCount,
			DismissedCount:   row.DismissedCount,
			PendingCount:     row.PendingCount,
		})
	}
	return result, nil
}

func (r *discoveryRepo) ListSourceFeedbackStats(ctx context.Context, days int) ([]*biz.SourceFeedbackStat, error) {
	type feedbackRow struct {
		Source       string `gorm:"column:source"`
		SampleSize   int64  `gorm:"column:sample_size"`
		SuccessCount int64  `gorm:"column:success_count"`
	}

	query := r.data.db.WithContext(ctx).Table("discovered_topics AS dt").
		Joins("LEFT JOIN ideas AS i ON i.id = dt.idea_id AND i.deleted_at IS NULL").
		Where("dt.idea_id > 0")

	if days > 0 {
		since := time.Now().AddDate(0, 0, -days)
		query = query.Where("dt.discovered_at >= ?", since)
	}

	var rows []feedbackRow
	if err := query.
		Select(`
			dt.source AS source,
			COUNT(*) AS sample_size,
			SUM(CASE WHEN i.status IN ('promising', 'graduated') THEN 1 ELSE 0 END) AS success_count
		`).
		Group("dt.source").
		Order("sample_size DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.SourceFeedbackStat, 0, len(rows))
	for _, row := range rows {
		if row.Source == "" || row.SampleSize <= 0 {
			continue
		}
		successRate := float64(row.SuccessCount) / float64(row.SampleSize)
		result = append(result, &biz.SourceFeedbackStat{
			Source:      row.Source,
			SampleSize:  row.SampleSize,
			SuccessRate: successRate,
		})
	}
	return result, nil
}

func (r *discoveryRepo) GetTopic(ctx context.Context, id int64) (*biz.DiscoveredTopic, error) {
	var m DiscoveredTopicModel
	if err := r.data.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDiscoveredTopicBiz(&m), nil
}

func (r *discoveryRepo) UpdateTopicStatus(ctx context.Context, id int64, status string) error {
	return r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{}).Where("id = ?", id).
		Update("status", status).Error
}

func (r *discoveryRepo) UpdateTopicAnalysis(ctx context.Context, id int64, recommendation string, score float64) error {
	return r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"recommendation":  recommendation,
			"recommend_score": score,
			"status":          "recommended",
		}).Error
}

func (r *discoveryRepo) SetTopicIdeaID(ctx context.Context, id int64, ideaID int64) error {
	return r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"idea_id": ideaID,
			"status":  "submitted",
		}).Error
}

func (r *discoveryRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var count int64
	if err := r.data.db.WithContext(ctx).Model(&DiscoveredTopicModel{}).
		Where("content_hash = ?", hash).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func toDiscoveredTopicBiz(m *DiscoveredTopicModel) *biz.DiscoveredTopic {
	return &biz.DiscoveredTopic{
		ID:                m.ID,
		Title:             m.Title,
		Source:            m.Source,
		SourceURL:         m.SourceURL,
		Popularity:        m.Popularity,
		Replies:           m.Replies,
		Snippet:           m.Snippet,
		ContentHash:       m.ContentHash,
		Status:            m.Status,
		Recommendation:    m.Recommendation,
		PainScore:         m.PainScore,
		TrendScore:        m.TrendScore,
		FeasibilityScore:  m.FeasibilityScore,
		MonetizationScore: m.MonetizationScore,
		NoveltyScore:      m.NoveltyScore,
		RecommendScore:    m.RecommendScore,
		IdeaID:            m.IdeaID,
		DiscoveredAt:      m.DiscoveredAt,
		BatchID:           m.BatchID,
	}
}
