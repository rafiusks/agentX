use super::*;
use async_trait::async_trait;
use anyhow::{Result, Context};
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::time::Duration;

/// OpenAI-compatible provider for local models (Ollama, LM Studio, llama.cpp, etc.)
pub struct OpenAICompatibleProvider {
    config: ProviderConfig,
    client: reqwest::Client,
}

impl OpenAICompatibleProvider {
    pub fn new(config: ProviderConfig) -> Result<Self> {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(
                config.timeout_secs.unwrap_or(300) // Longer timeout for local models
            ))
            .build()?;
            
        Ok(Self { config, client })
    }
    
    pub fn for_ollama() -> Result<Self> {
        let config = ProviderConfig {
            api_key: None, // Ollama doesn't need API keys
            base_url: Some("http://localhost:11434".to_string()),
            timeout_secs: Some(300),
            max_retries: Some(3),
        };
        Self::new(config)
    }
    
    pub fn for_lm_studio() -> Result<Self> {
        let config = ProviderConfig {
            api_key: None, // LM Studio doesn't need API keys
            base_url: Some("http://localhost:1234".to_string()),
            timeout_secs: Some(300),
            max_retries: Some(3),
        };
        Self::new(config)
    }
    
    pub fn for_llamacpp() -> Result<Self> {
        let config = ProviderConfig {
            api_key: None, // llama.cpp server doesn't need API keys
            base_url: Some("http://localhost:8080".to_string()),
            timeout_secs: Some(300),
            max_retries: Some(3),
        };
        Self::new(config)
    }
    
    async fn list_models(&self) -> Result<Vec<String>> {
        let base_url = self.get_base_url();
        let url = format!("{}/v1/models", base_url);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .context("Failed to list models")?;
            
        if !response.status().is_success() {
            return Err(anyhow::anyhow!("Failed to list models: {}", response.status()));
        }
        
        let models_response: ModelsResponse = response.json().await?;
        Ok(models_response.data.into_iter().map(|m| m.id).collect())
    }
    
    fn get_base_url(&self) -> &str {
        self.config.base_url.as_deref().unwrap_or("http://localhost:11434")
    }
}

// OpenAI API structures
#[derive(Debug, Serialize)]
struct ChatCompletionRequest {
    model: String,
    messages: Vec<ChatMessage>,
    temperature: Option<f32>,
    max_tokens: Option<u32>,
    stream: bool,
}

#[derive(Debug, Serialize, Deserialize)]
struct ChatMessage {
    role: String,
    content: String,
}

#[derive(Debug, Deserialize)]
struct ChatCompletionResponse {
    id: String,
    choices: Vec<Choice>,
    usage: Option<Usage>,
}

#[derive(Debug, Deserialize)]
struct Choice {
    message: ChatMessage,
    finish_reason: Option<String>,
}

#[derive(Debug, Deserialize)]
struct Usage {
    prompt_tokens: u32,
    completion_tokens: u32,
    total_tokens: u32,
}

#[derive(Debug, Deserialize)]
struct StreamingChunk {
    choices: Vec<StreamingChoice>,
}

#[derive(Debug, Deserialize)]
struct StreamingChoice {
    delta: Delta,
    finish_reason: Option<String>,
}

#[derive(Debug, Deserialize)]
struct Delta {
    content: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ModelsResponse {
    data: Vec<Model>,
}

#[derive(Debug, Deserialize)]
struct Model {
    id: String,
}

#[async_trait]
impl LLMProvider for OpenAICompatibleProvider {
    fn name(&self) -> &str {
        "openai-compatible"
    }
    
    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        // Try to dynamically fetch models
        let models = match self.list_models().await {
            Ok(model_ids) => {
                model_ids.into_iter().map(|id| ModelInfo {
                    id: id.clone(),
                    name: id,
                    context_window: 4096, // Default, varies by model
                    input_cost_per_1k: None, // Local models are free
                    output_cost_per_1k: None,
                }).collect()
            }
            Err(_) => {
                // Fallback to common models if API fails
                vec![
                    ModelInfo {
                        id: "llama2".to_string(),
                        name: "Llama 2".to_string(),
                        context_window: 4096,
                        input_cost_per_1k: None,
                        output_cost_per_1k: None,
                    },
                    ModelInfo {
                        id: "mistral".to_string(),
                        name: "Mistral".to_string(),
                        context_window: 8192,
                        input_cost_per_1k: None,
                        output_cost_per_1k: None,
                    },
                    ModelInfo {
                        id: "codellama".to_string(),
                        name: "Code Llama".to_string(),
                        context_window: 4096,
                        input_cost_per_1k: None,
                        output_cost_per_1k: None,
                    },
                ]
            }
        };
        
