package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"idea_arena/internal/conf"
	"idea_arena/internal/pkg/llm"
	"idea_arena/internal/pkg/search"

	"github.com/go-kratos/kratos/v2/log"
)

// TodoItem 辩论 TODO 条目
type TodoItem struct {
	ID            string  `json:"id"`
	Issue         string  `json:"issue"`
	Detail        string  `json:"detail"`
	Priority      string  `json:"priority"`       // P0, P1, P2
	Status        string  `json:"status"`          // open, resolved, wontfix
	CreatedBy     string  `json:"created_by"`      // opponent, proposer, referee
	CreatedRound  int     `json:"created_round"`
	ResolvedRound *int    `json:"resolved_round"`
	Resolution    *string `json:"resolution"`
}

// TodoList 完整 TODO LIST
type TodoList struct {
	Todos []TodoItem `json:"todos"`
}

// DebateRound 辩论轮次记录
type DebateRound struct {
	Round              int     `json:"round"`
	Proposer           string  `json:"proposer"`
	Opponent           string  `json:"opponent"`
	Referee            string  `json:"referee,omitempty"`
	Feasibility        float64 `json:"feasibility"`
	Economics          float64 `json:"economics"`
	Profit             float64 `json:"profit"`
	Overall            float64 `json:"overall"`
	EvalReason         string  `json:"eval_reason"`
	TodoSnapshot       string  `json:"todo_snapshot,omitempty"`
	Timestamp          string  `json:"timestamp"`
	HumanIntervention  string  `json:"human_intervention,omitempty"`
}

// DebateLog 完整辩论日志
type DebateLog struct {
	Rounds       []DebateRound `json:"rounds"`
	SearchData   string        `json:"search_data"`
	Summary      string        `json:"summary"`
	JudgeRefined string        `json:"judge_refined"`
}

