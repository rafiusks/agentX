package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ConfigManager manages LLM configuration
type ConfigManager struct {
	configs map[string]interface{}
	mu      sync.RWMutex
}

// NewConfigManager creates a new config manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]interface{}),
	}
}

// Load loads configuration from a file
func (cm *ConfigManager) Load(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	cm.configs = config
	return nil
}

// Get retrieves a configuration value
func (cm *ConfigManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	val, exists := cm.configs[key]
	return val, exists
}

// Set sets a configuration value
func (cm *ConfigManager) Set(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.configs[key] = value
}

// GetString retrieves a string configuration value
func (cm *ConfigManager) GetString(key string) (string, bool) {
	val, exists := cm.Get(key)
	if !exists {
		return "", false
	}
	
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an integer configuration value
func (cm *ConfigManager) GetInt(key string) (int, bool) {
	val, exists := cm.Get(key)
	if !exists {
		return 0, false
	}
	
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a boolean configuration value
func (cm *ConfigManager) GetBool(key string) (bool, bool) {
	val, exists := cm.Get(key)
	if !exists {
		return false, false
	}
	
	b, ok := val.(bool)
	return b, ok
}

// GetProviderConfig retrieves provider-specific configuration
func (cm *ConfigManager) GetProviderConfig(provider string) (ProviderConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	val, exists := cm.configs["providers"]
	if !exists {
		return ProviderConfig{}, false
	}
	
	providers, ok := val.(map[string]interface{})
	if !ok {
		return ProviderConfig{}, false
	}
	
	providerData, exists := providers[provider]
	if !exists {
		return ProviderConfig{}, false
	}
	
	// Convert to JSON and back to ProviderConfig
	jsonData, err := json.Marshal(providerData)
	if err != nil {
		return ProviderConfig{}, false
	}
	
	var config ProviderConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return ProviderConfig{}, false
	}
	
	return config, true
}

// GetDefaultProvider returns the default provider name
func (cm *ConfigManager) GetDefaultProvider() string {
	if provider, ok := cm.GetString("default_provider"); ok {
		return provider
	}
	return "openai"
}

// GetDefaultModel returns the default model for a provider
func (cm *ConfigManager) GetDefaultModel(provider string) string {
	key := fmt.Sprintf("default_models.%s", provider)
	if model, ok := cm.GetString(key); ok {
		return model
	}
	
	// Fallback defaults
	switch provider {
	case "openai":
		return "gpt-3.5-turbo"
	case "anthropic":
		return "claude-3-sonnet-20240229"
	default:
		return ""
	}
}