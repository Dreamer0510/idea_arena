package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// LLMProvider 业务实体
type LLMProvider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	ModelsURL string `json:"models_url"`
	IsDefault bool   `json:"is_default"`
	Enabled   bool   `json:"enabled"`
}

// ProviderModelInfo 服务商支持的模型信息
type ProviderModelInfo struct {
	ModelID string `json:"model_id"`
	OwnedBy string `json:"owned_by"`
}

// ProviderRepo LLM 服务商仓储
type ProviderRepo struct {
	data *Data
	log  *log.Helper
}

// NewProviderRepo 创建 ProviderRepo
func NewProviderRepo(data *Data, logger log.Logger) *ProviderRepo {
	return &ProviderRepo{data: data, log: log.NewHelper(logger)}
}

func (r *ProviderRepo) List(ctx context.Context) ([]*LLMProvider, error) {
	var models []LLMProviderModel
	if err := r.data.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	providers := make([]*LLMProvider, len(models))
	for i, m := range models {
		providers[i] = modelToProvider(&m)
	}
	return providers, nil
}

func (r *ProviderRepo) GetByID(ctx context.Context, id int64) (*LLMProvider, error) {
	var model LLMProviderModel
	if err := r.data.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return modelToProvider(&model), nil
}

func (r *ProviderRepo) GetDefault(ctx context.Context) (*LLMProvider, error) {
	var model LLMProviderModel
	if err := r.data.db.WithContext(ctx).Where("is_default = ? AND enabled = ?", true, true).First(&model).Error; err != nil {
		// fallback: get any enabled provider
		if err2 := r.data.db.WithContext(ctx).Where("enabled = ?", true).First(&model).Error; err2 != nil {
			return nil, fmt.Errorf("no enabled provider found")
		}
	}
	return modelToProvider(&model), nil
}

func (r *ProviderRepo) Create(ctx context.Context, p *LLMProvider) (*LLMProvider, error) {
	model := providerToModel(p)
	if p.IsDefault {
		// 清除其他 default
		r.data.db.WithContext(ctx).Model(&LLMProviderModel{}).Where("is_default = ?", true).Update("is_default", false)
	}
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return modelToProvider(model), nil
}

func (r *ProviderRepo) Update(ctx context.Context, p *LLMProvider) error {
	if p.IsDefault {
		r.data.db.WithContext(ctx).Model(&LLMProviderModel{}).Where("id != ? AND is_default = ?", p.ID, true).Update("is_default", false)
	}
	return r.data.db.WithContext(ctx).Model(&LLMProviderModel{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"name":       p.Name,
		"base_url":   p.BaseURL,
		"api_key":    p.APIKey,
		"models_url": p.ModelsURL,
		"is_default": p.IsDefault,
		"enabled":    p.Enabled,
	}).Error
}

func (r *ProviderRepo) Delete(ctx context.Context, id int64) error {
	// 同时删除关联的模型缓存
	r.data.db.WithContext(ctx).Where("provider_id = ?", id).Delete(&LLMProviderModelListModel{})
	return r.data.db.WithContext(ctx).Delete(&LLMProviderModel{}, id).Error
}

// SaveModels 保存服务商的模型列表（全量替换）
func (r *ProviderRepo) SaveModels(ctx context.Context, providerID int64, models []ProviderModelInfo) error {
	tx := r.data.db.WithContext(ctx).Begin()
	// 清除旧的
	if err := tx.Where("provider_id = ?", providerID).Delete(&LLMProviderModelListModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 批量插入
	if len(models) > 0 {
		records := make([]LLMProviderModelListModel, len(models))
		for i, m := range models {
			records[i] = LLMProviderModelListModel{
				ProviderID: providerID,
				ModelID:    m.ModelID,
				OwnedBy:    m.OwnedBy,
			}
		}
		if err := tx.CreateInBatches(records, 100).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// GetModels 获取服务商的已缓存模型列表
func (r *ProviderRepo) GetModels(ctx context.Context, providerID int64) ([]ProviderModelInfo, error) {
	var records []LLMProviderModelListModel
	if err := r.data.db.WithContext(ctx).Where("provider_id = ?", providerID).Order("model_id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	models := make([]ProviderModelInfo, len(records))
	for i, r := range records {
		models[i] = ProviderModelInfo{ModelID: r.ModelID, OwnedBy: r.OwnedBy}
	}
	return models, nil
}

func modelToProvider(m *LLMProviderModel) *LLMProvider {
	return &LLMProvider{
		ID:        m.ID,
		Name:      m.Name,
		BaseURL:   m.BaseURL,
		APIKey:    m.APIKey,
		ModelsURL: m.ModelsURL,
		IsDefault: m.IsDefault,
		Enabled:   m.Enabled,
	}
}

func providerToModel(p *LLMProvider) *LLMProviderModel {
	return &LLMProviderModel{
		ID:        p.ID,
		Name:      p.Name,
		BaseURL:   p.BaseURL,
		APIKey:    p.APIKey,
		ModelsURL: p.ModelsURL,
		IsDefault: p.IsDefault,
		Enabled:   p.Enabled,
	}
}
