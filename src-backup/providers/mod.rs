use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::pin::Pin;
use futures::Stream;
use anyhow::Result;

pub mod openai;
pub mod anthropic;
pub mod openai_compatible;
pub mod mcp;
pub mod demo;

/// Represents a message in a conversation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    pub role: MessageRole,
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MessageRole {
    System,
    User,
    Assistant,
}

/// Request for a completion
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompletionRequest {
    pub messages: Vec<Message>,
    pub model: String,
    pub temperature: Option<f32>,
    pub max_tokens: Option<u32>,
    pub stream: bool,
}

/// Response from a completion
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompletionResponse {
    pub content: String,
    pub model: String,
    pub usage: Option<TokenUsage>,
}

/// Token usage information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenUsage {
    pub prompt_tokens: u32,
    pub completion_tokens: u32,
    pub total_tokens: u32,
}

/// Streaming response chunk
#[derive(Debug, Clone)]
pub struct StreamingResponse {
    pub delta: String,
    pub finish_reason: Option<String>,
}

/// Provider capabilities
#[derive(Debug, Clone)]
pub struct ProviderCapabilities {
    pub models: Vec<ModelInfo>,
    pub supports_streaming: bool,
    pub supports_functions: bool,
    pub supports_system_messages: bool,
}

#[derive(Debug, Clone)]
pub struct ModelInfo {
    pub id: String,
    pub name: String,
    pub context_window: u32,
    pub input_cost_per_1k: Option<f64>,
    pub output_cost_per_1k: Option<f64>,
}

/// Configuration for a provider
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderConfig {
    pub api_key: Option<String>,
    pub base_url: Option<String>,
    pub timeout_secs: Option<u64>,
    pub max_retries: Option<u32>,
}

/// Stream type for responses
pub type ResponseStream = Pin<Box<dyn Stream<Item = Result<StreamingResponse>> + Send>>;

/// Main trait that all LLM providers must implement
#[async_trait]
pub trait LLMProvider: Send + Sync {
    /// Get provider name
    fn name(&self) -> &str;
    
    /// Get provider capabilities
    async fn capabilities(&self) -> Result<ProviderCapabilities>;
    
    /// Complete a prompt (non-streaming)
    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse>;
    
    /// Complete a prompt (streaming)
    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream>;
    
    /// Validate configuration
    async fn validate_config(&self) -> Result<()>;
    
    /// Estimate token count for a prompt
    fn estimate_tokens(&self, text: &str) -> u32 {
        // Simple estimation: ~4 characters per token
        (text.len() / 4) as u32
    }
}

/// Provider registry for managing multiple providers
pub struct ProviderRegistry {
    providers: std::collections::HashMap<String, Box<dyn LLMProvider>>,
}

impl ProviderRegistry {
    pub fn new() -> Self {
        Self {
            providers: std::collections::HashMap::new(),
        }
    }
    
    pub fn register(&mut self, provider: Box<dyn LLMProvider>) {
        self.providers.insert(provider.name().to_string(), provider);
    }
    
    pub fn get(&self, name: &str) -> Option<&dyn LLMProvider> {
        self.providers.get(name).map(|p| p.as_ref())
    }
    
    pub fn list_providers(&self) -> Vec<&str> {
        self.providers.keys().map(|s| s.as_str()).collect()
    }
}

/// Error types for providers
#[derive(Debug, thiserror::Error)]
pub enum ProviderError {
    #[error("API key not configured")]
    MissingApiKey,
    
    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),
    
    #[error("Model not found: {0}")]
    ModelNotFound(String),
    
    #[error("Rate limit exceeded")]
    RateLimit,
    
    #[error("Network error: {0}")]
    Network(String),
    
    #[error("API error: {0}")]
    ApiError(String),
}