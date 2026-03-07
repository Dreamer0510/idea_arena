package server

import (
	"idea_arena/internal/conf"
	"idea_arena/internal/pkg/auth"
	mw "idea_arena/internal/pkg/middleware"
	"idea_arena/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/time/rate"
)

// NewHTTPServer 创建 HTTP 服务器
func NewHTTPServer(
	c *conf.Server,
	jwtHelper *auth.JWTHelper,
	ideaSvc *service.IdeaService,
	authSvc *service.AuthService,
	queueSvc *service.QueueService,
	registrySvc *service.RegistryService,
	debateSvc *service.DebateService,
	adminSvc *service.AdminService,
	adminExtSvc *service.AdminExtService,
	adminProviderSvc *service.AdminProviderService,
	logger log.Logger,
) *kratoshttp.Server {
	// 不需要认证的路径
	skipAuthPaths := []string{
		"/api/v1/auth/login",
		"/healthz",
		"/api/v1/registry",
		"/api/v1/ideas",          // GET 列表和详情不需要认证
		"/api/v1/debate/status",  // SSE 状态推送不需要认证
		"/api/v1/debate",         // SSE stream 不需要认证
	}

	opts := []kratoshttp.ServerOption{
		kratoshttp.Address(c.HTTP.Addr),
		kratoshttp.Timeout(c.HTTP.GetTimeout()),
		kratoshttp.Filter(mw.CORSFilter()),
		kratoshttp.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			mw.RateLimit(rate.NewLimiter(100, 200)),
			mw.JWTAuth(jwtHelper, skipAuthPaths),
		),
	}

	srv := kratoshttp.NewServer(opts...)

	// 注册路由
	r := srv.Route("/")
	ideaSvc.RegisterHTTPRoutes(r)
	authSvc.RegisterHTTPRoutes(r)
	queueSvc.RegisterHTTPRoutes(r)
	registrySvc.RegisterHTTPRoutes(r)
	debateSvc.RegisterHTTPRoutes(r)
	adminSvc.RegisterHTTPRoutes(r)
	adminExtSvc.RegisterHTTPRoutes(r)
	adminProviderSvc.RegisterHTTPRoutes(r)

	return srv
}
