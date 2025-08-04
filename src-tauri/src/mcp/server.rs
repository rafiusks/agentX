use anyhow::{Result, anyhow};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use tokio::process::{Command, Child};
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use std::process::Stdio;
use futures::stream;
use std::sync::Arc;
use tokio::sync::Mutex;

use super::{MCPTool, MCPToolCall, MCPToolResult};
use crate::providers::{LLMProvider, CompletionRequest, CompletionResponse, ResponseStream, StreamingResponse, MessageRole, ProviderCapabilities, ModelInfo};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MCPServerConfig {
    pub name: String,
    pub command: String,
    pub args: Vec<String>,
    pub env: Option<std::collections::HashMap<String, String>>,
    pub capabilities: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct JsonRpcRequest {
    jsonrpc: String,
    id: u64,
    method: String,
    params: serde_json::Value,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct JsonRpcResponse {
    jsonrpc: String,
    id: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<JsonRpcError>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct JsonRpcNotification {
    jsonrpc: String,
    method: String,
    params: serde_json::Value,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(untagged)]
pub enum JsonRpcMessage {
    Request(JsonRpcRequest),
    Response(JsonRpcResponse),
    Notification(JsonRpcNotification),
}

#[derive(Debug, Serialize, Deserialize)]
pub struct JsonRpcError {
    code: i32,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<serde_json::Value>,
}

#[derive(Debug)]
pub struct MCPServer {
    config: MCPServerConfig,
    process: Arc<Mutex<Option<Child>>>,
    request_id: Arc<Mutex<u64>>,
}

impl MCPServer {
    pub async fn new(config: MCPServerConfig) -> Result<Self> {
        Ok(Self {
            config,
            process: Arc::new(Mutex::new(None)),
            request_id: Arc::new(Mutex::new(0)),
        })
    }

    pub async fn connect(&self) -> Result<()> {
        let mut cmd = Command::new(&self.config.command);
        cmd.args(&self.config.args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        if let Some(env) = &self.config.env {
            for (key, value) in env {
                cmd.env(key, value);
            }
        }

        let child = cmd.spawn()?;
        
        // Scope the process lock to avoid deadlock
        {
            let mut process = self.process.lock().await;
            *process = Some(child);
            
            // Check if process is still running and pipes are available
            if let Some(ref mut child) = *process {
                match child.try_wait() {
                    Ok(Some(status)) => {
                        return Err(anyhow!("MCP server process exited immediately with status: {}", status));
                    }
                    Ok(None) => {
                        println!("MCP server process is running");
                    }
                    Err(e) => {
                        return Err(anyhow!("Failed to check process status: {}", e));
                    }
                }
                
                // Check if pipes are available
                println!("Checking pipes - stdin: {}, stdout: {}, stderr: {}", 
                    child.stdin.is_some(), 
                    child.stdout.is_some(), 
                    child.stderr.is_some()
                );
            }
        } // Lock is released here

        // Send initialization request
        self.initialize().await?;

        Ok(())
    }

    async fn initialize(&self) -> Result<()> {
        let init_params = serde_json::json!({
            "protocolVersion": "0.1.0",
            "clientInfo": {
                "name": "AgentX",
                "version": "0.1.0"
            },
            "capabilities": {
                "sampling": {},
                "roots": {
                    "listChanged": true
                }
            }
        });

        let response = self.send_request("initialize", init_params).await?;
        tracing::info!("MCP server initialized: {:?}", response);

        // Send initialized notification
        self.send_notification("initialized", serde_json::json!({})).await?;

        Ok(())
    }

    async fn send_request(&self, method: &str, params: serde_json::Value) -> Result<serde_json::Value> {
        let mut id_lock = self.request_id.lock().await;
        let id = *id_lock;
        *id_lock += 1;
        drop(id_lock);

        let request = JsonRpcRequest {
            jsonrpc: "2.0".to_string(),
            id,
            method: method.to_string(),
            params,
        };

        println!("Sending MCP request: {} with ID {}", method, id);

        println!("Attempting to acquire process lock...");
        let mut process_lock = self.process.lock().await;
        println!("Process lock acquired");
        if let Some(process) = process_lock.as_mut() {
            println!("Process found in lock");
            if let Some(stdin) = process.stdin.as_mut() {
                println!("Stdin found, proceeding with JSON send");
                let msg = serde_json::to_string(&request)?;
                println!("Sending JSON: {}", msg);
                match stdin.write_all(msg.as_bytes()).await {
                    Ok(()) => println!("Successfully wrote JSON to stdin"),
                    Err(e) => return Err(anyhow!("Failed to write to stdin: {}", e)),
                }
                match stdin.write_all(b"\n").await {
                    Ok(()) => println!("Successfully wrote newline to stdin"),
                    Err(e) => return Err(anyhow!("Failed to write newline to stdin: {}", e)),
                }
                match stdin.flush().await {
                    Ok(()) => println!("Successfully flushed stdin"),
                    Err(e) => return Err(anyhow!("Failed to flush stdin: {}", e)),
                }
            } else {
                return Err(anyhow!("No stdin available on process"));
            }
            
            // Check if process is still alive after sending data
            match process.try_wait() {
                Ok(Some(status)) => {
                    return Err(anyhow!("MCP server process exited after sending data with status: {}", status));
                }
                Ok(None) => {
                    println!("MCP server process still running after sending data");
                }
                Err(e) => {
                    return Err(anyhow!("Failed to check process status after sending: {}", e));
                }
            }

            // Read response with timeout
            if let Some(stdout) = process.stdout.take() {
                println!("Starting to read from stdout...");
                let mut reader = BufReader::new(stdout);
                let mut line = String::new();
                
                // Add timeout to prevent hanging
                let timeout_duration = std::time::Duration::from_secs(10);
                let result = tokio::time::timeout(timeout_duration, async {
                    println!("Inside timeout block, starting read loop...");
                    
                    // First, let's try to read just one line to see if anything comes through
                    match reader.read_line(&mut line).await {
                        Ok(0) => {
                            println!("EOF reached immediately - no data available");
                            return Err(anyhow!("EOF reached immediately"));
                        }
                        Ok(n) => {
                            println!("Read {} bytes: '{}'", n, line.trim());
                        }
                        Err(e) => {
                            println!("Error reading first line: {}", e);
                            return Err(anyhow!("Read error: {}", e));
                        }
                    }
                    
                    // Process the line we just read
                    if !line.trim().is_empty() {
                        if let Ok(msg) = serde_json::from_str::<JsonRpcMessage>(&line) {
                            match msg {
                                JsonRpcMessage::Response(response) => {
                                    if response.id == id {
                                        if let Some(error) = response.error {
                                            return Err(anyhow!("MCP error: {} - {}", error.code, error.message));
                                        }
                                        return response.result.ok_or_else(|| anyhow!("Empty response"));
                                    }
                                }
                                JsonRpcMessage::Notification(notif) => {
                                    println!("Received notification: {}", notif.method);
                                }
                                JsonRpcMessage::Request(req) => {
                                    println!("Received unexpected request: {}", req.method);
                                }
                            }
                        } else {
                            println!("Failed to parse JSON: {}", line);
                        }
                    }
                    
                    // Continue reading more lines
                    line.clear();
                    while reader.read_line(&mut line).await? > 0 {
                        println!("Received additional line: '{}'", line.trim());
                        if let Ok(msg) = serde_json::from_str::<JsonRpcMessage>(&line) {
                            match msg {
                                JsonRpcMessage::Response(response) => {
                                    if response.id == id {
                                        if let Some(error) = response.error {
                                            return Err(anyhow!("MCP error: {} - {}", error.code, error.message));
                                        }
                                        return response.result.ok_or_else(|| anyhow!("Empty response"));
                                    }
                                }
                                JsonRpcMessage::Notification(notif) => {
                                    println!("Received notification: {}", notif.method);
                                }
                                JsonRpcMessage::Request(req) => {
                                    println!("Received unexpected request: {}", req.method);
                                }
                            }
                        } else {
                            println!("Failed to parse JSON: {}", line);
                        }
                        line.clear();
                    }
                    Err(anyhow!("No response received"))
                }).await;
                
                // Restore stdout regardless of result
                process.stdout = Some(reader.into_inner());
                
                match result {
                    Ok(inner_result) => inner_result,
                    Err(_) => Err(anyhow!("Timeout waiting for MCP server response"))
                }
            } else {
                Err(anyhow!("No stdout available"))
            }
        } else {
            Err(anyhow!("No process available"))
        }
    }

    async fn send_notification(&self, method: &str, params: serde_json::Value) -> Result<()> {
        let notification = JsonRpcNotification {
            jsonrpc: "2.0".to_string(),
            method: method.to_string(),
            params,
        };

        let mut process_lock = self.process.lock().await;
        if let Some(process) = process_lock.as_mut() {
            if let Some(stdin) = process.stdin.as_mut() {
                let msg = serde_json::to_string(&notification)?;
                stdin.write_all(msg.as_bytes()).await?;
                stdin.write_all(b"\n").await?;
                stdin.flush().await?;
            }
        }

        Ok(())
    }

    /// List available tools from this MCP server
    pub async fn list_tools(&self) -> Result<Vec<MCPTool>> {
        println!("Listing tools for MCP server...");
        let response = self.send_request("tools/list", serde_json::json!({})).await?;
        println!("Tools list response: {:?}", response);
        
        let tools_array = response.get("tools")
            .and_then(|t| t.as_array())
            .ok_or_else(|| anyhow!("Invalid tools response format"))?;

        println!("Found {} tools in response", tools_array.len());
        let mut tools = Vec::new();
        for (i, tool_value) in tools_array.iter().enumerate() {
            println!("Processing tool {}: {:?}", i, tool_value);
            if let Ok(tool) = serde_json::from_value::<MCPTool>(tool_value.clone()) {
                println!("Successfully parsed tool: {:?}", tool);
                tools.push(tool);
            } else {
                println!("Failed to parse tool {}: {:?}", i, tool_value);
            }
        }

        println!("Returning {} tools", tools.len());
        Ok(tools)
    }

    /// Call a tool on this MCP server
    pub async fn call_tool(&self, tool_call: MCPToolCall) -> Result<MCPToolResult> {
        let params = serde_json::json!({
            "name": tool_call.name,
            "arguments": tool_call.arguments
        });

        let response = self.send_request("tools/call", params).await?;
        
        // Parse the tool result
        let result: MCPToolResult = serde_json::from_value(response)
            .map_err(|e| anyhow!("Failed to parse tool result: {}", e))?;

        Ok(result)
    }

    pub async fn disconnect(&self) -> Result<()> {
        let mut process_lock = self.process.lock().await;
        if let Some(mut process) = process_lock.take() {
            process.kill().await?;
        }
        Ok(())
    }
}

#[async_trait]
impl LLMProvider for MCPServer {
    fn name(&self) -> &str {
        &self.config.name
    }

    async fn capabilities(&self) -> Result<ProviderCapabilities> {
        Ok(ProviderCapabilities {
            models: vec![ModelInfo {
                id: "mcp-default".to_string(),
                name: "MCP Default Model".to_string(),
                context_window: 128000,
                input_cost_per_1k: None,
                output_cost_per_1k: None,
            }],
            supports_streaming: true,
            supports_functions: true,
            supports_system_messages: true,
        })
    }

    async fn complete(&self, request: CompletionRequest) -> Result<CompletionResponse> {
        // Convert our request to MCP format
        let messages: Vec<serde_json::Value> = request.messages.iter().map(|msg| {
            serde_json::json!({
                "role": match msg.role {
                    MessageRole::System => "system",
                    MessageRole::User => "user",
                    MessageRole::Assistant => "assistant",
                    MessageRole::Function => "function",
                },
                "content": {
                    "type": "text",
                    "text": &msg.content
                }
            })
        }).collect();

        let params = serde_json::json!({
            "messages": messages,
            "modelPreferences": {
                "hints": [{
                    "name": request.model
                }]
            },
            "systemPrompt": "You are a helpful AI assistant",
            "includeContext": "thisConversation",
            "temperature": request.temperature,
            "maxTokens": request.max_tokens.unwrap_or(1000)
        });

        let response = self.send_request("sampling/createMessage", params).await?;
        
        // Extract content from response
        let content = response["content"]["text"]
            .as_str()
            .unwrap_or("No response")
            .to_string();

        Ok(CompletionResponse {
            content,
            model: response["model"].as_str().unwrap_or(&request.model).to_string(),
            usage: None,
            function_call: None,
        })
    }

    async fn stream_complete(&self, request: CompletionRequest) -> Result<ResponseStream> {
        // For now, simulate streaming by getting complete response and chunking it
        let response = self.complete(request).await?;
        
        // Split response into chunks
        let chunks: Vec<_> = response.content
            .split_whitespace()
            .map(|word| StreamingResponse {
                delta: format!("{} ", word),
                finish_reason: None,
                function_call_delta: None,
            })
            .collect();

        // Add final chunk
        let mut chunks = chunks;
        if let Some(last) = chunks.last_mut() {
            last.finish_reason = Some("stop".to_string());
        }

        Ok(Box::pin(stream::iter(chunks.into_iter().map(Ok))))
    }

    async fn validate_config(&self) -> Result<()> {
        let process_lock = self.process.lock().await;
        if process_lock.is_none() {
            return Err(anyhow!("MCP server not connected"));
        }
        Ok(())
    }
}

impl MCPServer {
    pub async fn is_available(&self) -> bool {
        let process_lock = self.process.lock().await;
        process_lock.is_some()
    }
}

#[derive(Debug)]
pub struct MCPRegistry {
    servers: Arc<Mutex<Vec<Arc<MCPServer>>>>,
}

impl MCPRegistry {
    pub fn new() -> Self {
        Self {
            servers: Arc::new(Mutex::new(Vec::new())),
        }
    }

    pub async fn register_server(&self, config: MCPServerConfig) -> Result<()> {
        let server = Arc::new(MCPServer::new(config).await?);
        server.connect().await?;
        
        let mut servers = self.servers.lock().await;
        servers.push(server);
        
        Ok(())
    }

    pub async fn get_server(&self, name: &str) -> Option<Arc<MCPServer>> {
        let servers = self.servers.lock().await;
        servers.iter().find(|s| s.config.name == name).cloned()
    }

    pub async fn list_servers(&self) -> Vec<String> {
        let servers = self.servers.lock().await;
        servers.iter().map(|s| s.config.name.clone()).collect()
    }

    pub async fn disconnect_all(&self) -> Result<()> {
        let servers = self.servers.lock().await;
        for server in servers.iter() {
            server.disconnect().await?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_mcp_config() {
        let config = MCPServerConfig {
            name: "test-server".to_string(),
            command: "node".to_string(),
            args: vec!["mcp-server.js".to_string()],
            env: None,
            capabilities: vec!["sampling".to_string()],
        };

        let server = MCPServer::new(config).await.unwrap();
        assert!(!server.is_available().await);
    }
}