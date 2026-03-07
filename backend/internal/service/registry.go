package service

import (
	"net/http"
	"time"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// RegistryService 统一后台注册服务
type RegistryService struct {
	conf      *conf.Registry
	startTime time.Time
	log       *log.Helper
}

// NewRegistryService 创建 RegistryService
func NewRegistryService(c *conf.Registry, logger log.Logger) *RegistryService {
	return &RegistryService{
		conf:      c,
		startTime: time.Now(),
		log:       log.NewHelper(logger),
	}
}

// RegisterHTTPRoutes 注册 HTTP 路由
func (s *RegistryService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.GET("/healthz", s.HealthCheck)
	r.GET("/api/v1/registry/info", s.GetProjectInfo)
	r.GET("/api/v1/registry/routes", s.ListRoutes)
	r.GET("/api/v1/registry/permissions", s.ListPermissions)
}

func (s *RegistryService) HealthCheck(ctx kratoshttp.Context) error {
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"status":         "healthy",
		"version":        s.conf.Version,
		"uptime_seconds": int64(time.Since(s.startTime).Seconds()),
	})
}

func (s *RegistryService) GetProjectInfo(ctx kratoshttp.Context) error {
	return ctx.Result(http.StatusOK, map[string]interface{}{
		"project_id":   s.conf.ProjectID,
		"project_name": s.conf.ProjectName,
		"description":  s.conf.Description,
		"version":      s.conf.Version,
	})
}

func (s *RegistryService) ListRoutes(ctx kratoshttp.Context) error {
	routes := []map[string]interface{}{
		{"method": "POST", "path": "/api/v1/auth/login", "description": "用户登录", "auth_required": false},
		{"method": "GET", "path": "/api/v1/auth/me", "description": "获取当前用户", "auth_required": true},
		{"method": "GET", "path": "/api/v1/ideas", "description": "Idea 列表", "auth_required": false},
		{"method": "GET", "path": "/api/v1/ideas/:id", "description": "Idea 详情", "auth_required": false},
		{"method": "POST", "path": "/api/v1/ideas", "description": "创建 Idea", "auth_required": true},
		{"method": "PUT", "path": "/api/v1/ideas/:id", "description": "更新 Idea", "auth_required": true},
		{"method": "DELETE", "path": "/api/v1/ideas/:id", "description": "删除 Idea", "auth_required": true},
		{"method": "POST", "path": "/api/v1/debate/start/:id", "description": "启动辩论", "auth_required": true},
		{"method": "GET", "path": "/api/v1/debate/status/:id", "description": "辩论状态 (SSE)", "auth_required": false},
		{"method": "POST", "path": "/api/v1/queue/submit", "description": "提交话题", "auth_required": true},
		{"method": "GET", "path": "/api/v1/queue/list", "description": "队列列表", "auth_required": true},
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"routes": routes})
}

func (s *RegistryService) ListPermissions(ctx kratoshttp.Context) error {
	permissions := []map[string]string{
		{"key": "idea:create", "name": "创建 Idea", "description": "允许创建新的创业点子"},
		{"key": "idea:update", "name": "更新 Idea", "description": "允许更新创业点子"},
		{"key": "idea:delete", "name": "删除 Idea", "description": "允许删除创业点子"},
		{"key": "debate:start", "name": "启动辩论", "description": "允许启动辩论"},
		{"key": "queue:submit", "name": "提交话题", "description": "允许手动提交话题"},
		{"key": "admin:manage", "name": "系统管理", "description": "管理员权限"},
	}
	return ctx.Result(http.StatusOK, map[string]interface{}{"permissions": permissions})
}
