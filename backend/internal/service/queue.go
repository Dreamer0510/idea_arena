package service

import (
	"net/http"
	"strconv"

	"idea_arena/internal/biz"
	mw "idea_arena/internal/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// QueueService 手动队列 HTTP 服务
type QueueService struct {
	uc  *biz.QueueUsecase
	log *log.Helper
}

// NewQueueService 创建 QueueService
func NewQueueService(uc *biz.QueueUsecase, logger log.Logger) *QueueService {
	return &QueueService{uc: uc, log: log.NewHelper(logger)}
}

// RegisterHTTPRoutes 注册 HTTP 路由
func (s *QueueService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.POST("/api/v1/queue/submit", s.SubmitTopic)
	r.GET("/api/v1/queue/list", s.ListQueue)
}

type submitTopicRequest struct {
	Topic string `json:"topic"`
}

func (s *QueueService) SubmitTopic(ctx kratoshttp.Context) error {
	var req submitTopicRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Topic == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "topic is required"})
	}

	var submittedBy int64
	if claims := mw.AuthClaims(ctx); claims != nil {
		submittedBy = claims.UserID
	}

	item, err := s.uc.Submit(ctx, req.Topic, submittedBy)
	if err != nil {
		return ctx.Result(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	return ctx.Result(http.StatusCreated, map[string]interface{}{
		"id":      item.ID,
		"message": "topic submitted successfully",
	})
}

func (s *QueueService) ListQueue(ctx kratoshttp.Context) error {
	r := ctx.Request()
	query := &biz.QueueListQuery{
		Status: r.URL.Query().Get("status"),
	}
	if p, _ := strconv.Atoi(r.URL.Query().Get("page")); p > 0 {
		query.Page = p
	} else {
		query.Page = 1
	}
	if ps, _ := strconv.Atoi(r.URL.Query().Get("page_size")); ps > 0 {
		query.PageSize = ps
	} else {
		query.PageSize = 20
	}

	result, err := s.uc.List(ctx, query)
	if err != nil {
		return err
	}

	return ctx.Result(http.StatusOK, result)
}