// DebateEvent SSE 事件
type DebateEvent struct {
	Type    string      `json:"type"`    // search, proposal, opponent, eval, judge, meta, summary, system, done, error
	IdeaID  int64       `json:"idea_id"`
	Round   int         `json:"round,omitempty"`
	Agent   string      `json:"agent,omitempty"`
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// DebateCompleteFunc 辩论完成回调函数类型
type DebateCompleteFunc func(ctx context.Context, ideaID int64, finalStatus string, scores [4]float64)

// DebateUsecase 辩论引擎业务用例
type DebateUsecase struct {
	ideaRepo   IdeaRepo
	llmClient  *llm.Client
	searcher   *search.Aggregator
	debateConf *conf.Debate
	log        *log.Helper

	// 辩论完成回调（用于飞书通知等）
	onComplete DebateCompleteFunc

	// 多服务商支持
	providerManager *llm.ProviderManager
	agentRepo       AgentConfigRepository
	providerRepo    ProviderRepository

	// 并发控制
	semaphore chan struct{}
	mu        sync.Mutex
	running   map[int64]context.CancelFunc
}

// NewDebateUsecase 创建辩论引擎
func NewDebateUsecase(
	ideaRepo IdeaRepo,
	llmClient *llm.Client,
	searcher *search.Aggregator,
	debateConf *conf.Debate,
	logger log.Logger,
) *DebateUsecase {
	maxConcurrent := 2
	if debateConf.MaxConcurrent > 0 {
		maxConcurrent = debateConf.MaxConcurrent
	}
	return &DebateUsecase{
		ideaRepo:   ideaRepo,
		llmClient:  llmClient,
		searcher:   searcher,
		debateConf: debateConf,
		log:        log.NewHelper(logger),
		semaphore:  make(chan struct{}, maxConcurrent),
		running:    make(map[int64]context.CancelFunc),
	}
}

// SetOnComplete 设置辩论完成回调
func (uc *DebateUsecase) SetOnComplete(fn DebateCompleteFunc) {
	uc.onComplete = fn
	uc.log.Infof("[Debate] onComplete callback registered (fn=%v)", fn != nil)
}

// AgentConfigRepository Agent配置仓储接口
type AgentConfigRepository interface {
	GetByID(ctx context.Context, id string) (*AgentConfigDTO, error)
	List(ctx context.Context) ([]*AgentConfigDTO, error)
}

// ProviderRepository 服务商仓储接口
type ProviderRepository interface {
	GetByID(ctx context.Context, id int64) (*ProviderDTO, error)
	GetDefault(ctx context.Context) (*ProviderDTO, error)
}

// AgentConfigDTO Agent配置 DTO
type AgentConfigDTO struct {
	ID           string
	Name         string
	Emoji        string
	Role         string
	SystemPrompt string
	Temperature  float64
	ProviderID   int64
	ModelName    string
	Enabled      bool
}

// ProviderDTO 服务商 DTO
type ProviderDTO struct {
	ID      int64
	Name    string
	BaseURL string
	APIKey  string
}

// SetProviderManager 注入多服务商管理器
func (uc *DebateUsecase) SetProviderManager(pm *llm.ProviderManager, agentRepo AgentConfigRepository, providerRepo ProviderRepository) {
	uc.providerManager = pm
	uc.agentRepo = agentRepo
	uc.providerRepo = providerRepo
	uc.log.Info("[Debate] ProviderManager injected")
}

// callAgent 智能调用 Agent：优先使用服务商系统，回退到旧的 config.yaml 配置
func (uc *DebateUsecase) callAgent(ctx context.Context, agentID string, defaultAgent *llm.Agent, messages []llm.Message) (string, error) {
	// 尝试从 DB 获取 Agent 配置
	if uc.agentRepo != nil && uc.providerManager != nil {
		agentCfg, err := uc.agentRepo.GetByID(ctx, agentID)
		if err == nil && agentCfg != nil && agentCfg.ProviderID > 0 && agentCfg.ModelName != "" {
			// 使用 DB 配置的服务商+模型
			systemPrompt := agentCfg.SystemPrompt
			if systemPrompt == "" && defaultAgent != nil {
				systemPrompt = defaultAgent.SystemPrompt
			}
			opts := &llm.CallOptions{Temperature: agentCfg.Temperature}
			result, err := uc.providerManager.CallWithProvider(ctx, agentCfg.ProviderID, agentCfg.ModelName, systemPrompt, messages, opts)
			if err != nil {
				uc.log.Warnf("[Debate] Provider call failed for agent %s (provider=%d model=%s): %v, falling back to default",
					agentID, agentCfg.ProviderID, agentCfg.ModelName, err)
			} else {
				return result, nil
			}
		}
	}

	// 回退到旧的 config.yaml 配置
	if defaultAgent == nil {
		return "", fmt.Errorf("agent %s not found and no default available", agentID)
	}
	return uc.llmClient.CallWithRole(ctx, defaultAgent.Role, defaultAgent.SystemPrompt, messages)
}

// StartDebate 启动辩论（异步，通过 eventCh 推送事件）
func (uc *DebateUsecase) StartDebate(ctx context.Context, ideaID int64, eventCh chan<- DebateEvent) error {
	// 检查 idea 是否存在
	idea, err := uc.ideaRepo.GetByID(ctx, ideaID)
	if err != nil {
		return fmt.Errorf("idea not found: %w", err)
	}

	// 检查状态
	if idea.Status == "debating" {
		return fmt.Errorf("idea %d is already debating", ideaID)
	}

	// 获取并发信号量
	select {
	case uc.semaphore <- struct{}{}:
	default:
		return fmt.Errorf("max concurrent debates reached, please wait")
	}

	// 创建可取消的上下文
	debateCtx, cancel := context.WithTimeout(ctx, uc.debateConf.GetTimeout())

	uc.mu.Lock()
	uc.running[ideaID] = cancel
	uc.mu.Unlock()

	// 更新状态为辩论中
	_ = uc.ideaRepo.UpdateStatus(ctx, ideaID, "debating")

	// 异步执行辩论
	go func() {
		defer func() {
			<-uc.semaphore
			cancel()
			uc.mu.Lock()
			delete(uc.running, ideaID)
			uc.mu.Unlock()
			close(eventCh)
		}()
		uc.runDebate(debateCtx, idea, eventCh)
	}()

	return nil
}

// Intervene 人工介入：基于已有辩论状态，注入人工提示词，继续额外轮次辩论
// 注意：人工介入不受并发信号量限制，优先级高于自动辩论
func (uc *DebateUsecase) Intervene(ctx context.Context, ideaID int64, extraRounds int, humanPrompt string, eventCh chan<- DebateEvent) error {
	idea, err := uc.ideaRepo.GetByID(ctx, ideaID)
	if err != nil {
		return fmt.Errorf("idea not found: %w", err)
	}
	if idea.Status == "debating" {
		return fmt.Errorf("idea %d is already debating", ideaID)
	}

	debateCtx, cancel := context.WithTimeout(ctx, uc.debateConf.GetTimeout())
	uc.mu.Lock()
	uc.running[ideaID] = cancel
	uc.mu.Unlock()

	_ = uc.ideaRepo.UpdateStatus(ctx, ideaID, "debating")

	go func() {
		defer func() {
			cancel()
			uc.mu.Lock()
			delete(uc.running, ideaID)
			uc.mu.Unlock()
			close(eventCh)
		}()
		uc.runIntervention(debateCtx, idea, extraRounds, humanPrompt, eventCh)
	}()

	return nil
}

// runIntervention 执行人工介入后的续辩流程
func (uc *DebateUsecase) runIntervention(ctx context.Context, idea *Idea, extraRounds int, humanPrompt string, eventCh chan<- DebateEvent) {
	ideaID := idea.ID
	topic := idea.Topic
	proposal := idea.Proposal

	sendEvent := func(evt DebateEvent) {
		evt.IdeaID = ideaID
		select {
		case eventCh <- evt:
		case <-ctx.Done():
		}
	}

	// 加载已有辩论日志
	var debateLog DebateLog
	if idea.DebateLog != "" {
		_ = json.Unmarshal([]byte(idea.DebateLog), &debateLog)
	}

	// 恢复上一轮状态
	var lastScores [4]float64
	var todoList TodoList
	startRound := len(debateLog.Rounds) + 1
	if len(debateLog.Rounds) > 0 {
		last := debateLog.Rounds[len(debateLog.Rounds)-1]
		lastScores = [4]float64{last.Feasibility, last.Economics, last.Profit, last.Overall}
		if last.TodoSnapshot != "" {
			_ = json.Unmarshal([]byte(last.TodoSnapshot), &todoList)
		}
	}

	searchContext := debateLog.SearchData

	sendEvent(DebateEvent{Type: "system", Content: fmt.Sprintf("🧑‍💻 人工介入：将进行 %d 轮额外辩论\n指令：%s", extraRounds, humanPrompt)})

	graduationScore := 7.5
	if uc.debateConf.GraduationScore > 0 {
		graduationScore = uc.debateConf.GraduationScore
	}

	humanDirective := fmt.Sprintf("\n\n⚠️ 【人工介入指令】以下是产品负责人的直接反馈，请务必优先处理：\n%s", humanPrompt)

	for round := startRound; round < startRound+extraRounds; round++ {
		select {
		case <-ctx.Done():
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("intervention debate timeout"))
			return
		default:
		}

		todoJSON := todoListToJSON(todoList)

		// ===== 1. 审判官审查（注入人工指令）=====
		sendEvent(DebateEvent{Type: "opponent", Round: round, Agent: llm.AgentOpponent.Emoji,
			Content: fmt.Sprintf("第 %d 轮（人工介入）— 审判官正在审查...", round)})

		opponentPrompt := fmt.Sprintf("以下是最新的搜索数据供你参考：\n\n%s\n\n当前 TODO LIST：\n%s\n\n这是第 %d 轮审查（人工介入续辩）。%s\n\n请：\n1. 充分考虑产品负责人的反馈\n2. 检查创想者对每个 open TODO 的回应质量\n3. 发现新问题时新增 TODO\n4. 在输出最后附上 [TODO_CHANGES] JSON 块", searchContext, todoJSON, round, humanDirective)

		opponentHistory := []llm.Message{
			{Role: "user", Content: fmt.Sprintf("话题：%s\n\n创想者的最新方案：\n%s", topic, proposal)},
			{Role: "user", Content: opponentPrompt},
		}

		opponentResponse, err := uc.callAgent(ctx, "opponent", llm.AgentOpponent, opponentHistory)
		if err != nil {
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("round %d opponent failed: %w", round, err))
			return
		}
		sendEvent(DebateEvent{Type: "opponent", Round: round, Agent: llm.AgentOpponent.Emoji, Content: opponentResponse})

		// ===== 2. 创想者回应（注入人工指令）=====
		sendEvent(DebateEvent{Type: "proposal", Round: round, Agent: llm.AgentProposer.Emoji,
			Content: fmt.Sprintf("第 %d 轮（人工介入）— 创想者正在回应...", round)})

		proposerPrompt := fmt.Sprintf("审判官的第 %d 轮审查意见：\n%s\n\n当前 TODO LIST：\n%s\n\n当前评分：可行性=%.1f 经济性=%.1f 利润潜力=%.1f 综合=%.1f%s\n\n请逐个回应 open 状态的 TODO，然后输出改进后的完整方案。在输出最后附上 [TODO_RESPONSES] JSON 块。", round, opponentResponse, todoJSON, lastScores[0], lastScores[1], lastScores[2], lastScores[3], humanDirective)

		proposerHistory := []llm.Message{
			{Role: "user", Content: fmt.Sprintf("话题：%s\n\n你之前的方案：\n%s", topic, proposal)},
			{Role: "user", Content: proposerPrompt},
		}

		proposerResponse, err := uc.callAgent(ctx, "proposer", llm.AgentProposer, proposerHistory)
		if err != nil {
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("round %d proposer failed: %w", round, err))
			return
		}

		proposal = extractProposalText(proposerResponse)
		sendEvent(DebateEvent{Type: "proposal", Round: round, Agent: llm.AgentProposer.Emoji, Content: proposerResponse})

		// ===== 3. 裁判裁定 =====
		sendEvent(DebateEvent{Type: "eval", Round: round, Agent: llm.AgentReferee.Emoji,
			Content: fmt.Sprintf("第 %d 轮 — 裁判正在评分和裁定 TODO...", round)})

		refereePrompt := fmt.Sprintf("话题：%s\n\n当前 TODO LIST：\n%s\n\n审判官的审查意见：\n%s\n\n创想者的回应和改进方案：\n%s\n\n请给出评分（[EVAL]行）和 TODO 裁定（[TODO_VERDICT] JSON）。", topic, todoJSON, opponentResponse, proposerResponse)

		refereeHistory := []llm.Message{
			{Role: "user", Content: refereePrompt},
		}
		refereeResponse, err := uc.callAgent(ctx, "referee", llm.AgentReferee, refereeHistory)
		if err != nil {
			uc.log.Warnf("round %d referee failed: %v", round, err)
			refereeResponse = "[EVAL] feasibility=5 economics=5 profit=5 overall=5 reason=裁判评估失败"
		}

		scores := parseEvalScores(refereeResponse)
		lastScores = scores

		if verdictList := parseTodoVerdict(refereeResponse); len(verdictList.Todos) > 0 {
			todoList = verdictList
		}

		todoSnapshotJSON := todoListToJSON(todoList)
		sendEvent(DebateEvent{Type: "eval", Round: round, Agent: llm.AgentReferee.Emoji,
			Content: refereeResponse,
			Data: map[string]interface{}{
				"feasibility":  scores[0],
				"economics":    scores[1],
				"profit":       scores[2],
				"overall":      scores[3],
				"todo_snapshot": todoSnapshotJSON,
			}})

		debateRound := DebateRound{
			Round:        round,
			Proposer:     proposerResponse,
			Opponent:     opponentResponse,
			Referee:      refereeResponse,
			Feasibility:  scores[0],
			Economics:    scores[1],
			Profit:       scores[2],
			Overall:      scores[3],
			EvalReason:   extractReason(refereeResponse),
			TodoSnapshot: todoSnapshotJSON,
			Timestamp:    time.Now().Format(time.RFC3339),
		}
		if round == startRound {
			debateRound.HumanIntervention = humanPrompt
		}
		debateLog.Rounds = append(debateLog.Rounds, debateRound)

		logJSON, _ := json.Marshal(debateLog)
		_ = uc.ideaRepo.UpdateDebateLog(ctx, ideaID, string(logJSON), round)
		_ = uc.ideaRepo.UpdateScores(ctx, ideaID, scores[0], scores[1], scores[2], scores[3])
		_ = uc.ideaRepo.Update(ctx, &Idea{ID: ideaID, Proposal: proposal})
	}

	// 更新最终状态
	var finalStatus string
	switch {
	case lastScores[3] >= graduationScore:
		finalStatus = "graduated"
	case lastScores[3] >= 5.5:
		finalStatus = "promising"
	default:
		finalStatus = "failed"
	}

	logJSON, _ := json.Marshal(debateLog)
	_ = uc.ideaRepo.Update(ctx, &Idea{
		ID:        ideaID,
		Status:    finalStatus,
		DebateLog: string(logJSON),
	})

	sendEvent(DebateEvent{Type: "done", Content: fmt.Sprintf("人工介入辩论完成！最终评分: %.1f, 状态: %s", lastScores[3], finalStatus),
		Data: map[string]interface{}{
			"status":      finalStatus,
			"overall":     lastScores[3],
			"round_count": len(debateLog.Rounds),
		}})

	if uc.onComplete != nil {
		go uc.onComplete(context.Background(), ideaID, finalStatus, lastScores)
	}
}

