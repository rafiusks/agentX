package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Server          ServerConfig              `json:"server"`
	Database        DatabaseConfig            `json:"database"`
	Providers       map[string]ProviderConfig `json:"providers"`
	DefaultProvider string                    `json:"default_provider"`
	DefaultModel    string                    `json:"default_model"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}

type ProviderConfig struct {
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	BaseURL       string                 `json:"base_url,omitempty"`
	APIKey        string                 `json:"api_key,omitempty"`
	Models        []string               `json:"models"`
	DefaultModel  string                 `json:"default_model"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	
	// Add config paths
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	
	// Check for user config directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(homeDir, ".agentx"))
	}

	// Set defaults
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "agentx")
	viper.SetDefault("database.database", "agentx")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("default_provider", "openai")

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Create default config
			return createDefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Load environment variables
	loadEnvOverrides(&cfg)

	return &cfg, nil
}

func createDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "agentx",
			Password: "",
			Database: "agentx",
			SSLMode:  "disable",
		},
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:         "openai",
				Name:         "OpenAI",
				Models:       []string{"gpt-4", "gpt-3.5-turbo"},
				DefaultModel: "gpt-3.5-turbo",
			},
			"anthropic": {
				Type:         "anthropic",
				Name:         "Anthropic",
				Models:       []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229"},
				DefaultModel: "claude-3-sonnet-20240229",
			},
			"ollama": {
				Type:         "openai-compatible",
				Name:         "Ollama",
				BaseURL:      "http://localhost:11434",
				Models:       []string{}, // Will be discovered dynamically
				DefaultModel: "",
			},
			"lm-studio": {
				Type:         "openai-compatible",
				Name:         "LM Studio",
				BaseURL:      "http://localhost:1234",
				Models:       []string{}, // Will be discovered dynamically
				DefaultModel: "",
			},
		},
		DefaultProvider: "openai",
	}
}

func loadEnvOverrides(cfg *Config) {
	// Override with environment variables
	if port := os.Getenv("AGENTX_PORT"); port != "" {
		viper.Set("server.port", port)
	}
	
	if host := os.Getenv("AGENTX_HOST"); host != "" {
		viper.Set("server.host", host)
	}

	// Database overrides
	if dbHost := os.Getenv("POSTGRES_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("POSTGRES_PORT"); dbPort != "" {
		if port, err := strconv.Atoi(dbPort); err == nil {
			cfg.Database.Port = port
		}
	}
	if dbUser := os.Getenv("POSTGRES_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPass := os.Getenv("POSTGRES_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if dbName := os.Getenv("POSTGRES_DB"); dbName != "" {
		cfg.Database.Database = dbName
	}

	// Note: API keys are now stored in the database, not environment variables
}

func (c *Config) Save() error {
	return viper.WriteConfig()
}