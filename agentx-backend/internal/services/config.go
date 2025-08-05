package services

import (
	"context"

	"github.com/agentx/agentx-backend/internal/repository"
)

// ConfigService manages application configuration
type ConfigService struct {
	configRepo repository.ConfigRepository
}

// NewConfigService creates a new config service
func NewConfigService(configRepo repository.ConfigRepository) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
	}
}

// GetSettings returns the current settings
func (s *ConfigService) GetSettings(ctx context.Context) (map[string]interface{}, error) {
	return s.configRepo.GetAll(ctx)
}

// UpdateSettings updates settings
func (s *ConfigService) UpdateSettings(ctx context.Context, settings map[string]interface{}) error {
	for key, value := range settings {
		if err := s.configRepo.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// GetSetting returns a specific setting
func (s *ConfigService) GetSetting(ctx context.Context, key string) (interface{}, error) {
	return s.configRepo.Get(ctx, key)
}

// SetSetting sets a specific setting
func (s *ConfigService) SetSetting(ctx context.Context, key string, value interface{}) error {
	return s.configRepo.Set(ctx, key, value)
}