// IsRunning 检查辩论是否正在进行
func (uc *DebateUsecase) IsRunning(ideaID int64) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	_, ok := uc.running[ideaID]
	return ok
}

// runDebate 执行完整辩论流程
func (uc *DebateUsecase) runDebate(ctx context.Context, idea *Idea, eventCh chan<- DebateEvent) {
	ideaID := idea.ID
	topic := idea.Topic

	sendEvent := func(evt DebateEvent) {
		evt.IdeaID = ideaID
		select {
		case eventCh <- evt:
		case <-ctx.Done():
		}
	}

	// ========== Step 1: 多源搜索 ==========
	sendEvent(DebateEvent{Type: "search", Agent: "🔍", Content: "正在进行多源搜索..."})

	_, searchContext := uc.searcher.MultiSourceSearch(ctx, topic)
	_ = uc.ideaRepo.Update(ctx, &Idea{ID: ideaID, SearchData: searchContext})

	sendEvent(DebateEvent{Type: "search", Agent: "🔍", Content: "搜索完成"})

	// ========== Step 2: 初始提案 ==========
	sendEvent(DebateEvent{Type: "proposal", Round: 1, Agent: llm.AgentProposer.Emoji, Content: "创想者正在构思初始提案..."})

	proposalPrompt := fmt.Sprintf("话题方向：%s\n\n以下是搜索到的最新信息，请基于这些信息提出你的初始提案：\n\n%s", topic, searchContext)
	proposal, err := uc.callAgent(ctx, "proposer", llm.AgentProposer, []llm.Message{
		{Role: "user", Content: proposalPrompt},
	})
	if err != nil {
		uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("initial proposal failed: %w", err))
		return
	}

	_ = uc.ideaRepo.Update(ctx, &Idea{ID: ideaID, Proposal: proposal})
	sendEvent(DebateEvent{Type: "proposal", Round: 1, Agent: llm.AgentProposer.Emoji, Content: proposal})

	// ========== Step 3: 产品元数据提取 ==========
	sendEvent(DebateEvent{Type: "meta", Agent: llm.AgentMetaExtractor.Emoji, Content: "正在提取产品元数据..."})
	go uc.extractMeta(ctx, ideaID, proposal)

	// ========== Step 4: 多轮辩论 ==========
	maxRounds := 20
	if uc.debateConf.MaxRounds > 0 {
		maxRounds = uc.debateConf.MaxRounds
	}
	graduationScore := 7.5
	if uc.debateConf.GraduationScore > 0 {
		graduationScore = uc.debateConf.GraduationScore
	}

	var debateLog DebateLog
	debateLog.SearchData = searchContext

	var lastScores [4]float64 // feasibility, economics, profit, overall
	var bestOverall float64
	var declineCount int
	var firstRoundScore float64 // 记录第一轮分数，用于判断是否强制跑满
	var todoList TodoList       // 当前 TODO LIST（由裁判权威管理）

	for round := 1; round <= maxRounds; round++ {
		select {
		case <-ctx.Done():
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("debate timeout"))
			return
		default:
		}

		// ===== 1. 审判官审查 =====
		sendEvent(DebateEvent{Type: "opponent", Round: round, Agent: llm.AgentOpponent.Emoji,
			Content: fmt.Sprintf("第 %d 轮 — 毒舌审判官正在审查...", round)})

		todoJSON := todoListToJSON(todoList)
		var opponentPrompt string
		if round == 1 {
			opponentPrompt = fmt.Sprintf("以下是最新的搜索数据供你参考：\n\n%s\n\n请审查以上方案，找出所有关键问题，创建初始 TODO LIST。对每个问题标记优先级（P0/P1/P2）并给出改进建议。\n\n在输出最后，请附上 [TODO_CHANGES] JSON 块。", searchContext)
		} else {
			opponentPrompt = fmt.Sprintf("以下是最新的搜索数据供你参考：\n\n%s\n\n当前 TODO LIST：\n%s\n\n这是第 %d 轮审查。请：\n1. 检查创想者对每个 open TODO 的回应质量\n2. 对 disputed 的 TODO 做出判断\n3. 发现新问题时新增 TODO\n4. 在输出最后附上 [TODO_CHANGES] JSON 块", searchContext, todoJSON, round)
		}

		opponentHistory := []llm.Message{
			{Role: "user", Content: fmt.Sprintf("话题：%s\n\n创想者的最新方案：\n%s", topic, proposal)},
			{Role: "user", Content: opponentPrompt},
		}

		opponentResponse, err := uc.callAgent(ctx, "opponent", llm.AgentOpponent, opponentHistory)
		if err != nil {
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("round %d opponent failed: %w", round, err))
			return
		}
		sendEvent(DebateEvent{Type: "opponent", Round: round, Agent: llm.AgentOpponent.Emoji, Content: opponentResponse})

		// ===== 2. 创想者回应 =====
		sendEvent(DebateEvent{Type: "proposal", Round: round, Agent: llm.AgentProposer.Emoji,
			Content: fmt.Sprintf("第 %d 轮 — 创想者正在回应并改进方案...", round)})

		var proposerPrompt string
		if round == 1 {
			proposerPrompt = fmt.Sprintf("审判官对你的初始方案提出了以下审查意见，并创建了 TODO LIST：\n\n%s\n\n请逐个回应 TODO 中的问题，然后输出改进后的完整方案。在输出最后附上 [TODO_RESPONSES] JSON 块。", opponentResponse)
		} else {
			proposerPrompt = fmt.Sprintf("审判官的第 %d 轮审查意见：\n%s\n\n当前 TODO LIST：\n%s\n\n当前评分：可行性=%.1f 经济性=%.1f 利润潜力=%.1f 综合=%.1f\n\n请逐个回应 open 状态的 TODO，然后输出改进后的完整方案。在输出最后附上 [TODO_RESPONSES] JSON 块。", round, opponentResponse, todoJSON, lastScores[0], lastScores[1], lastScores[2], lastScores[3])
		}

		proposerHistory := []llm.Message{
			{Role: "user", Content: fmt.Sprintf("话题：%s\n\n你之前的方案：\n%s", topic, proposal)},
			{Role: "user", Content: proposerPrompt},
		}

		proposerResponse, err := uc.callAgent(ctx, "proposer", llm.AgentProposer, proposerHistory)
		if err != nil {
			uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("round %d proposer failed: %w", round, err))
			return
		}

		// 更新 proposal 为创想者最新输出（去掉尾部 JSON 块后的方案正文）
		proposal = extractProposalText(proposerResponse)
		sendEvent(DebateEvent{Type: "proposal", Round: round, Agent: llm.AgentProposer.Emoji, Content: proposerResponse})

		// ===== 3. 裁判裁定 =====
		sendEvent(DebateEvent{Type: "eval", Round: round, Agent: llm.AgentReferee.Emoji,
			Content: fmt.Sprintf("第 %d 轮 — 裁判正在评分和裁定 TODO...", round)})

		refereePrompt := fmt.Sprintf("话题：%s\n\n当前 TODO LIST：\n%s\n\n审判官的审查意见：\n%s\n\n创想者的回应和改进方案：\n%s\n\n请给出评分（[EVAL]行）和 TODO 裁定（[TODO_VERDICT] JSON）。", topic, todoJSON, opponentResponse, proposerResponse)

		refereeHistory := []llm.Message{
			{Role: "user", Content: refereePrompt},
		}
		refereeResponse, err := uc.callAgent(ctx, "referee", llm.AgentReferee, refereeHistory)
		if err != nil {
			uc.log.Warnf("round %d referee failed: %v", round, err)
			refereeResponse = "[EVAL] feasibility=5 economics=5 profit=5 overall=5 reason=裁判评估失败"
		}

		// 解析评分
		scores := parseEvalScores(refereeResponse)
		lastScores = scores
		if round == 1 {
			firstRoundScore = scores[3]
		}

		// 解析 TODO 裁定并更新 todoList
		if verdictList := parseTodoVerdict(refereeResponse); len(verdictList.Todos) > 0 {
			todoList = verdictList
		} else if round == 1 {
			// 第一轮如果裁判没输出 TODO_VERDICT，从审判官的 TODO_CHANGES 构建初始列表
			todoList = buildInitialTodoFromOpponent(opponentResponse, round)
		}

		todoSnapshotJSON := todoListToJSON(todoList)
		sendEvent(DebateEvent{Type: "eval", Round: round, Agent: llm.AgentReferee.Emoji,
			Content: refereeResponse,
			Data: map[string]interface{}{
				"feasibility":   scores[0],
				"economics":     scores[1],
				"profit":        scores[2],
				"overall":       scores[3],
				"todo_snapshot":  todoSnapshotJSON,
			}})

		// 保存辩论轮次
		debateRound := DebateRound{
			Round:        round,
			Proposer:     proposerResponse,
			Opponent:     opponentResponse,
			Referee:      refereeResponse,
			Feasibility:  scores[0],
			Economics:    scores[1],
			Profit:       scores[2],
			Overall:      scores[3],
			EvalReason:   extractReason(refereeResponse),
			TodoSnapshot: todoSnapshotJSON,
			Timestamp:    time.Now().Format(time.RFC3339),
		}
		debateLog.Rounds = append(debateLog.Rounds, debateRound)

		logJSON, _ := json.Marshal(debateLog)
		_ = uc.ideaRepo.UpdateDebateLog(ctx, ideaID, string(logJSON), round)
		_ = uc.ideaRepo.UpdateScores(ctx, ideaID, scores[0], scores[1], scores[2], scores[3])

		// --- 检查是否毕业 ---
		if firstRoundScore >= 7.0 {
			if round == 1 {
				uc.log.Infof("Idea %d first round score %.1f >= 7.0, forcing full %d rounds debate", ideaID, firstRoundScore, maxRounds)
				sendEvent(DebateEvent{Type: "system", Content: fmt.Sprintf("⚡ 初始评分 %.1f 较高，将进行完整 %d 轮深度辩论", firstRoundScore, maxRounds)})
			}
		} else {
			minRounds := 3
			if uc.debateConf.MinRounds > 0 {
				minRounds = uc.debateConf.MinRounds
			}
			if scores[3] >= graduationScore && round >= minRounds {
				uc.log.Infof("Idea %d graduated at round %d with score %.1f (min_rounds=%d satisfied)", ideaID, round, scores[3], minRounds)
				break
			} else if scores[3] >= graduationScore {
				uc.log.Infof("Idea %d score %.1f >= %.1f but round %d < min_rounds %d, continuing debate", ideaID, scores[3], graduationScore, round, minRounds)
			}
		}

		// --- 跟踪最佳分数和连续下降 ---
		if scores[3] > bestOverall {
			bestOverall = scores[3]
			declineCount = 0
		} else {
			declineCount++
		}
		if firstRoundScore < 7.0 && declineCount >= 5 && bestOverall < 4.0 {
			uc.log.Infof("Idea %d early stop: scores declined 5 consecutive rounds and best=%.1f < 4.0", ideaID, bestOverall)
			break
		}
	}

	// ========== Step 5: 终极裁决 ==========
	sendEvent(DebateEvent{Type: "judge", Agent: llm.AgentJudge.Emoji, Content: "终极仲裁正在生成可行性报告..."})

	// 构建 Judge 输入：最终方案 + TODO 状态 + 最后一轮评分锚定
	todoSummary := buildTodoSummaryForJudge(todoList)
	judgeInput := fmt.Sprintf("话题：%s\n\n创想者的最终方案：\n%s\n\n辩论共进行了 %d 轮。\n\n最终 TODO LIST 状态：\n%s\n\n⚠️ 最后一轮裁判评分：可行性=%.1f 经济性=%.1f 利润潜力=%.1f 综合=%.1f\n请基于以上信息生成结构化可行性报告。",
		topic, proposal, len(debateLog.Rounds), todoSummary, lastScores[0], lastScores[1], lastScores[2], lastScores[3])

	judgeHistory := []llm.Message{
		{Role: "user", Content: judgeInput},
	}
	judgeResponse, err := uc.callAgent(ctx, "judge", llm.AgentJudge, judgeHistory)
	if err != nil {
		uc.handleError(ctx, ideaID, sendEvent, fmt.Errorf("judge failed: %w", err))
		return
	}

	// 从裁决报告中解析最终评分
	finalScores := parseJudgeScores(judgeResponse)
	if finalScores[3] > 0 {
		lastScores = finalScores
	}

	_ = uc.ideaRepo.Update(ctx, &Idea{
		ID:          ideaID,
		FinalReport: judgeResponse,
	})
	_ = uc.ideaRepo.UpdateScores(ctx, ideaID, lastScores[0], lastScores[1], lastScores[2], lastScores[3])

	sendEvent(DebateEvent{Type: "judge", Agent: llm.AgentJudge.Emoji, Content: judgeResponse})

	// ========== Step 6: 生成摘要 ==========
	sendEvent(DebateEvent{Type: "summary", Agent: llm.AgentDebateSummarizer.Emoji, Content: "生成辩论摘要..."})

	summaryHistory := []llm.Message{
		{Role: "user", Content: fmt.Sprintf("以下是关于\"%s\"的完整辩论过程，请生成精炼摘要：\n\n%s\n\n最终裁决报告：\n%s",
			idea.Topic, proposal, judgeResponse)},
	}
	summaryResponse, err := uc.callAgent(ctx, "debate_summarizer", llm.AgentDebateSummarizer, summaryHistory)
	if err != nil {
		summaryResponse = fmt.Sprintf("关于\"%s\"的辩论共进行了%d轮。", idea.Topic, len(debateLog.Rounds))
	}

	debateLog.Summary = summaryResponse

	// ========== Step 7: 保存最终状态（三档：graduated / promising / failed）==========
	var finalStatus string
	switch {
	case lastScores[3] >= graduationScore:
		finalStatus = "graduated"
	case lastScores[3] >= 5.5:
		finalStatus = "promising"
	default:
		finalStatus = "failed"
	}

	logJSON, _ := json.Marshal(debateLog)
	_ = uc.ideaRepo.Update(ctx, &Idea{
		ID:        ideaID,
		Status:    finalStatus,
		DebateLog: string(logJSON),
	})

	sendEvent(DebateEvent{Type: "done", Content: fmt.Sprintf("辩论完成！最终评分: %.1f, 状态: %s", lastScores[3], finalStatus),
		Data: map[string]interface{}{
			"status":      finalStatus,
			"overall":     lastScores[3],
			"round_count": len(debateLog.Rounds),
		}})

	// 触发辩论完成回调（飞书通知等）
	uc.log.Infof("[Debate] Idea %d finished: status=%s, score=%.1f, onComplete=%v", ideaID, finalStatus, lastScores[3], uc.onComplete != nil)
	if uc.onComplete != nil {
		uc.log.Infof("[Debate] Triggering onComplete callback for Idea %d", ideaID)
		go uc.onComplete(context.Background(), ideaID, finalStatus, lastScores)
	}
}

