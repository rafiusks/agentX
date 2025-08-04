use super::*;
use async_trait::async_trait;
use anyhow::{Result, Context};
use serde::{Deserialize, Serialize};
use futures::StreamExt;

// Anthropic API structures
#[derive(Debug, Serialize)]
struct AnthropicRequest {
    model: String,
    messages: Vec<AnthropicMessage>,
    max_tokens: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    temperature: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    system: Option<String>,
    stream: bool,
}

#[derive(Debug, Serialize, Deserialize)]
struct AnthropicMessage {
    role: String,
    content: String,
}

#[derive(Debug, Deserialize)]
struct AnthropicResponse {
    id: String,
    model: String,
    content: Vec<ContentBlock>,
    usage: Usage,
}

#[derive(Debug, Deserialize)]
struct ContentBlock {
    #[serde(rename = "type")]
    content_type: String,
    text: String,
}

#[derive(Debug, Deserialize)]
struct Usage {
    input_tokens: u32,
    output_tokens: u32,
}

// Streaming response structures
#[derive(Debug, Deserialize)]
#[serde(tag = "type")]
enum StreamingEvent {
    #[serde(rename = "message_start")]
    MessageStart { message: MessageStart },
    #[serde(rename = "content_block_start")]
    ContentBlockStart { index: usize, content_block: ContentBlockStart },
    #[serde(rename = "content_block_delta")]
    ContentBlockDelta { index: usize, delta: Delta },
    #[serde(rename = "content_block_stop")]
    ContentBlockStop { index: usize },
    #[serde(rename = "message_delta")]
    MessageDelta { delta: MessageDeltaData, usage: Usage },
    #[serde(rename = "message_stop")]
    MessageStop,
}

#[derive(Debug, Deserialize)]
struct MessageStart {
    id: String,
    model: String,
    usage: Usage,
}

#[derive(Debug, Deserialize)]
struct ContentBlockStart {
    #[serde(rename = "type")]
    content_type: String,
    text: String,
}

#[derive(Debug, Deserialize)]
struct Delta {
    #[serde(rename = "type")]
    delta_type: String,
    text: String,
}

#[derive(Debug, Deserialize)]
struct MessageDeltaData {
    stop_reason: Option<String>,
}

pub struct AnthropicProvider {
    config: ProviderConfig,
    client: reqwest::Client,
}

