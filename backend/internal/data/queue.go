package data

import (
	"context"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type queueRepo struct {
	data *Data
	log  *log.Helper
}

// NewQueueRepo 创建 QueueRepo 实现
func NewQueueRepo(data *Data, logger log.Logger) biz.QueueRepo {
	return &queueRepo{data: data, log: log.NewHelper(logger)}
}

func (r *queueRepo) Submit(ctx context.Context, item *biz.QueueItem) (*biz.QueueItem, error) {
	model := &ManualQueueModel{
		Topic:       item.Topic,
		Status:      item.Status,
		SubmittedBy: item.SubmittedBy,
	}
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return &biz.QueueItem{
		ID:          model.ID,
		Topic:       model.Topic,
		Status:      model.Status,
		SubmittedBy: model.SubmittedBy,
		CreatedAt:   model.CreatedAt,
	}, nil
}

func (r *queueRepo) List(ctx context.Context, query *biz.QueueListQuery) (*biz.QueueListResult, error) {
	var models []ManualQueueModel
	var total int64

	db := r.data.db.WithContext(ctx).Model(&ManualQueueModel{})
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]*biz.QueueItem, len(models))
	for i, m := range models {
		items[i] = &biz.QueueItem{
			ID:          m.ID,
			Topic:       m.Topic,
			Status:      m.Status,
			SubmittedBy: m.SubmittedBy,
			CreatedAt:   m.CreatedAt,
		}
	}

	return &biz.QueueListResult{
		Items: items,
		Total: int(total),
	}, nil
}

func (r *queueRepo) CheckDuplicate(ctx context.Context, topic string) (bool, error) {
	var count int64
	if err := r.data.db.WithContext(ctx).Model(&ManualQueueModel{}).
		Where("topic = ? AND status IN ?", topic, []string{"queued", "processing"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *queueRepo) GetPending(ctx context.Context, limit int) ([]*biz.QueueItem, error) {
	var models []ManualQueueModel
	if err := r.data.db.WithContext(ctx).Where("status = ?", "queued").
		Order("created_at ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]*biz.QueueItem, len(models))
	for i, m := range models {
		items[i] = &biz.QueueItem{
			ID:          m.ID,
			Topic:       m.Topic,
			Status:      m.Status,
			SubmittedBy: m.SubmittedBy,
			CreatedAt:   m.CreatedAt,
		}
	}
	return items, nil
}

func (r *queueRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.data.db.WithContext(ctx).Model(&ManualQueueModel{}).Where("id = ?", id).Update("status", status).Error
}
