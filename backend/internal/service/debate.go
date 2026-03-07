package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// DebateService 辩论服务
type DebateService struct {
	debateUc *biz.DebateUsecase
	log      *log.Helper

	// SSE 订阅管理
	mu          sync.RWMutex
	subscribers map[int64][]chan biz.DebateEvent
}

// NewDebateService 创建辩论服务
func NewDebateService(debateUc *biz.DebateUsecase, logger log.Logger) *DebateService {
	return &DebateService{
		debateUc:    debateUc,
		log:         log.NewHelper(logger),
		subscribers: make(map[int64][]chan biz.DebateEvent),
	}
}

// RegisterHTTPRoutes 注册路由
func (s *DebateService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.POST("/api/v1/debate/{id}/start", s.StartDebate)
	r.POST("/api/v1/debate/{id}/intervene", s.Intervene)
	r.GET("/api/v1/debate/{id}/status", s.DebateStatus)
	r.GET("/api/v1/debate/{id}/stream", s.DebateStream)
}

// Intervene 人工介入
func (s *DebateService) Intervene(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		ExtraRounds int    `json:"extra_rounds"`
		Prompt      string `json:"prompt"`
	}
	if err := ctx.Bind(&req); err != nil {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Prompt == "" {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
	}
	if req.ExtraRounds <= 0 {
		req.ExtraRounds = 3
	}
	if req.ExtraRounds > 20 {
		req.ExtraRounds = 20
	}

	eventCh := make(chan biz.DebateEvent, 100)
	if err := s.debateUc.Intervene(context.Background(), id, req.ExtraRounds, req.Prompt, eventCh); err != nil {
		return writeJSON(ctx, http.StatusConflict, map[string]string{"error": err.Error()})
	}

	go s.broadcastEvents(id, eventCh)

	return writeJSON(ctx, http.StatusOK, map[string]interface{}{
		"message":      "intervention started",
		"idea_id":      id,
		"extra_rounds": req.ExtraRounds,
	})
}

// StartDebate 启动辩论
func (s *DebateService) StartDebate(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 创建事件通道
	eventCh := make(chan biz.DebateEvent, 100)

	// 启动辩论
	if err := s.debateUc.StartDebate(context.Background(), id, eventCh); err != nil {
		return writeJSON(ctx, http.StatusConflict, map[string]string{"error": err.Error()})
	}

	// 在后台广播事件给 SSE 订阅者并处理通知
	go s.broadcastEvents(id, eventCh)

	return writeJSON(ctx, http.StatusOK, map[string]interface{}{
		"message": "debate started",
		"idea_id": id,
	})
}

// DebateStatus 获取辩论状态
func (s *DebateService) DebateStatus(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	running := s.debateUc.IsRunning(id)
	return writeJSON(ctx, http.StatusOK, map[string]interface{}{
		"idea_id": id,
		"running": running,
	})
}

// DebateStream SSE 实时推送
func (s *DebateService) DebateStream(ctx kratoshttp.Context) error {
	idStr := ctx.Vars().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 设置 SSE headers
	w := ctx.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return writeJSON(ctx, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
	}

	// 创建订阅通道
	ch := make(chan biz.DebateEvent, 100)
	s.subscribe(id, ch)
	defer s.unsubscribe(id, ch)

	// 发送初始连接事件
	connectEvent, _ := json.Marshal(biz.DebateEvent{
		Type:    "connected",
		IdeaID:  id,
		Content: "SSE connection established",
	})
	fmt.Fprintf(w, "data: %s\n\n", connectEvent)
	flusher.Flush()

	// 如果辩论未在运行，发送提示
	if !s.debateUc.IsRunning(id) {
		notRunning, _ := json.Marshal(biz.DebateEvent{
			Type:    "info",
			IdeaID:  id,
			Content: "No active debate for this idea. Start a debate first via POST /api/v1/debate/{id}/start",
		})
		fmt.Fprintf(w, "data: %s\n\n", notRunning)
		flusher.Flush()
	}

	// 持续推送事件
	reqCtx := ctx.Request().Context()
	for {
		select {
		case <-reqCtx.Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				// 通道关闭，辩论结束
				doneEvent, _ := json.Marshal(biz.DebateEvent{
					Type:    "stream_end",
					IdeaID:  id,
					Content: "debate stream ended",
				})
				fmt.Fprintf(w, "data: %s\n\n", doneEvent)
				flusher.Flush()
				return nil
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// broadcastEvents 广播事件给所有订阅者并处理通知
func (s *DebateService) broadcastEvents(ideaID int64, eventCh <-chan biz.DebateEvent) {
	for evt := range eventCh {
		s.mu.RLock()
		subs := s.subscribers[ideaID]
		s.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- evt:
			default:
				// 订阅者通道满了，跳过
			}
		}

		// 飞书通知已通过 DebateUsecase.onComplete 回调统一处理，无需在此重复
	}

	// 辩论结束，关闭所有订阅者通道
	s.mu.Lock()
	subs := s.subscribers[ideaID]
	for _, ch := range subs {
		close(ch)
	}
	delete(s.subscribers, ideaID)
	s.mu.Unlock()
}

func (s *DebateService) subscribe(ideaID int64, ch chan biz.DebateEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[ideaID] = append(s.subscribers[ideaID], ch)
}

func (s *DebateService) unsubscribe(ideaID int64, ch chan biz.DebateEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs := s.subscribers[ideaID]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[ideaID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func writeJSON(ctx kratoshttp.Context, status int, v interface{}) error {
	ctx.Response().Header().Set("Content-Type", "application/json")
	ctx.Response().WriteHeader(status)
	return json.NewEncoder(ctx.Response()).Encode(v)
}
