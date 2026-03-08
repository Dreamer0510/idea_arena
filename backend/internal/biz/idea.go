package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Idea 业务实体
type Idea struct {
	ID               int64
	Topic            string
	Status           string
	Proposal         string
	DebateLog        string
	FinalReport      string
	ScoreFeasibility float64
	ScoreEconomics   float64
	ScoreProfit      float64
	ScoreOverall     float64
	RoundCount       int
	TechStack        string
	DevPrompt        string
	Tags             []string
	ProductName      string
	OneLiner         string
	SearchData       string
	JudgeRefined     string
	VoteResult       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IdeaListQuery 列表查询参数
type IdeaListQuery struct {
	Page      int
	PageSize  int
	Status    string
	SortBy    string
	SortOrder string
	MinScore  float64
}

// IdeaListResult 列表查询结果
type IdeaListResult struct {
	Items    []*Idea
	Total    int
	Page     int
	PageSize int
}

// IdeaRepo 数据仓储接口（在 biz 层定义，data 层实现）
type IdeaRepo interface {
	Create(ctx context.Context, idea *Idea) (*Idea, error)
	GetByID(ctx context.Context, id int64) (*Idea, error)
	Update(ctx context.Context, idea *Idea) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query *IdeaListQuery) (*IdeaListResult, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateScores(ctx context.Context, id int64, feasibility, economics, profit, overall float64) error
	UpdateDebateLog(ctx context.Context, id int64, debateLog string, roundCount int) error
	GetPending(ctx context.Context, limit int) ([]*Idea, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
}

// IdeaUsecase Idea 业务用例
type IdeaUsecase struct {
	repo IdeaRepo
	log  *log.Helper
}

// NewIdeaUsecase 创建 IdeaUsecase
func NewIdeaUsecase(repo IdeaRepo, logger log.Logger) *IdeaUsecase {
	return &IdeaUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Create 创建 Idea
func (uc *IdeaUsecase) Create(ctx context.Context, topic string) (*Idea, error) {
	idea := &Idea{
		Topic:  topic,
		Status: "pending",
	}
	return uc.repo.Create(ctx, idea)
}

// Get 获取 Idea 详情
func (uc *IdeaUsecase) Get(ctx context.Context, id int64) (*Idea, error) {
	return uc.repo.GetByID(ctx, id)
}

// List 获取 Idea 列表
func (uc *IdeaUsecase) List(ctx context.Context, query *IdeaListQuery) (*IdeaListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.SortBy == "" {
		query.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}
	return uc.repo.List(ctx, query)
}

// Update 更新 Idea
func (uc *IdeaUsecase) Update(ctx context.Context, idea *Idea) error {
	return uc.repo.Update(ctx, idea)
}

// Delete 删除 Idea
func (uc *IdeaUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