// extractMeta 异步提取产品元数据
func (uc *DebateUsecase) extractMeta(ctx context.Context, ideaID int64, proposal string) {
	metaResponse, err := uc.callAgent(ctx, "meta_extractor", llm.AgentMetaExtractor, []llm.Message{
		{Role: "user", Content: proposal},
	})
	if err != nil {
		uc.log.Warnf("meta extraction failed for idea %d: %v", ideaID, err)
		return
	}

	// 解析 JSON
	var meta struct {
		ProductName    string `json:"product_name"`
		ProductDesc    string `json:"product_desc"`
		TargetMarket   string `json:"target_market"`
		TargetAudience string `json:"target_audience"`
	}

	// 清理可能的 markdown 代码块
	cleaned := metaResponse
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &meta); err != nil {
		uc.log.Warnf("meta parse failed for idea %d: %v, raw: %s", ideaID, err, metaResponse[:min(len(metaResponse), 200)])
		return
	}

	_ = uc.ideaRepo.Update(ctx, &Idea{
		ID:          ideaID,
		ProductName: meta.ProductName,
		OneLiner:    meta.ProductDesc,
	})
}

// handleError 处理辩论错误
func (uc *DebateUsecase) handleError(ctx context.Context, ideaID int64, sendEvent func(DebateEvent), err error) {
	uc.log.Errorf("debate error for idea %d: %v", ideaID, err)
	_ = uc.ideaRepo.UpdateStatus(context.Background(), ideaID, "failed")
	sendEvent(DebateEvent{Type: "error", Content: err.Error()})
}

