use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::PathBuf;
use std::time::Duration;

use crate::mcp::server::MCPServerConfig;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentXConfig {
    pub default_provider: String,
    pub default_model: String,
    pub providers: HashMap<String, ProviderConfig>,
    pub ui: UIConfig,
    #[serde(default)]
    pub mcp_servers: Vec<MCPServerConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderConfig {
    pub name: String,
    pub provider_type: ProviderType,
    pub base_url: String,
    pub api_key: Option<String>,
    pub models: Vec<ModelConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProviderType {
    OpenAICompatible,
    OpenAI,
    Anthropic,
    Ollama,
    MCP,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelConfig {
    pub name: String,
    pub display_name: String,
    pub context_size: usize,
    pub supports_streaming: bool,
    pub description: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UIConfig {
    pub show_model_selector: bool,
    pub auto_stream: bool,
    pub theme: String,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ProviderStatus {
    Online,
    Offline,
    ConfigError(String),
    Unknown,
}

impl ProviderStatus {
    pub fn icon(&self) -> &'static str {
        match self {
            ProviderStatus::Online => "🟢",
            ProviderStatus::Offline => "🔴", 
            ProviderStatus::ConfigError(_) => "🟡",
            ProviderStatus::Unknown => "⚪",
        }
    }
    
    pub fn description(&self) -> String {
        match self {
            ProviderStatus::Online => "Available".to_string(),
            ProviderStatus::Offline => "Not available".to_string(),
            ProviderStatus::ConfigError(msg) => format!("Configuration issue: {}", msg),
            ProviderStatus::Unknown => "Status unknown".to_string(),
        }
    }
}

#[derive(Debug, Clone)]
pub struct ProviderHealth {
    pub status: ProviderStatus,
    pub response_time: Option<Duration>,
    pub last_checked: std::time::SystemTime,
    pub available_models: Vec<String>,
}

impl Default for AgentXConfig {
    fn default() -> Self {
        let mut providers = HashMap::new();
        
        // Default Ollama configuration
        providers.insert("ollama".to_string(), ProviderConfig {
            name: "Ollama".to_string(),
            provider_type: ProviderType::Ollama,
            base_url: "http://localhost:11434".to_string(),
            api_key: None,
            models: vec![
                ModelConfig {
                    name: "llama2".to_string(),
                    display_name: "Llama 2 7B".to_string(),
                    context_size: 4096,
                    supports_streaming: true,
                    description: Some("Meta's Llama 2 7B model - good for general tasks".to_string()),
                },
                ModelConfig {
                    name: "codellama".to_string(),
                    display_name: "Code Llama 7B".to_string(),
                    context_size: 4096,
                    supports_streaming: true,
                    description: Some("Code-focused Llama model for programming tasks".to_string()),
                },
                ModelConfig {
                    name: "mistral".to_string(),
                    display_name: "Mistral 7B".to_string(),
                    context_size: 8192,
                    supports_streaming: true,
                    description: Some("Efficient and fast 7B parameter model".to_string()),
                },
            ],
        });
        
        // Default OpenAI configuration (disabled by default)
        providers.insert("openai".to_string(), ProviderConfig {
            name: "OpenAI".to_string(),
            provider_type: ProviderType::OpenAI,
            base_url: "https://api.openai.com/v1".to_string(),
            api_key: None, // User must provide
            models: vec![
                ModelConfig {
                    name: "gpt-3.5-turbo".to_string(),
                    display_name: "GPT-3.5 Turbo".to_string(),
                    context_size: 4096,
                    supports_streaming: true,
                    description: Some("Fast and efficient for most tasks".to_string()),
                },
                ModelConfig {
                    name: "gpt-4".to_string(),
                    display_name: "GPT-4".to_string(),
                    context_size: 8192,
                    supports_streaming: true,
                    description: Some("Most capable model for complex reasoning".to_string()),
                },
            ],
        });
        
        Self {
            default_provider: "ollama".to_string(),
            default_model: "llama2".to_string(),
            providers,
            ui: UIConfig {
                show_model_selector: true,
                auto_stream: true,
                theme: "default".to_string(),
            },
            mcp_servers: vec![],
        }
    }
}

impl AgentXConfig {
    pub fn load() -> Result<Self> {
        let config_path = Self::config_file_path()?;
        
        if config_path.exists() {
            let content = std::fs::read_to_string(&config_path)?;
            let config: Self = toml::from_str(&content)?;
            Ok(config)
        } else {
            // Create default config file
            let default_config = Self::default();
            default_config.save()?;
            Ok(default_config)
        }
    }
    
    pub fn save(&self) -> Result<()> {
        let config_path = Self::config_file_path()?;
        
        // Ensure config directory exists
        if let Some(parent) = config_path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        let content = toml::to_string_pretty(self)?;
        std::fs::write(config_path, content)?;
        Ok(())
    }
    
    fn config_file_path() -> Result<PathBuf> {
        let home = dirs::home_dir()
            .ok_or_else(|| anyhow::anyhow!("Could not find home directory"))?;
        Ok(home.join(".config").join("agentx").join("config.toml"))
    }
    
    pub fn get_provider(&self, name: &str) -> Option<&ProviderConfig> {
        self.providers.get(name)
    }
    
    pub fn get_default_provider(&self) -> Option<&ProviderConfig> {
        self.get_provider(&self.default_provider)
    }
    
    pub fn get_model(&self, provider_name: &str, model_name: &str) -> Option<&ModelConfig> {
        self.get_provider(provider_name)?
            .models
            .iter()
            .find(|m| m.name == model_name)
    }
    
    pub fn get_default_model(&self) -> Option<&ModelConfig> {
        self.get_model(&self.default_provider, &self.default_model)
    }
    
    pub fn list_available_models(&self) -> Vec<(String, ModelConfig)> {
        let mut models = Vec::new();
        for (provider_name, provider) in &self.providers {
            for model in &provider.models {
                models.push((format!("{}/{}", provider_name, model.name), model.clone()));
            }
        }
        models.sort_by(|a, b| a.0.cmp(&b.0));
        models
    }
    
    pub fn set_default_model(&mut self, provider: String, model: String) -> Result<()> {
        // Validate that the provider and model exist
        if let Some(p) = self.get_provider(&provider) {
            if p.models.iter().any(|m| m.name == model) {
                self.default_provider = provider;
                self.default_model = model;
                self.save()?;
                return Ok(());
            }
        }
        Err(anyhow::anyhow!("Provider {} or model {} not found", provider, model))
    }
    
    pub async fn check_provider_health(&self, provider_name: &str) -> ProviderHealth {
        let start_time = std::time::Instant::now();
        
        let health = match self.get_provider(provider_name) {
            Some(provider_config) => {
                match self.check_provider_availability(provider_config).await {
                    Ok(models) => ProviderHealth {
                        status: ProviderStatus::Online,
                        response_time: Some(start_time.elapsed()),
                        last_checked: std::time::SystemTime::now(),
                        available_models: models,
                    },
                    Err(_e) => ProviderHealth {
                        status: ProviderStatus::Offline,
                        response_time: Some(start_time.elapsed()),
                        last_checked: std::time::SystemTime::now(),
                        available_models: vec![],
                    }
                }
            }
            None => ProviderHealth {
                status: ProviderStatus::ConfigError("Provider not found".to_string()),
                response_time: None,
                last_checked: std::time::SystemTime::now(),
                available_models: vec![],
            }
        };
        
        health
    }
    
    pub async fn check_all_providers_health(&self) -> HashMap<String, ProviderHealth> {
        let mut health_map = HashMap::new();
        
        for provider_name in self.providers.keys() {
            let health = self.check_provider_health(provider_name).await;
            health_map.insert(provider_name.clone(), health);
        }
        
        health_map
    }
    
    async fn check_provider_availability(&self, provider_config: &ProviderConfig) -> Result<Vec<String>> {
        match provider_config.provider_type {
            ProviderType::Ollama | ProviderType::OpenAICompatible => {
                self.check_openai_compatible_availability(provider_config).await
            }
            ProviderType::OpenAI => {
                self.check_openai_availability(provider_config).await
            }
            ProviderType::Anthropic => {
                self.check_anthropic_availability(provider_config).await
            }
            ProviderType::MCP => {
                // MCP servers are checked differently
                Ok(vec!["mcp-default".to_string()])
            }
        }
    }
    
    async fn check_openai_compatible_availability(&self, provider_config: &ProviderConfig) -> Result<Vec<String>> {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()?;
            
        let url = format!("{}/v1/models", provider_config.base_url);
        let response = client.get(&url).send().await?;
        
        if response.status().is_success() {
            // Try to parse the models response
            let models_response: serde_json::Value = response.json().await?;
            let models = models_response["data"]
                .as_array()
                .unwrap_or(&vec![])
                .iter()
                .filter_map(|model| model["id"].as_str().map(|s| s.to_string()))
                .collect();
            Ok(models)
        } else {
            Err(anyhow::anyhow!("HTTP {}: {}", response.status(), response.text().await?))
        }
    }
    
    async fn check_openai_availability(&self, provider_config: &ProviderConfig) -> Result<Vec<String>> {
        if provider_config.api_key.is_none() {
            return Err(anyhow::anyhow!("OpenAI API key required"));
        }
        
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()?;
            
        let url = format!("{}/models", provider_config.base_url);
        let response = client
            .get(&url)
            .bearer_auth(provider_config.api_key.as_ref().unwrap())
            .send()
            .await?;
        
        if response.status().is_success() {
            let models_response: serde_json::Value = response.json().await?;
            let models = models_response["data"]
                .as_array()
                .unwrap_or(&vec![])
                .iter()
                .filter_map(|model| model["id"].as_str().map(|s| s.to_string()))
                .collect();
            Ok(models)
        } else {
            Err(anyhow::anyhow!("HTTP {}: {}", response.status(), response.text().await?))
        }
    }
    
    async fn check_anthropic_availability(&self, provider_config: &ProviderConfig) -> Result<Vec<String>> {
        if provider_config.api_key.is_none() {
            return Err(anyhow::anyhow!("Anthropic API key required"));
        }
        
        // Anthropic doesn't have a models endpoint, so we'll just check if we can reach the API
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()?;
            
        // Use a minimal completion request to test connectivity
        let test_request = serde_json::json!({
            "model": "claude-3-haiku-20240307",
            "max_tokens": 1,
            "messages": [{"role": "user", "content": "test"}]
        });
        
        let url = format!("{}/messages", provider_config.base_url);
        let response = client
            .post(&url)
            .header("x-api-key", provider_config.api_key.as_ref().unwrap())
            .header("anthropic-version", "2023-06-01")
            .json(&test_request)
            .send()
            .await?;
        
        if response.status().is_success() || response.status() == 400 {
            // 400 is expected for our minimal test request
            Ok(provider_config.models.iter().map(|m| m.name.clone()).collect())
        } else {
            Err(anyhow::anyhow!("HTTP {}: {}", response.status(), response.text().await?))
        }
    }
}