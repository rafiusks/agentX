use super::*;
use async_trait::async_trait;
use anyhow::{Result, Context};
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::time::Duration;
use std::sync::{Arc, Mutex};

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
    #[serde(skip_serializing_if = "Option::is_none")]
    functions: Option<Vec<FunctionDef>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    function_call: Option<FunctionCallOption>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tools: Option<Vec<Tool>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tool_choice: Option<ToolChoiceOption>,
}

#[derive(Debug, Serialize)]
struct FunctionDef {
    name: String,
    description: String,
    parameters: serde_json::Value,
}

#[derive(Debug, Serialize)]
#[serde(untagged)]
enum FunctionCallOption {
    String(String),
    Object { name: String },
}

impl FunctionCallOption {
    fn none() -> Self {
        FunctionCallOption::String("none".to_string())
    }
    
    fn auto() -> Self {
        FunctionCallOption::String("auto".to_string())
    }
    
    fn function(name: String) -> Self {
        FunctionCallOption::Object { name }
    }
}

#[derive(Debug, Serialize)]
struct Tool {
    #[serde(rename = "type")]
    tool_type: String,
    function: FunctionDef,
}

#[derive(Debug, Serialize)]
#[serde(untagged)]
enum ToolChoiceOption {
    None,
    Auto,
    Tool { #[serde(rename = "type")] tool_type: String, function: ToolChoice },
}

#[derive(Debug, Serialize)]
struct ToolChoice {
    name: String,
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
    message: ResponseMessage,
    finish_reason: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ResponseMessage {
    role: String,
    content: Option<String>,
    #[serde(default)]
    function_call: Option<FunctionCallResponse>,
    #[serde(default)]
    tool_calls: Option<Vec<ToolCallResponse>>,
}

#[derive(Debug, Deserialize)]
struct FunctionCallResponse {
    name: String,
    arguments: String,
}

#[derive(Debug, Deserialize)]
struct ToolCallResponse {
    id: String,
    #[serde(rename = "type")]
    tool_type: String,
    function: FunctionCallResponse,
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
    #[serde(default)]
    function_call: Option<FunctionCallDelta>,
    #[serde(default)]
    tool_calls: Option<Vec<ToolCallDelta>>,
}

#[derive(Debug, Deserialize)]
struct FunctionCallDelta {
    name: Option<String>,
    arguments: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ToolCallDelta {
    index: Option<usize>,
    id: Option<String>,
    #[serde(rename = "type")]
    tool_type: Option<String>,
    function: Option<FunctionCallDelta>,
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
        
        // Check if we're running Ollama and if the models support functions
        let supports_functions = if self.get_base_url().contains("11434") {
            // This is likely Ollama, check model capabilities
            models.iter().any(|model| {
                super::ollama_models::model_supports_functions(&model.id)
            })
        } else {
            // For other OpenAI-compatible servers, assume they support functions
            true
        };
        
        Ok(ProviderCapabilities {
            models,
            supports_streaming: true,
            supports_functions,
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
                MessageRole::Function => "function".to_string(),
            },
            content: m.content.clone(),
        }).collect();
        
        // Convert functions to the format expected by OpenAI API
        let (functions, function_call) = if let Some(funcs) = request.functions {
            let api_functions: Vec<FunctionDef> = funcs.iter().map(|f| FunctionDef {
                name: f.name.clone(),
                description: f.description.clone(),
                parameters: f.parameters.clone(),
            }).collect();
            
            let call_option = match request.tool_choice {
                Some(super::ToolChoice::None) => Some(FunctionCallOption::none()),
                Some(super::ToolChoice::Auto) => Some(FunctionCallOption::auto()),
                Some(super::ToolChoice::Function { name }) => Some(FunctionCallOption::function(name)),
                None => Some(FunctionCallOption::auto()),
            };
            
            (Some(api_functions), call_option)
        } else {
            (None, None)
        };
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: false,
            functions,
            function_call,
            tools: None, // We'll use functions for now, not tools
            tool_choice: None,
        };
        
        // Debug logging
        println!("[OpenAI-Compatible] Sending completion request to {}", url);
        println!("[OpenAI-Compatible] Model: {}", api_request.model);
        if let Some(ref funcs) = api_request.functions {
            println!("[OpenAI-Compatible] Functions: {} functions included", funcs.len());
            for func in funcs {
                println!("[OpenAI-Compatible]   - {}: {}", func.name, func.description);
            }
            // Also log the full request for debugging
            println!("[OpenAI-Compatible] Full request: {}", serde_json::to_string_pretty(&api_request).unwrap_or_default());
        } else {
            println!("[OpenAI-Compatible] No functions included in request");
        }
        
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
            
        let choice = api_response.choices
            .into_iter()
            .next()
            .ok_or_else(|| anyhow::anyhow!("No choices in response"))?;
            
        let raw_content = choice.message.content.unwrap_or_default();
        
        // Check for standard OpenAI function call format first
        let mut function_call = choice.message.function_call.map(|fc| super::FunctionCall {
            name: fc.name,
            arguments: fc.arguments,
        });
        