// parseEvalScores 解析评估分数 [EVAL] feasibility=X economics=Y profit=Z overall=W reason=...
func parseEvalScores(response string) [4]float64 {
	var scores [4]float64
	scores = [4]float64{5, 5, 5, 5} // 默认

	re := regexp.MustCompile(`\[EVAL\]\s*feasibility=([\d.]+)\s+economics=([\d.]+)\s+profit=([\d.]+)\s+overall=([\d.]+)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) >= 5 {
		scores[0], _ = strconv.ParseFloat(matches[1], 64)
		scores[1], _ = strconv.ParseFloat(matches[2], 64)
		scores[2], _ = strconv.ParseFloat(matches[3], 64)
		scores[3], _ = strconv.ParseFloat(matches[4], 64)
	}

	return scores
}

// parseJudgeScores 解析终极裁决分数 [SCORES] feasibility=X economics=Y profit=Z overall=W verdict=...
func parseJudgeScores(response string) [4]float64 {
	var scores [4]float64

	re := regexp.MustCompile(`\[SCORES\]\s*feasibility=([\d.]+)\s+economics=([\d.]+)\s+profit=([\d.]+)\s+overall=([\d.]+)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) >= 5 {
		scores[0], _ = strconv.ParseFloat(matches[1], 64)
		scores[1], _ = strconv.ParseFloat(matches[2], 64)
		scores[2], _ = strconv.ParseFloat(matches[3], 64)
		scores[3], _ = strconv.ParseFloat(matches[4], 64)
	}

	return scores
}

