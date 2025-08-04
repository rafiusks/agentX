use anyhow::Result;
use async_trait::async_trait;
use std::sync::Arc;

use super::*;
use crate::mcp::manager::MCPManager;

/// Enhanced LLM provider that adds MCP tool support to any base provider
pub struct MCPEnhancedProvider {
    base_provider: Arc<dyn LLMProvider + Send + Sync>,
    mcp_manager: Arc<MCPManager>,
    provider_name: String,
}

impl MCPEnhancedProvider {
    pub fn new(
        base_provider: Arc<dyn LLMProvider + Send + Sync>,
        mcp_manager: Arc<MCPManager>,
    ) -> Self {
        let provider_name = format!("{}_with_mcp", base_provider.name());
        
        Self {
            base_provider,
            mcp_manager,
            provider_name,
        }
    }
}

#[async_trait]
impl LLMProvider for MCPEnhancedProvider {
    fn name(&self) -> &str {
        &self.provider_name
    }

    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        let mut caps = self.base_provider.capabilities().await?;
        // Enable functions if the base provider supports them
        caps.supports_functions = caps.supports_functions;
        Ok(caps)
    }

    async fn complete(&self, mut request: CompletionRequest) -> Result<CompletionResponse> {
        // Add MCP tools as functions if the provider supports functions
        if request.functions.is_none() {
            let caps = self.capabilities().await?;
            if caps.supports_functions {
                if let Ok(functions) = self.mcp_manager.get_functions().await {
                    if !functions.is_empty() {
                        request.functions = Some(functions);
                        // Set tool choice to auto to let the AI decide when to use tools
                        if request.tool_choice.is_none() {
                            request.tool_choice = Some(ToolChoice::Auto);
                        }
                    }
                }
            }
        }

        // Get completion from base provider
        let mut response = self.base_provider.complete(request).await?;

        // If the AI wants to call a function, execute it via MCP
        if let Some(function_call) = &response.function_call {
            match self.mcp_manager.execute_function(function_call.clone()).await {
                Ok(result) => {
                    // For now, append the result to the response content
                    // In a full implementation, you'd want to continue the conversation
                    if response.content.is_empty() {
                        response.content = result;
                    } else {
                        response.content = format!("{}\n\nTool Result:\n{}", response.content, result);
                    }
                }
                Err(e) => {
                    let error_msg = format!("Tool execution failed: {}", e);
                    if response.content.is_empty() {
                        response.content = error_msg;
                    } else {
                        response.content = format!("{}\n\nTool Error:\n{}", response.content, error_msg);
                    }
                }
            }
        }

        Ok(response)
    }

    async fn stream_complete(&self, mut request: CompletionRequest) -> Result<ResponseStream> {
        // Add MCP tools as functions if the provider supports functions
        if request.functions.is_none() {
            let caps = self.capabilities().await?;
            if caps.supports_functions {
                if let Ok(functions) = self.mcp_manager.get_functions().await {
                    if !functions.is_empty() {
                        request.functions = Some(functions);
                        if request.tool_choice.is_none() {
                            request.tool_choice = Some(ToolChoice::Auto);
                        }
                    }
                }
            }
        }

        // For streaming, we'll handle function calls in a follow-up
        // The base provider will stream the function call, and we can execute it
        // when the stream completes. This is a simplified approach.
        self.base_provider.stream_complete(request).await
    }

    async fn validate_config(&self) -> Result<()> {
        self.base_provider.validate_config().await
    }

    fn estimate_tokens(&self, text: &str) -> u32 {
        self.base_provider.estimate_tokens(text)
    }
}