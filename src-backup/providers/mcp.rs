use anyhow::{Result, anyhow};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use tokio::process::{Command, Child};
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use std::process::Stdio;
use futures::stream;
use std::sync::Arc;
use tokio::sync::Mutex;

use super::{LLMProvider, CompletionRequest, CompletionResponse, ResponseStream, StreamingResponse, MessageRole, ProviderCapabilities, ModelInfo};

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
        let mut process = self.process.lock().await;
        *process = Some(child);

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

        let mut process_lock = self.process.lock().await;
        if let Some(process) = process_lock.as_mut() {
            if let Some(stdin) = process.stdin.as_mut() {
                let msg = serde_json::to_string(&request)?;
                stdin.write_all(msg.as_bytes()).await?;
                stdin.write_all(b"\n").await?;
                stdin.flush().await?;
            }

            // Read response
            if let Some(stdout) = process.stdout.take() {
                let mut reader = BufReader::new(stdout);
                let mut line = String::new();
                
                while reader.read_line(&mut line).await? > 0 {
                    if let Ok(msg) = serde_json::from_str::<JsonRpcMessage>(&line) {
                        if let JsonRpcMessage::Response(response) = msg {
                            if response.id == id {
                                process.stdout = Some(reader.into_inner());
                                if let Some(error) = response.error {
                                    return Err(anyhow!("MCP error: {} - {}", error.code, error.message));
                                }
                                return response.result.ok_or_else(|| anyhow!("Empty response"));
                            }
                        }
                    }
                    line.clear();
                }
                process.stdout = Some(reader.into_inner());
            }
        }

        Err(anyhow!("Failed to get response from MCP server"))
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