// extractReason 从评估响应中提取 reason
func extractReason(response string) string {
	re := regexp.MustCompile(`reason=(.+)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ========== TODO LIST 相关工具函数 ==========

// todoListToJSON 将 TodoList 转为 JSON 字符串
func todoListToJSON(tl TodoList) string {
	if len(tl.Todos) == 0 {
		return `{"todos":[]}`
	}
	b, err := json.Marshal(tl)
	if err != nil {
		return `{"todos":[]}`
	}
	return string(b)
}

// extractProposalText 从创想者回应中提取方案正文（去掉尾部的 [TODO_RESPONSES] JSON 块）
func extractProposalText(response string) string {
	idx := strings.Index(response, "[TODO_RESPONSES]")
	if idx > 0 {
		return strings.TrimSpace(response[:idx])
	}
	return response
}

// parseTodoVerdict 从裁判回应中解析 [TODO_VERDICT] JSON
func parseTodoVerdict(response string) TodoList {
	var result TodoList

	idx := strings.Index(response, "[TODO_VERDICT]")
	if idx < 0 {
		return result
	}

	jsonStr := response[idx+len("[TODO_VERDICT]"):]
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = cleanJSONBlock(jsonStr)

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// 尝试只解析 todos 数组
		var wrapper struct {
			Todos []TodoItem `json:"todos"`
		}
		if err2 := json.Unmarshal([]byte(jsonStr), &wrapper); err2 == nil {
			result.Todos = wrapper.Todos
		}
	}

	return result
}

// buildInitialTodoFromOpponent 从审判官第一轮的 [TODO_CHANGES] 构建初始 TODO LIST
func buildInitialTodoFromOpponent(opponentResponse string, round int) TodoList {
	var result TodoList

	idx := strings.Index(opponentResponse, "[TODO_CHANGES]")
	if idx < 0 {
		return result
	}

	jsonStr := opponentResponse[idx+len("[TODO_CHANGES]"):]
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = cleanJSONBlock(jsonStr)

	var changes struct {
		Review []struct {
			ID               string `json:"id"`
			StatusSuggestion string `json:"status_suggestion"`
			Comment          string `json:"comment"`
		} `json:"review"`
		Add []struct {
			ID       string `json:"id"`
			Issue    string `json:"issue"`
			Detail   string `json:"detail"`
			Priority string `json:"priority"`
		} `json:"add"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &changes); err != nil {
		return result
	}

	for _, item := range changes.Add {
		result.Todos = append(result.Todos, TodoItem{
			ID:           item.ID,
			Issue:        item.Issue,
			Detail:       item.Detail,
			Priority:     item.Priority,
			Status:       "open",
			CreatedBy:    "opponent",
			CreatedRound: round,
		})
	}

	return result
}

