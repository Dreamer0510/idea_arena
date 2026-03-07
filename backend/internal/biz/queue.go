package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// QueueItem 手动队列项
type QueueItem struct {
	ID          int64
	Topic       string
	Status      string
	SubmittedBy int64
	CreatedAt   time.Time
}

// QueueListQuery 队列查询参数
type QueueListQuery struct {
	Page     int
	PageSize int
	Status   string
}

// QueueListResult 队列查询结果
type QueueListResult struct {
	Items []*QueueItem
	Total int
}

// QueueRepo 数据仓储接口
type QueueRepo interface {
	Submit(ctx context.Context, item *QueueItem) (*QueueItem, error)
	List(ctx context.Context, query *QueueListQuery) (*QueueListResult, error)
	CheckDuplicate(ctx context.Context, topic string) (bool, error)
	GetPending(ctx context.Context, limit int) ([]*QueueItem, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// QueueUsecase 队列业务用例
type QueueUsecase struct {
	repo QueueRepo
	log  *log.Helper
}

// NewQueueUsecase 创建 QueueUsecase
func NewQueueUsecase(repo QueueRepo, logger log.Logger) *QueueUsecase {
	return &QueueUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Submit 提交话题到队列
func (uc *QueueUsecase) Submit(ctx context.Context, topic string, submittedBy int64) (*QueueItem, error) {
	// 去重检查
	exists, err := uc.repo.CheckDuplicate(ctx, topic)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("topic already exists in queue")
	}
	return uc.repo.Submit(ctx, &QueueItem{
		Topic:       topic,
		Status:      "queued",
		SubmittedBy: submittedBy,
	})
}

// List 获取队列列表
func (uc *QueueUsecase) List(ctx context.Context, query *QueueListQuery) (*QueueListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	return uc.repo.List(ctx, query)
}
