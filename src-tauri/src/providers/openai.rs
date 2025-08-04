use super::*;
use async_trait::async_trait;
use anyhow::{Result, Context};
use serde::{Deserialize, Serialize};
use futures::StreamExt;
use serde_json;

// OpenAI API structures
#[derive(Debug, Serialize)]
struct ChatCompletionRequest {
    model: String,
    messages: Vec<ChatMessage>,
    temperature: Option<f32>,
    max_tokens: Option<u32>,
    stream: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    functions: Option<Vec<OpenAIFunction>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    function_call: Option<serde_json::Value>,
}

#[derive(Debug, Serialize, Deserialize)]
struct ChatMessage {
    role: String,
    content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    function_call: Option<FunctionCallResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>, // For function role messages
}

#[derive(Debug, Serialize, Deserialize)]
struct OpenAIFunction {
    name: String,
    description: String,
    parameters: serde_json::Value,
}

#[derive(Debug, Serialize, Deserialize)]
struct FunctionCallResponse {
    name: String,
    arguments: String,
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
    function_call: Option<DeltaFunctionCall>,
}

#[derive(Debug, Deserialize)]
struct DeltaFunctionCall {
    name: Option<String>,
    arguments: Option<String>,
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
                MessageRole::Function => "function",
            }.to_string(),
            content: m.content.clone(),
            function_call: m.function_call.as_ref().map(|fc| FunctionCallResponse {
                name: fc.name.clone(),
                arguments: fc.arguments.clone(),
            }),
            name: if matches!(m.role, MessageRole::Function) {
                m.function_call.as_ref().map(|fc| fc.name.clone())
            } else {
                None
            },
        }).collect();
        
        let functions = request.functions.as_ref().map(|funcs| {
            funcs.iter().map(|f| OpenAIFunction {
                name: f.name.clone(),
                description: f.description.clone(),
                parameters: f.parameters.clone(),
            }).collect()
        });
        
        let function_call = request.tool_choice.as_ref().map(|tc| match tc {
            ToolChoice::None => serde_json::json!("none"),
            ToolChoice::Auto => serde_json::json!("auto"),
            ToolChoice::Function { name } => serde_json::json!({ "name": name }),
        });
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: false,
            functions,
            function_call,
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
        
        let choice = api_response.choices
            .first()
            .ok_or_else(|| anyhow::anyhow!("No choices in OpenAI response"))?;
        
        let content = choice.message.content.clone();
        let function_call = choice.message.function_call.as_ref().map(|fc| FunctionCall {
            name: fc.name.clone(),
            arguments: fc.arguments.clone(),
        });
        
        Ok(CompletionResponse {
            content,
            model: api_response.model,
            usage: api_response.usage.map(|u| TokenUsage {
                prompt_tokens: u.prompt_tokens,
                completion_tokens: u.completion_tokens,
                total_tokens: u.total_tokens,
            }),
            function_call,
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
                MessageRole::Function => "function",
            }.to_string(),
            content: m.content.clone(),
            function_call: m.function_call.as_ref().map(|fc| FunctionCallResponse {
                name: fc.name.clone(),
                arguments: fc.arguments.clone(),
            }),
            name: if matches!(m.role, MessageRole::Function) {
                m.function_call.as_ref().map(|fc| fc.name.clone())
            } else {
                None
            },
        }).collect();
        
        let functions = request.functions.as_ref().map(|funcs| {
            funcs.iter().map(|f| OpenAIFunction {
                name: f.name.clone(),
                description: f.description.clone(),
                parameters: f.parameters.clone(),
            }).collect()
        });
        
        let function_call = request.tool_choice.as_ref().map(|tc| match tc {
            ToolChoice::None => serde_json::json!("none"),
            ToolChoice::Auto => serde_json::json!("auto"),
            ToolChoice::Function { name } => serde_json::json!({ "name": name }),
        });
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: true,
            functions,
            function_call,
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
                let mut function_call_delta = None;
                
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
                                if let Some(fc) = &choice.delta.function_call {
                                    function_call_delta = Some(FunctionCallDelta {
                                        name: fc.name.clone(),
                                        arguments: fc.arguments.clone(),
                                    });
                                }
                                if let Some(reason) = &choice.finish_reason {
                                    finish_reason = Some(reason.clone());
                                }
                            }
                        }
                    }
                }
                
                Ok(StreamingResponse { delta, finish_reason, function_call_delta })
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