// buildTodoSummaryForJudge 构建人类可读的 TODO 摘要供终极仲裁参考
func buildTodoSummaryForJudge(tl TodoList) string {
	if len(tl.Todos) == 0 {
		return "无 TODO 记录"
	}

	var sb strings.Builder
	var openP0, openP1, resolved, total int
	total = len(tl.Todos)

	for _, t := range tl.Todos {
		var statusIcon string
		switch t.Status {
		case "resolved":
			statusIcon = "✅"
			resolved++
		case "wontfix":
			statusIcon = "🚫"
			resolved++
		default:
			statusIcon = "⏳"
			if t.Priority == "P0" {
				openP0++
			} else if t.Priority == "P1" {
				openP1++
			}
		}

		sb.WriteString(fmt.Sprintf("%s %s [%s] %s", statusIcon, t.ID, t.Priority, t.Issue))
		if t.Resolution != nil && *t.Resolution != "" {
			sb.WriteString(fmt.Sprintf(" → %s", *t.Resolution))
		}
		sb.WriteString("\n")
	}

	summary := fmt.Sprintf("共 %d 个 TODO，已解决 %d 个，P0未解决 %d 个，P1未解决 %d 个\n\n%s",
		total, resolved, openP0, openP1, sb.String())

	return summary
}

// cleanJSONBlock 清理可能的 markdown 代码块包裹
func cleanJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	// 找到 JSON 结束位置（最后一个 } 或 ]）
	lastBrace := strings.LastIndex(s, "}")
	lastBracket := strings.LastIndex(s, "]")
	end := lastBrace
	if lastBracket > end {
		end = lastBracket
	}
	if end > 0 {
		s = s[:end+1]
	}
	s = strings.TrimSpace(s)
	return s
}
