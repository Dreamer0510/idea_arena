package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
)

// User 业务实体
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

// UserRepo 数据仓储接口
type UserRepo interface {
	Create(ctx context.Context, user *User) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Update(ctx context.Context, id int64, role string) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
}

// UserUsecase 用户业务用例
type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

// NewUserUsecase 创建 UserUsecase
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Login 用户登录
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*User, error) {
	user, err := uc.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.Unauthorized("INVALID_CREDENTIALS", "invalid password")
	}
	return user, nil
}

// GetByID 通过 ID 获取用户
func (uc *UserUsecase) GetByID(ctx context.Context, id int64) (*User, error) {
	return uc.repo.GetByID(ctx, id)
}

// List 获取所有用户
func (uc *UserUsecase) List(ctx context.Context) ([]*User, error) {
	return uc.repo.List(ctx)
}

// CreateUser 创建用户
func (uc *UserUsecase) CreateUser(ctx context.Context, username, password, role string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return uc.repo.Create(ctx, &User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	})
}

// UpdateRole 更新用户角色
func (uc *UserUsecase) UpdateRole(ctx context.Context, id int64, role string) error {
	return uc.repo.Update(ctx, id, role)
}

// ChangePassword 修改密码（需验证旧密码）
func (uc *UserUsecase) ChangePassword(ctx context.Context, id int64, oldPwd, newPwd string) error {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return errors.NotFound("USER_NOT_FOUND", "user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPwd)); err != nil {
		return errors.Unauthorized("INVALID_PASSWORD", "old password incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return uc.repo.UpdatePassword(ctx, id, string(hash))
}

// DeleteUser 删除用户
func (uc *UserUsecase) DeleteUser(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// EnsureAdmin 确保默认管理员存在
func (uc *UserUsecase) EnsureAdmin(ctx context.Context) error {
	_, err := uc.repo.GetByUsername(ctx, "admin")
	if err == nil {
		return nil // 已存在
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = uc.repo.Create(ctx, &User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	})
	return err
}
