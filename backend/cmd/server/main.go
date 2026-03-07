package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"idea_arena/internal/biz"
	"idea_arena/internal/conf"
	"idea_arena/internal/data"
	"idea_arena/internal/pkg/auth"
	"idea_arena/internal/pkg/crawler"
	"idea_arena/internal/pkg/llm"
	"idea_arena/internal/pkg/notify"
	"idea_arena/internal/pkg/search"
	"idea_arena/internal/server"
	"idea_arena/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "conf", "configs", "config path, eg: -conf configs")
}

func main() {
	flag.Parse()

	// 日志
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)

	// 加载配置
	c := config.New(
		config.WithSource(
			file.NewSource(configPath),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 初始化数据层
	dataLayer, cleanup, err := data.NewData(bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 初始化仓储
	ideaRepo := data.NewIdeaRepo(dataLayer, logger)
	userRepo := data.NewUserRepo(dataLayer, logger)
	queueRepo := data.NewQueueRepo(dataLayer, logger)

	// 初始化业务用例
	ideaUc := biz.NewIdeaUsecase(ideaRepo, logger)
	userUc := biz.NewUserUsecase(userRepo, logger)
	queueUc := biz.NewQueueUsecase(queueRepo, logger)

	// 确保默认管理员存在
	if err := userUc.EnsureAdmin(context.Background()); err != nil {
		log.NewHelper(logger).Warnf("failed to ensure admin: %v", err)
	}

	// 初始化 JWT
	jwtHelper := auth.NewJWTHelper(bc.Auth.JWTSecret, bc.Auth.GetJWTExpire())

	// 初始化 LLM 客户端
	llmClient := llm.NewClient(bc.LLM, logger)

	// 初始化搜索聚合器
	searcher := search.NewAggregator(bc.Search, logger)

	// 初始化辩论引擎
	debateUc := biz.NewDebateUsecase(ideaRepo, llmClient, searcher, bc.Debate, logger)

	// 初始化飞书通知
	var feishuNotifier *notify.FeishuNotifier
	if bc.Notify != nil && bc.Notify.Feishu != nil {
		feishuNotifier = notify.NewFeishuNotifier(bc.Notify.Feishu, logger)
	}

	// 注册辩论完成回调（飞书通知）
	if feishuNotifier != nil {
		// 启动汇总通知调度器（每3小时）
		feishuNotifier.StartSummaryScheduler(context.Background())

		debateUc.SetOnComplete(func(ctx context.Context, ideaID int64, finalStatus string, scores [4]float64) {
			idea, err := ideaUc.Get(ctx, ideaID)
			if err != nil {
				log.NewHelper(logger).Warnf("feishu callback: failed to get idea %d: %v", ideaID, err)
				return
			}
			// 解析辩论日志，提取摘要和各轮评分（落地曲线）
			var summary string
			var roundScores []float64
			if idea.DebateLog != "" {
				var dl struct {
					Rounds []struct {
						Overall float64 `json:"overall"`
					} `json:"rounds"`
					Summary string `json:"summary"`
				}
				if err := json.Unmarshal([]byte(idea.DebateLog), &dl); err == nil {
					summary = dl.Summary
					for _, r := range dl.Rounds {
						if r.Overall > 0 {
							roundScores = append(roundScores, r.Overall)
						}
					}
				}
			}

			info := &notify.IdeaInfo{
				ID:               idea.ID,
				Topic:            idea.Topic,
				ProductName:      idea.ProductName,
				OneLiner:         idea.OneLiner,
				Status:           idea.Status,
				ScoreFeasibility: idea.ScoreFeasibility,
				ScoreEconomics:   idea.ScoreEconomics,
				ScoreProfit:      idea.ScoreProfit,
				ScoreOverall:     idea.ScoreOverall,
				RoundCount:       idea.RoundCount,
				Tags:             idea.Tags,
				Summary:          summary,
				FinalReport:      idea.FinalReport,
				JudgeRefined:     idea.JudgeRefined,
				RoundScores:      roundScores,
			}

			// 所有结果都记录到统计（用于汇总通知）
			feishuNotifier.RecordDebateResult(info)

			// 只对 graduated/promising 发即时通知，failed 不单独通知
			switch idea.Status {
			case "graduated":
				if err := feishuNotifier.SendDebateResult(ctx, info); err != nil {
					log.NewHelper(logger).Warnf("feishu notification failed for idea %d: %v", ideaID, err)
				}
				if idea.ScoreOverall >= 8.0 {
					_ = feishuNotifier.SendHighScoreAlert(ctx, info)
				}
			case "promising":
				if err := feishuNotifier.SendDebateResult(ctx, info); err != nil {
					log.NewHelper(logger).Warnf("feishu notification failed for idea %d: %v", ideaID, err)
				}
			}
		})
	}

	// 初始化话题发现仓储和用例
	discoveryRepo := data.NewDiscoveryRepo(dataLayer, logger)
	discoveryUc := biz.NewDiscoveryUsecase(discoveryRepo, ideaRepo, llmClient, logger)

	// 注册爬虫插件（9个渠道：吾爱破解 + 关键词搜索 + 搜索热点趋势 + 社交痛点 + 学术前沿 + 融资信号 + 政策信号 + 需求信号 + LLM创意生成）
	discoveryUc.RegisterPlugin(crawler.NewPojie52Plugin(logger))
	discoveryUc.RegisterPlugin(crawler.NewKeywordSearchPlugin(logger, discoveryUc.GetEnabledKeywords))
	discoveryUc.RegisterPlugin(crawler.NewTrendSearchPlugin(logger, discoveryUc.GetEnabledConstraintTags, discoveryUc.GetEnabledTrendQueries))
	discoveryUc.RegisterPlugin(crawler.NewSocialPainPlugin(logger, discoveryUc.GetEnabledConstraintTags))
	discoveryUc.RegisterPlugin(crawler.NewAcademicFrontierPlugin(logger, discoveryUc.GetEnabledConstraintTags))
	discoveryUc.RegisterPlugin(crawler.NewFundingSignalPlugin(logger, discoveryUc.GetEnabledConstraintTags))
	discoveryUc.RegisterPlugin(crawler.NewPolicySignalPlugin(logger, discoveryUc.GetEnabledConstraintTags))
	discoveryUc.RegisterPlugin(crawler.NewDemandSignalPlugin(logger, discoveryUc.GetEnabledConstraintTags))
	discoveryUc.RegisterPlugin(crawler.NewLLMCreativePlugin(llmClient, logger, discoveryUc.GetEnabledConstraintTags))

	// 注入辩论引擎（用于全自动发现→辩论流程）
	discoveryUc.SetDebateUsecase(debateUc)

	// 启动全自动话题发现+辩论调度器
	discoveryUc.StartScheduler(context.Background())

	// 初始化扩展仓储
	postRepo := data.NewSocialPostRepo(dataLayer, logger)
	agentRepo := data.NewAgentConfigRepo(dataLayer, logger)
	providerRepo := data.NewProviderRepo(dataLayer, logger)

	// 初始化 ProviderManager 并加载已有服务商
	providerManager := llm.NewProviderManager(logger)
	existingProviders, _ := providerRepo.List(context.Background())

	// 自动迁移：如果 DB 中没有服务商，从 config.yaml 创建默认服务商
	if len(existingProviders) == 0 && bc.LLM != nil && bc.LLM.BaseURL != "" && bc.LLM.APIKey != "" {
		seeded, seedErr := providerRepo.Create(context.Background(), &data.LLMProvider{
			Name:      "默认服务商 (config.yaml)",
			BaseURL:   bc.LLM.BaseURL,
			APIKey:    bc.LLM.APIKey,
			IsDefault: true,
			Enabled:   true,
		})
		if seedErr == nil {
			existingProviders = append(existingProviders, seeded)
			log.NewHelper(logger).Infof("[Init] Auto-seeded provider from config.yaml: id=%d url=%s", seeded.ID, seeded.BaseURL)
		}
	}

	for _, p := range existingProviders {
		if p.Enabled {
			providerManager.RegisterProvider(llm.ProviderInfo{
				ID:      p.ID,
				Name:    p.Name,
				BaseURL: p.BaseURL,
				APIKey:  p.APIKey,
			})
		}
	}

	// 将 ProviderManager 注入 DebateUsecase（通过适配器）
	debateUc.SetProviderManager(providerManager, &agentConfigRepoAdapter{repo: agentRepo}, &providerRepoAdapter{repo: providerRepo})

	// 初始化服务
	ideaSvc := service.NewIdeaService(ideaUc, logger)
	authSvc := service.NewAuthService(userUc, jwtHelper, logger)
	queueSvc := service.NewQueueService(queueUc, logger)
	registrySvc := service.NewRegistryService(bc.Registry, logger)
	debateSvc := service.NewDebateService(debateUc, logger)
	adminSvc := service.NewAdminService(discoveryUc, debateUc, logger)
	adminExtSvc := service.NewAdminExtService(ideaRepo, userUc, bc.LLM, llmClient, postRepo, agentRepo, logger)
	adminProviderSvc := service.NewAdminProviderService(providerRepo, providerManager, logger)

	// 创建 HTTP 服务器
	httpSrv := server.NewHTTPServer(bc.Server, jwtHelper, ideaSvc, authSvc, queueSvc, registrySvc, debateSvc, adminSvc, adminExtSvc, adminProviderSvc, logger)

	// 创建 Kratos 应用
	app := kratos.New(
		kratos.Name(bc.Registry.ProjectID),
		kratos.Version(bc.Registry.Version),
		kratos.Logger(logger),
		kratos.Server(httpSrv),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// ========== Adapters ==========

// agentConfigRepoAdapter 适配 data.AgentConfigRepo -> biz.AgentConfigRepository
type agentConfigRepoAdapter struct {
	repo *data.AgentConfigRepo
}

func (a *agentConfigRepoAdapter) GetByID(ctx context.Context, id string) (*biz.AgentConfigDTO, error) {
	cfg, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &biz.AgentConfigDTO{
		ID: cfg.ID, Name: cfg.Name, Emoji: cfg.Emoji, Role: cfg.Role,
		SystemPrompt: cfg.SystemPrompt, Temperature: cfg.Temperature,
		ProviderID: cfg.ProviderID, ModelName: cfg.ModelName, Enabled: cfg.Enabled,
	}, nil
}

func (a *agentConfigRepoAdapter) List(ctx context.Context) ([]*biz.AgentConfigDTO, error) {
	cfgs, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AgentConfigDTO, len(cfgs))
	for i, c := range cfgs {
		result[i] = &biz.AgentConfigDTO{
			ID: c.ID, Name: c.Name, Emoji: c.Emoji, Role: c.Role,
			SystemPrompt: c.SystemPrompt, Temperature: c.Temperature,
			ProviderID: c.ProviderID, ModelName: c.ModelName, Enabled: c.Enabled,
		}
	}
	return result, nil
}

// providerRepoAdapter 适配 data.ProviderRepo -> biz.ProviderRepository
type providerRepoAdapter struct {
	repo *data.ProviderRepo
}

func (a *providerRepoAdapter) GetByID(ctx context.Context, id int64) (*biz.ProviderDTO, error) {
	p, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &biz.ProviderDTO{ID: p.ID, Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey}, nil
}

func (a *providerRepoAdapter) GetDefault(ctx context.Context) (*biz.ProviderDTO, error) {
	p, err := a.repo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.ProviderDTO{ID: p.ID, Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey}, nil
}

// NewData 导出给 Wire 用（暂时不用 Wire，直接手动注入）
var _ http.Server
