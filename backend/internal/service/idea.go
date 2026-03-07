package service

import (
	"net/http"
	"strconv"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

var cst = time.FixedZone("CST", 8*60*60)

// IdeaService Ideas HTTP 服务
type IdeaService struct {
	uc  *biz.IdeaUsecase
	log *log.Helper
}

// NewIdeaService 创建 IdeaService
func NewIdeaService(uc *biz.IdeaUsecase, logger log.Logger) *IdeaService {
	return &IdeaService{uc: uc, log: log.NewHelper(logger)}
}

// RegisterHTTPRoutes 注册 HTTP 路由
func (s *IdeaService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.GET("/api/v1/ideas", s.ListIdeas)
	r.GET("/api/v1/ideas/{id}", s.GetIdea)
	r.POST("/api/v1/ideas", s.CreateIdea)
	r.PUT("/api/v1/ideas/{id}", s.UpdateIdea)
	r.DELETE("/api/v1/ideas/{id}", s.DeleteIdea)
}

type listIdeasResponse struct {
	Items    []*ideaInfoResponse `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type ideaInfoResponse struct {
	ID               int64    `json:"id"`
	Topic            string   `json:"topic"`
	Status           string   `json:"status"`
	ProductName      string   `json:"product_name"`
	OneLiner         string   `json:"one_liner"`
	ScoreFeasibility float64  `json:"score_feasibility"`
	ScoreEconomics   float64  `json:"score_economics"`
	ScoreProfit      float64  `json:"score_profit"`
	ScoreOverall     float64  `json:"score_overall"`
	RoundCount       int      `json:"round_count"`
	Tags             []string `json:"tags"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type ideaDetailResponse struct {
	ideaInfoResponse
	Proposal     string `json:"proposal"`
	DebateLog    string `json:"debate_log"`
	FinalReport  string `json:"final_report"`
	TechStack    string `json:"tech_stack"`
	DevPrompt    string `json:"dev_prompt"`
	SearchData   string `json:"search_data"`
	JudgeRefined string `json:"judge_refined"`
}

type createIdeaRequest struct {
	Topic string `json:"topic"`
}

func (s *IdeaService) ListIdeas(ctx kratoshttp.Context) error {
	r := ctx.Request()
	query := &biz.IdeaListQuery{
		Status:    r.URL.Query().Get("status"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
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
	if ms, err := strconv.ParseFloat(r.URL.Query().Get("min_score"), 64); err == nil && ms > 0 {
		query.MinScore = ms
	}

	result, err := s.uc.List(ctx, query)
	if err != nil {
		return err
	}

	items := make([]*ideaInfoResponse, len(result.Items))
	for i, idea := range result.Items {
		items[i] = toIdeaInfoResponse(idea)
	}

	return ctx.Result(http.StatusOK, &listIdeasResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func (s *IdeaService) GetIdea(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	idea, err := s.uc.Get(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "idea not found"})
	}

	return ctx.Result(http.StatusOK, toIdeaDetailResponse(idea))
}

func (s *IdeaService) CreateIdea(ctx kratoshttp.Context) error {
	var req createIdeaRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Topic == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "topic is required"})
	}

	idea, err := s.uc.Create(ctx, req.Topic)
	if err != nil {
		return err
	}

	return ctx.Result(http.StatusCreated, map[string]interface{}{
		"id":     idea.ID,
		"topic":  idea.Topic,
		"status": idea.Status,
	})
}

func (s *IdeaService) UpdateIdea(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		Topic  string `json:"topic"`
		Status string `json:"status"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	existing, err := s.uc.Get(ctx, id)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "idea not found"})
	}

	if req.Topic != "" {
		existing.Topic = req.Topic
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := s.uc.Update(ctx, existing); err != nil {
		return err
	}

	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *IdeaService) DeleteIdea(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := s.uc.Delete(ctx, id); err != nil {
		return err
	}

	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func toIdeaInfoResponse(idea *biz.Idea) *ideaInfoResponse {
	return &ideaInfoResponse{
		ID:               idea.ID,
		Topic:            idea.Topic,
		Status:           idea.Status,
		ProductName:      idea.ProductName,
		OneLiner:         idea.OneLiner,
		ScoreFeasibility: idea.ScoreFeasibility,
		ScoreEconomics:   idea.ScoreEconomics,
		ScoreProfit:      idea.ScoreProfit,
		ScoreOverall:     idea.ScoreOverall,
		RoundCount:       idea.RoundCount,
		Tags:             idea.Tags,
		CreatedAt:        idea.CreatedAt.In(cst).Format("2006-01-02T15:04:05+08:00"),
		UpdatedAt:        idea.UpdatedAt.In(cst).Format("2006-01-02T15:04:05+08:00"),
	}
}

func toIdeaDetailResponse(idea *biz.Idea) *ideaDetailResponse {
	return &ideaDetailResponse{
		ideaInfoResponse: *toIdeaInfoResponse(idea),
		Proposal:         idea.Proposal,
		DebateLog:        idea.DebateLog,
		FinalReport:      idea.FinalReport,
		TechStack:        idea.TechStack,
		DevPrompt:        idea.DevPrompt,
		SearchData:       idea.SearchData,
		JudgeRefined:     idea.JudgeRefined,
	}
}
