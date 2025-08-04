use super::*;
use async_trait::async_trait;
use anyhow::Result;
use futures::stream::{self, StreamExt};
use std::time::Duration;
use tokio::time::sleep;

pub struct DemoProvider {
    responses: Vec<(&'static str, &'static str)>,
}

impl DemoProvider {
    pub fn new() -> Self {
        Self {
            responses: vec![
                ("hello", "Hello! I'm AgentX Demo Mode. I'm a built-in assistant that works without any external API keys or services."),
                ("help", "I can help you with:\n• Basic programming questions\n• Explaining AgentX features\n• Setting up real LLM providers\n\nTo use real AI models, set up OpenAI, Anthropic, or Ollama."),
                ("rust", "Rust is a systems programming language focused on safety, speed, and concurrency. Here's a simple example:\n\n```rust\nfn main() {\n    println!(\"Hello from Rust!\");\n}\n```"),
                ("setup", "To set up real LLM providers:\n\n1. **OpenAI**: Set OPENAI_API_KEY environment variable\n2. **Anthropic**: Set ANTHROPIC_API_KEY environment variable\n3. **Ollama**: Install from https://ollama.ai and run `ollama pull llama2`\n\nThen restart AgentX!"),
                ("agentx", "AgentX is an AI IDE for agentic software development. Features include:\n• Multi-provider LLM support\n• Streaming responses\n• Progressive UI (Simple → Mission Control → Pro)\n• Keyboard shortcuts (F1 for help)\n• MCP protocol support"),
            ],
        }
    }
    
    fn find_response(&self, input: &str) -> String {
        let input_lower = input.to_lowercase();
        
        // Look for keyword matches
        for (keyword, response) in &self.responses {
            if input_lower.contains(keyword) {
                return response.to_string();
            }
        }
        
        // Default response
        format!(
            "I'm running in demo mode with limited capabilities. Your message: \"{}\"\n\n\
            I can answer basic questions about programming, Rust, and AgentX. \
            For full AI capabilities, please set up a real LLM provider (OpenAI, Anthropic, or Ollama).",
            input
        )
    }
}

impl Default for DemoProvider {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl LLMProvider for DemoProvider {
    fn name(&self) -> &str {
        "demo"
    }
    
    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        Ok(ProviderCapabilities {
            models: vec![
                ModelInfo {
                    id: "demo-assistant".to_string(),
                    name: "Demo Assistant".to_string(),
                    context_window: 1000,
                    input_cost_per_1k: Some(0.0),
                    output_cost_per_1k: Some(0.0),
                },
            ],
            supports_streaming: true,
            supports_functions: false,
            supports_system_messages: false,
        })
    }
    
    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse> {
        // Get the last user message
        let user_message = request.messages
            .iter()
            .rev()
            .find(|m| matches!(m.role, MessageRole::User))
            .map(|m| m.content.as_str())
            .unwrap_or("");
        
        let response = self.find_response(user_message);
        
        // Simulate some thinking time
        sleep(Duration::from_millis(300)).await;
        
        let response_len = response.len();
        let user_message_len = user_message.len();
        
        Ok(CompletionResponse {
            content: response,
            model: "demo-assistant".to_string(),
            usage: Some(TokenUsage {
                prompt_tokens: user_message_len as u32 / 4,
                completion_tokens: response_len as u32 / 4,
                total_tokens: (user_message_len + response_len) as u32 / 4,
            }),
        })
    }
    
    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream> {
        // Get the response
        let user_message = request.messages
            .iter()
            .rev()
            .find(|m| matches!(m.role, MessageRole::User))
            .map(|m| m.content.as_str())
            .unwrap_or("");
        
        let response = self.find_response(user_message);
        
        // Split into words for streaming
        let words: Vec<String> = response
            .split_whitespace()
            .map(|w| format!("{} ", w))
            .collect();
        
        // Create a stream that emits words with delays
        let stream = stream::iter(words.into_iter().enumerate())
            .then(|(_i, word)| async move {
                // Simulate typing speed
                sleep(Duration::from_millis(50)).await;
                
                Ok(StreamingResponse {
                    delta: word,
                    finish_reason: None,
                })
            })
            .chain(stream::once(async {
                Ok(StreamingResponse {
                    delta: String::new(),
                    finish_reason: Some("stop".to_string()),
                })
            }));
        
        Ok(Box::pin(stream))
    }
    
    async fn validate_config(&self) -> Result<()> {
        // Demo provider is always valid
        Ok(())
    }
}