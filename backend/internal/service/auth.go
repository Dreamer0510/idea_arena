package service

import (
	"net/http"
	"strings"

	"idea_arena/internal/biz"
	"idea_arena/internal/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AuthService 认证 HTTP 服务
type AuthService struct {
	uc        *biz.UserUsecase
	jwtHelper *auth.JWTHelper
	log       *log.Helper
}

// NewAuthService 创建 AuthService
func NewAuthService(uc *biz.UserUsecase, jwtHelper *auth.JWTHelper, logger log.Logger) *AuthService {
	return &AuthService{uc: uc, jwtHelper: jwtHelper, log: log.NewHelper(logger)}
}

// RegisterHTTPRoutes 注册 HTTP 路由
func (s *AuthService) RegisterHTTPRoutes(r *kratoshttp.Router) {
	r.POST("/api/v1/auth/login", s.Login)
	r.POST("/api/v1/auth/logout", s.Logout)
	r.GET("/api/v1/auth/me", s.GetCurrentUser)
	r.PUT("/api/v1/auth/password", s.ChangePassword)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *AuthService) Login(ctx kratoshttp.Context) error {
	var req loginRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return ctx.Result(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	token, err := s.jwtHelper.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return ctx.Result(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	return ctx.Result(http.StatusOK, &loginResponse{
		Token: token,
		User: userResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

func (s *AuthService) Logout(ctx kratoshttp.Context) error {
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}

func (s *AuthService) extractClaims(ctx kratoshttp.Context) *auth.Claims {
	authHeader := ctx.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil
	}
	claims, err := s.jwtHelper.ParseToken(parts[1])
	if err != nil {
		return nil
	}
	return claims
}

func (s *AuthService) GetCurrentUser(ctx kratoshttp.Context) error {
	claims := s.extractClaims(ctx)
	if claims == nil {
		return ctx.Result(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := s.uc.GetByID(ctx, claims.UserID)
	if err != nil {
		return ctx.Result(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return ctx.Result(http.StatusOK, &userResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (s *AuthService) ChangePassword(ctx kratoshttp.Context) error {
	claims := s.extractClaims(ctx)
	if claims == nil {
		return ctx.Result(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := ctx.Bind(&req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "old_password and new_password required"})
	}
	if len(req.NewPassword) < 6 {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "new password must be at least 6 characters"})
	}

	if err := s.uc.ChangePassword(ctx, claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		return ctx.Result(http.StatusBadRequest, map[string]string{"error": "旧密码不正确"})
	}

	s.log.Infof("[Auth] User %s changed password", claims.Username)
	return ctx.Result(http.StatusOK, map[string]bool{"success": true})
}
