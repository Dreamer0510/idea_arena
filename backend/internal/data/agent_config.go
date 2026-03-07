package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// AgentConfig 业务实体
type AgentConfig struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Emoji        string  `json:"emoji"`
	Role         string  `json:"role"`
	SystemPrompt string  `json:"system_prompt"`
	Temperature  float64 `json:"temperature"`
	ProviderID   int64   `json:"provider_id"`
	ModelName    string  `json:"model_name"`
	Enabled      bool    `json:"enabled"`
}

// AgentConfigRepo Agent配置仓储
type AgentConfigRepo struct {
	data *Data
	log  *log.Helper
}

// NewAgentConfigRepo 创建 AgentConfigRepo
func NewAgentConfigRepo(data *Data, logger log.Logger) *AgentConfigRepo {
	return &AgentConfigRepo{data: data, log: log.NewHelper(logger)}
}

func (r *AgentConfigRepo) List(ctx context.Context) ([]*AgentConfig, error) {
	var models []AgentConfigModel
	if err := r.data.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	configs := make([]*AgentConfig, len(models))
	for i, m := range models {
		configs[i] = modelToAgentConfig(&m)
	}
	return configs, nil
}

func (r *AgentConfigRepo) GetByID(ctx context.Context, id string) (*AgentConfig, error) {
	var model AgentConfigModel
	if err := r.data.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return modelToAgentConfig(&model), nil
}

func (r *AgentConfigRepo) Upsert(ctx context.Context, config *AgentConfig) error {
	model := &AgentConfigModel{
		ID:           config.ID,
		Name:         config.Name,
		Emoji:        config.Emoji,
		Role:         config.Role,
		SystemPrompt: config.SystemPrompt,
		Temperature:  config.Temperature,
		ProviderID:   config.ProviderID,
		ModelName:    config.ModelName,
		Enabled:      config.Enabled,
	}
	return r.data.db.WithContext(ctx).Save(model).Error
}

func (r *AgentConfigRepo) BatchUpsert(ctx context.Context, configs []*AgentConfig) error {
	tx := r.data.db.WithContext(ctx).Begin()
	for _, c := range configs {
		model := &AgentConfigModel{
			ID:           c.ID,
			Name:         c.Name,
			Emoji:        c.Emoji,
			Role:         c.Role,
			SystemPrompt: c.SystemPrompt,
			Temperature:  c.Temperature,
			ProviderID:   c.ProviderID,
			ModelName:    c.ModelName,
			Enabled:      c.Enabled,
		}
		if err := tx.Save(model).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func modelToAgentConfig(m *AgentConfigModel) *AgentConfig {
	return &AgentConfig{
		ID:           m.ID,
		Name:         m.Name,
		Emoji:        m.Emoji,
		Role:         m.Role,
		SystemPrompt: m.SystemPrompt,
		Temperature:  m.Temperature,
		ProviderID:   m.ProviderID,
		ModelName:    m.ModelName,
		Enabled:      m.Enabled,
	}
}
