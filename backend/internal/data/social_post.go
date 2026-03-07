package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SocialPost 社交推文业务实体
type SocialPost struct {
	ID          int64      `json:"id"`
	IdeaID      int64      `json:"idea_id"`
	Platform    string     `json:"platform"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SocialPostRepo 社交推文仓储
type SocialPostRepo struct {
	data *Data
	log  *log.Helper
}

// NewSocialPostRepo 创建 SocialPostRepo
func NewSocialPostRepo(data *Data, logger log.Logger) *SocialPostRepo {
	return &SocialPostRepo{data: data, log: log.NewHelper(logger)}
}

func (r *SocialPostRepo) Create(ctx context.Context, post *SocialPost) (*SocialPost, error) {
	model := &SocialPostModel{
		IdeaID:   post.IdeaID,
		Platform: post.Platform,
		Title:    post.Title,
		Content:  post.Content,
		Status:   "draft",
	}
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return modelToPost(model), nil
}

func (r *SocialPostRepo) List(ctx context.Context, status string) ([]*SocialPost, error) {
	var models []SocialPostModel
	q := r.data.db.WithContext(ctx).Order("created_at desc")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	posts := make([]*SocialPost, len(models))
	for i, m := range models {
		posts[i] = modelToPost(&m)
	}
	return posts, nil
}

func (r *SocialPostRepo) GetByID(ctx context.Context, id int64) (*SocialPost, error) {
	var model SocialPostModel
	if err := r.data.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	return modelToPost(&model), nil
}

func (r *SocialPostRepo) Update(ctx context.Context, post *SocialPost) error {
	updates := map[string]interface{}{}
	if post.Title != "" {
		updates["title"] = post.Title
	}
	if post.Content != "" {
		updates["content"] = post.Content
	}
	if post.Status != "" {
		updates["status"] = post.Status
	}
	if len(updates) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).Model(&SocialPostModel{}).Where("id = ?", post.ID).Updates(updates).Error
}

func (r *SocialPostRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&SocialPostModel{}, id).Error
}

func modelToPost(m *SocialPostModel) *SocialPost {
	return &SocialPost{
		ID:          m.ID,
		IdeaID:      m.IdeaID,
		Platform:    m.Platform,
		Title:       m.Title,
		Content:     m.Content,
		Status:      m.Status,
		PublishedAt: m.PublishedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