        Ok(ProviderCapabilities {
            models,
            supports_streaming: true,
            supports_functions: false, // Most local models don't support function calling
            supports_system_messages: true,
        })
    }
    
    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse> {
        let base_url = self.get_base_url();
        let url = format!("{}/v1/chat/completions", base_url);
        
        let messages: Vec<ChatMessage> = request.messages.iter().map(|m| ChatMessage {
            role: match m.role {
                MessageRole::System => "system".to_string(),
                MessageRole::User => "user".to_string(),
                MessageRole::Assistant => "assistant".to_string(),
            },
            content: m.content.clone(),
        }).collect();
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: false,
        };
        
        let response = self.client
            .post(&url)
            .json(&api_request)
            .send()
            .await
            .context("Failed to send request")?;
            
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("API error: {}", error_text));
        }
        
        let api_response: ChatCompletionResponse = response.json().await
            .context("Failed to parse response")?;
            
        let content = api_response.choices
            .first()
            .map(|c| c.message.content.clone())
            .unwrap_or_default();
            
        let usage = api_response.usage.map(|u| TokenUsage {
            prompt_tokens: u.prompt_tokens,
            completion_tokens: u.completion_tokens,
            total_tokens: u.total_tokens,
        });
        
        Ok(CompletionResponse {
            content,
            model: request.model,
            usage,
        })
    }
    
    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream> {
        let base_url = self.get_base_url();
        let url = format!("{}/v1/chat/completions", base_url);
        
        let messages: Vec<ChatMessage> = request.messages.iter().map(|m| ChatMessage {
            role: match m.role {
                MessageRole::System => "system".to_string(),
                MessageRole::User => "user".to_string(),
                MessageRole::Assistant => "assistant".to_string(),
            },
            content: m.content.clone(),
        }).collect();
        
        let api_request = ChatCompletionRequest {
            model: request.model,
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: true,
        };
        
        let response = self.client
            .post(&url)
            .json(&api_request)
            .send()
            .await
            .context("Failed to send streaming request")?;
            
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("API error: {}", error_text));
        }
        
        let stream = response.bytes_stream();
        
        let stream = stream.map(move |chunk_result| {
            match chunk_result {
                Ok(bytes) => {
                    // Parse SSE format
                    let text = String::from_utf8_lossy(&bytes);
                    
                    // Skip empty lines and "data: [DONE]" messages
                    if text.trim().is_empty() || text.contains("[DONE]") {
                        return Ok(StreamingResponse {
                            delta: String::new(),
                            finish_reason: Some("stop".to_string()),
                        });
                    }
                    
                    // Extract JSON from SSE format
                    if let Some(json_str) = text.strip_prefix("data: ") {
                        match serde_json::from_str::<StreamingChunk>(json_str.trim()) {
                            Ok(chunk) => {
                                let delta = chunk.choices
                                    .first()
                                    .and_then(|c| c.delta.content.clone())
                                    .unwrap_or_default();
                                    
                                let finish_reason = chunk.choices
                                    .first()
                                    .and_then(|c| c.finish_reason.clone());
                                    
                                Ok(StreamingResponse { delta, finish_reason })
                            }
                            Err(e) => {
                                // Log error but continue stream
                                eprintln!("Failed to parse streaming chunk: {}", e);
                                Ok(StreamingResponse {
                                    delta: String::new(),
                                    finish_reason: None,
                                })
                            }
                        }
                    } else {
                        Ok(StreamingResponse {
                            delta: String::new(),
                            finish_reason: None,
                        })
                    }
                }
                Err(e) => Err(anyhow::anyhow!("Stream error: {}", e)),
            }
        });
        
        Ok(Box::pin(stream))
    }
    
    async fn validate_config(&self) -> Result<()> {
        // Try to ping the server
        let base_url = self.get_base_url();
        let url = format!("{}/v1/models", base_url);
        
        let response = self.client
            .get(&url)
            .timeout(Duration::from_secs(5))
            .send()
            .await;
            
        match response {
            Ok(resp) if resp.status().is_success() => Ok(()),
            Ok(resp) => Err(anyhow::anyhow!("Server returned error: {}", resp.status())),
            Err(e) => Err(anyhow::anyhow!("Cannot connect to server at {}: {}", base_url, e)),
        }
    }
}