package data

import (
	"context"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo 创建 UserRepo 实现
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{data: data, log: log.NewHelper(logger)}
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	model := &UserModel{
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
	}
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return &biz.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		Role:         model.Role,
		CreatedAt:    model.CreatedAt,
	}, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*biz.User, error) {
	var model UserModel
	if err := r.data.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		return nil, err
	}
	return &biz.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		Role:         model.Role,
		CreatedAt:    model.CreatedAt,
	}, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*biz.User, error) {
	var model UserModel
	if err := r.data.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	return &biz.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		Role:         model.Role,
		CreatedAt:    model.CreatedAt,
	}, nil
}

func (r *userRepo) List(ctx context.Context) ([]*biz.User, error) {
	var models []UserModel
	if err := r.data.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	users := make([]*biz.User, len(models))
	for i, m := range models {
		users[i] = &biz.User{
			ID:        m.ID,
			Username:  m.Username,
			Role:      m.Role,
			CreatedAt: m.CreatedAt,
		}
	}
	return users, nil
}

func (r *userRepo) Update(ctx context.Context, id int64, role string) error {
	return r.data.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", id).Update("role", role).Error
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	return r.data.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&UserModel{}, id).Error
}