        // If no standard function call, check for Deepseek format
        let content = if function_call.is_none() && raw_content.contains("[TOOL_REQUEST]") {
            if let Some(calls) = super::deepseek_parser::parse_deepseek_tool_calls(&raw_content) {
                // Take the first tool call
                if let Some(first_call) = calls.into_iter().next() {
                    function_call = Some(first_call);
                }
            }
            // Return content without the tool request markers
            super::deepseek_parser::extract_content_without_tools(&raw_content)
        } else {
            raw_content
        };
            
        let usage = api_response.usage.map(|u| TokenUsage {
            prompt_tokens: u.prompt_tokens,
            completion_tokens: u.completion_tokens,
            total_tokens: u.total_tokens,
        });
        
        Ok(CompletionResponse {
            content,
            model: request.model,
            usage,
            function_call,
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
                MessageRole::Function => "function".to_string(),
            },
            content: m.content.clone(),
        }).collect();
        
        // Convert functions to the format expected by OpenAI API
        let (functions, function_call) = if let Some(funcs) = request.functions {
            let api_functions: Vec<FunctionDef> = funcs.iter().map(|f| FunctionDef {
                name: f.name.clone(),
                description: f.description.clone(),
                parameters: f.parameters.clone(),
            }).collect();
            
            let call_option = match request.tool_choice {
                Some(super::ToolChoice::None) => Some(FunctionCallOption::none()),
                Some(super::ToolChoice::Auto) => Some(FunctionCallOption::auto()),
                Some(super::ToolChoice::Function { name }) => Some(FunctionCallOption::function(name)),
                None => Some(FunctionCallOption::auto()),
            };
            
            (Some(api_functions), call_option)
        } else {
            (None, None)
        };
        
        let api_request = ChatCompletionRequest {
            model: request.model.clone(),
            messages,
            temperature: request.temperature,
            max_tokens: request.max_tokens,
            stream: true,
            functions,
            function_call,
            tools: None,
            tool_choice: None,
        };
        
        // Debug logging
        println!("[OpenAI-Compatible] Sending streaming request to {}", url);
        println!("[OpenAI-Compatible] Model: {}", api_request.model);
        
        // Check if model supports functions
        let model_supports_functions = super::ollama_models::model_supports_functions(&api_request.model);
        
        if let Some(ref funcs) = api_request.functions {
            println!("[OpenAI-Compatible] Functions: {} functions included", funcs.len());
            if !model_supports_functions && self.get_base_url().contains("11434") {
                println!("[OpenAI-Compatible] ⚠️  WARNING: Model '{}' does not support function calling!", api_request.model);
                println!("[OpenAI-Compatible] ⚠️  Try: ollama pull mistral (or llama3.1, mixtral, qwen2.5)");
            }
            for func in funcs {
                println!("[OpenAI-Compatible]   - {}: {}", func.name, func.description);
            }
        } else {
            println!("[OpenAI-Compatible] No functions included in request");
        }
        
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
        
        // For accumulating Deepseek content
        let accumulated_content = Arc::new(Mutex::new(String::new()));
        let is_deepseek = request.model.contains("deepseek");
        
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
                            function_call_delta: None,
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
                                    
                                // For Deepseek, accumulate content to check for tool calls
                                if is_deepseek && !delta.is_empty() {
                                    let mut acc = accumulated_content.lock().unwrap();
                                    acc.push_str(&delta);
                                }
                                    
                                // Check for standard function call deltas
                                let mut function_call_delta = chunk.choices
                                    .first()
                                    .and_then(|c| c.delta.function_call.as_ref())
                                    .map(|fc| super::FunctionCallDelta {
                                        name: fc.name.clone(),
                                        arguments: fc.arguments.clone(),
                                    });
                                    
                                // If finishing and using Deepseek, check for tool calls
                                let mut processed_delta = delta.clone();
                                if is_deepseek && finish_reason.is_some() {
                                    let acc = accumulated_content.lock().unwrap();
                                    if acc.contains("[TOOL_REQUEST]") {
                                        if let Some(calls) = super::deepseek_parser::parse_deepseek_tool_calls(&acc) {
                                            if let Some(first_call) = calls.into_iter().next() {
                                                function_call_delta = Some(super::FunctionCallDelta {
                                                    name: Some(first_call.name),
                                                    arguments: Some(first_call.arguments),
                                                });
                                            }
                                        }
                                        // Don't show the raw tool syntax to user
                                        processed_delta = super::deepseek_parser::extract_content_without_tools(&acc);
                                    }
                                }
                                    
                                // Hide tool request syntax while streaming
                                if is_deepseek && (delta.contains("[TOOL_REQUEST]") || delta.contains("[END_TOOL_REQUEST]")) {
                                    processed_delta = String::new();
                                }
                                    
                                Ok(StreamingResponse { 
                                    delta: processed_delta, 
                                    finish_reason, 
                                    function_call_delta 
                                })
                            }
                            Err(e) => {
                                // Log error but continue stream
                                eprintln!("Failed to parse streaming chunk: {}", e);
                                Ok(StreamingResponse {
                                    delta: String::new(),
                                    finish_reason: None,
                                    function_call_delta: None,
                                })
                            }
                        }
                    } else {
                        Ok(StreamingResponse {
                            delta: String::new(),
                            finish_reason: None,
                            function_call_delta: None,
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