impl AnthropicProvider {
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
impl LLMProvider for AnthropicProvider {
    fn name(&self) -> &str {
        "anthropic"
    }
    
    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        Ok(ProviderCapabilities {
            models: vec![
                ModelInfo {
                    id: "claude-3-opus-20240229".to_string(),
                    name: "Claude 3 Opus".to_string(),
                    context_window: 200000,
                    input_cost_per_1k: Some(0.015),
                    output_cost_per_1k: Some(0.075),
                },
                ModelInfo {
                    id: "claude-3-sonnet-20240229".to_string(),
                    name: "Claude 3 Sonnet".to_string(),
                    context_window: 200000,
                    input_cost_per_1k: Some(0.003),
                    output_cost_per_1k: Some(0.015),
                },
                ModelInfo {
                    id: "claude-3-haiku-20240307".to_string(),
                    name: "Claude 3 Haiku".to_string(),
                    context_window: 200000,
                    input_cost_per_1k: Some(0.00025),
                    output_cost_per_1k: Some(0.00125),
                },
            ],
            supports_streaming: true,
            supports_functions: false, // Claude doesn't have native function calling
            supports_system_messages: true,
        })
    }
    
    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse> {
        let api_key = self.config.api_key.as_ref()
            .ok_or(ProviderError::MissingApiKey)?;
        
        let url = format!("{}/messages", 
            self.config.base_url.as_deref().unwrap_or("https://api.anthropic.com/v1"));
        
        // Extract system message if present
        let (system_msg, messages): (Option<String>, Vec<AnthropicMessage>) = {
            let mut system = None;
            let mut msgs = Vec::new();
            
            for msg in &request.messages {
                match msg.role {
                    MessageRole::System => system = Some(msg.content.clone()),
                    MessageRole::User => msgs.push(AnthropicMessage {
                        role: "user".to_string(),
                        content: msg.content.clone(),
                    }),
                    MessageRole::Assistant => msgs.push(AnthropicMessage {
                        role: "assistant".to_string(),
                        content: msg.content.clone(),
                    }),
                    MessageRole::Function => {
                        // Anthropic doesn't support function role, include as user message
                        msgs.push(AnthropicMessage {
                            role: "user".to_string(),
                            content: format!("Function result: {}", msg.content),
                        })
                    }
                }
            }
            
            (system, msgs)
        };
        
        let api_request = AnthropicRequest {
            model: request.model.clone(),
            messages,
            max_tokens: request.max_tokens.unwrap_or(1024),
            temperature: request.temperature,
            system: system_msg,
            stream: false,
        };
        
        let response = self.client
            .post(&url)
            .header("x-api-key", api_key)
            .header("anthropic-version", "2023-06-01")
            .header("content-type", "application/json")
            .json(&api_request)
            .send()
            .await
            .context("Failed to send request to Anthropic")?;
        
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("Anthropic API error: {}", error_text));
        }
        
        let api_response: AnthropicResponse = response.json().await
            .context("Failed to parse Anthropic response")?;
        
        let content = api_response.content
            .iter()
            .map(|c| c.text.clone())
            .collect::<Vec<_>>()
            .join("");
        
        Ok(CompletionResponse {
            content,
            model: api_response.model,
            usage: Some(TokenUsage {
                prompt_tokens: api_response.usage.input_tokens,
                completion_tokens: api_response.usage.output_tokens,
                total_tokens: api_response.usage.input_tokens + api_response.usage.output_tokens,
            }),
            function_call: None, // Anthropic doesn't support function calling yet
        })
    }
    
    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream> {
        let api_key = self.config.api_key.as_ref()
            .ok_or(ProviderError::MissingApiKey)?;
        
        let url = format!("{}/messages", 
            self.config.base_url.as_deref().unwrap_or("https://api.anthropic.com/v1"));
        
        // Extract system message if present
        let (system_msg, messages): (Option<String>, Vec<AnthropicMessage>) = {
            let mut system = None;
            let mut msgs = Vec::new();
            
            for msg in &request.messages {
                match msg.role {
                    MessageRole::System => system = Some(msg.content.clone()),
                    MessageRole::User => msgs.push(AnthropicMessage {
                        role: "user".to_string(),
                        content: msg.content.clone(),
                    }),
                    MessageRole::Assistant => msgs.push(AnthropicMessage {
                        role: "assistant".to_string(),
                        content: msg.content.clone(),
                    }),
                    MessageRole::Function => {
                        // Anthropic doesn't support function role, include as user message
                        msgs.push(AnthropicMessage {
                            role: "user".to_string(),
                            content: format!("Function result: {}", msg.content),
                        })
                    }
                }
            }
            
            (system, msgs)
        };
        
        let api_request = AnthropicRequest {
            model: request.model.clone(),
            messages,
            max_tokens: request.max_tokens.unwrap_or(1024),
            temperature: request.temperature,
            system: system_msg,
            stream: true,
        };
        
        let response = self.client
            .post(&url)
            .header("x-api-key", api_key)
            .header("anthropic-version", "2023-06-01")
            .header("content-type", "application/json")
            .json(&api_request)
            .send()
            .await
            .context("Failed to send streaming request to Anthropic")?;
        
        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(anyhow::anyhow!("Anthropic API error: {}", error_text));
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
                        if let Ok(event) = serde_json::from_str::<StreamingEvent>(data) {
                            match event {
                                StreamingEvent::ContentBlockDelta { delta: block_delta, .. } => {
                                    delta.push_str(&block_delta.text);
                                }
                                StreamingEvent::MessageStop => {
                                    finish_reason = Some("stop".to_string());
                                }
                                _ => {}
                            }
                        }
                    }
                }
                
                Ok(StreamingResponse { delta, finish_reason, function_call_delta: None })
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