use super::*;
use async_trait::async_trait;
use anyhow::{Result, Context};
use serde::{Deserialize, Serialize};
use futures::StreamExt;

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
    model: String,
    choices: Vec<Choice>,
    usage: Option<Usage>,
}

#[derive(Debug, Deserialize)]
struct Choice {
    message: ChatMessage,
    finish_reason: Option<String>,
    index: i32,
}

#[derive(Debug, Deserialize)]
struct Usage {
    prompt_tokens: u32,
    completion_tokens: u32,
    total_tokens: u32,
}

// Streaming response structures
#[derive(Debug, Deserialize)]
struct StreamChunk {
    id: String,
    model: String,
    choices: Vec<StreamChoice>,
}

#[derive(Debug, Deserialize)]
struct StreamChoice {
    delta: DeltaContent,
    finish_reason: Option<String>,
    index: i32,
}

#[derive(Debug, Deserialize)]
struct DeltaContent {
    content: Option<String>,
    role: Option<String>,
}

pub struct OpenAIProvider {
    config: ProviderConfig,
    client: reqwest::Client,
}

impl OpenAIProvider {
    pub fn new(config: ProviderConfig) -> Result<Self> {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(
                config.timeout_secs.unwrap_or(30)
            ))
            .build()?;
            
        Ok(Self { config, client })
    }
}

#[async_trait]
impl LLMProvider for OpenAIProvider {
    fn name(&self) -> &str {
        "openai"
    }
    
    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        Ok(ProviderCapabilities {
            models: vec![
                ModelInfo {
                    id: "gpt-4".to_string(),
                    name: "GPT-4".to_string(),
                    context_window: 8192,
                    input_cost_per_1k: Some(0.03),
                    output_cost_per_1k: Some(0.06),
                },
                ModelInfo {
                    id: "gpt-4-turbo-preview".to_string(),
                    name: "GPT-4 Turbo".to_string(),
                    context_window: 128000,
                    input_cost_per_1k: Some(0.01),
                    output_cost_per_1k: Some(0.03),
                },
                ModelInfo {
                    id: "gpt-3.5-turbo".to_string(),
                    name: "GPT-3.5 Turbo".to_string(),
                    context_window: 16385,
                    input_cost_per_1k: Some(0.0005),
                    output_cost_per_1k: Some(0.0015),
                },
            ],
            supports_streaming: true,
            supports_functions: true,
            supports_system_messages: true,
        })
    }
    
    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse> {
        let api_key = self.config.api_key.as_ref()
            .ok_or(ProviderError::MissingApiKey)?;
        
        let url = format!("{}/chat/completions", 
            self.config.base_url.as_deref().unwrap_or("https://api.openai.com/v1"));
        
        let messages: Vec<ChatMessage> = request.messages.iter().map(|m| ChatMessage {
            role: match m.role {
                MessageRole::System => "system",
                MessageRole::User => "user",
                MessageRole::Assistant => "assistant",
            }.to_string(),
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
            .bearer_auth(api_key)
            .json(&api_request)
            .send()
            .await
            .context("Failed to send request to OpenAI")?;
        
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("OpenAI API error: {}", error_text));
        }
        
        let api_response: ChatCompletionResponse = response.json().await
            .context("Failed to parse OpenAI response")?;
        
        let content = api_response.choices
            .first()
            .map(|c| c.message.content.clone())
            .unwrap_or_default();
        
        Ok(CompletionResponse {
            content,
            model: api_response.model,
            usage: api_response.usage.map(|u| TokenUsage {
                prompt_tokens: u.prompt_tokens,
                completion_tokens: u.completion_tokens,
                total_tokens: u.total_tokens,
            }),
        })
    }
    
    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream> {
        let api_key = self.config.api_key.as_ref()
            .ok_or(ProviderError::MissingApiKey)?;
        
        let url = format!("{}/chat/completions", 
            self.config.base_url.as_deref().unwrap_or("https://api.openai.com/v1"));
        
        let messages: Vec<ChatMessage> = request.messages.iter().map(|m| ChatMessage {
            role: match m.role {
                MessageRole::System => "system",
                MessageRole::User => "user",
                MessageRole::Assistant => "assistant",
            }.to_string(),
            content: m.content.clone(),
        }).collect();
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: true,
        };
        
        let response = self.client
            .post(&url)
            .bearer_auth(api_key)
            .json(&api_request)
            .send()
            .await
            .context("Failed to send streaming request to OpenAI")?;
        
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("OpenAI API error: {}", error_text));
        }
        
        let stream = response.bytes_stream()
            .map(move |chunk| {
                let chunk = chunk?;
                let text = String::from_utf8_lossy(&chunk);
                
                // Parse SSE format
                let mut delta = String::new();
                let mut finish_reason = None;
                
                for line in text.lines() {
                    if line.starts_with("data: ") {
                        let data = &line[6..];
                        if data == "[DONE]" {
                            finish_reason = Some("stop".to_string());
                        } else if let Ok(parsed) = serde_json::from_str::<StreamChunk>(data) {
                            if let Some(choice) = parsed.choices.first() {
                                if let Some(content) = &choice.delta.content {
                                    delta.push_str(content);
                                }
                                if let Some(reason) = &choice.finish_reason {
                                    finish_reason = Some(reason.clone());
                                }
                            }
                        }
                    }
                }
                
                Ok(StreamingResponse { delta, finish_reason })
            });
        
        Ok(Box::pin(stream))
    }
    
    async fn validate_config(&self) -> Result<()> {
        if self.config.api_key.is_none() {
            return Err(ProviderError::MissingApiKey.into());
        }
        Ok(())
    }
}