use anyhow::{Result, anyhow};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

use super::{MCPTool, MCPToolCall, MCPToolResult};
use super::server::{MCPServer, MCPServerConfig};
use crate::providers::{Function, FunctionCall};

/// Manages multiple MCP servers and their tools
pub struct MCPManager {
    servers: Arc<RwLock<HashMap<String, Arc<MCPServer>>>>,
    tool_to_server: Arc<RwLock<HashMap<String, String>>>, // tool_name -> server_name
}

impl MCPManager {
    pub fn new() -> Self {
        Self {
            servers: Arc::new(RwLock::new(HashMap::new())),
            tool_to_server: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Add and connect to an MCP server
    pub async fn add_server(&self, config: MCPServerConfig) -> Result<()> {
        let server_name = config.name.clone();
        println!("Creating MCP server instance for '{}'", server_name);
        
        let server = Arc::new(MCPServer::new(config).await?);
        println!("MCP server instance created, attempting to connect...");
        
        // Connect to the server
        match server.connect().await {
            Ok(()) => println!("MCP server '{}' connected successfully", server_name),
            Err(e) => {
                println!("Failed to connect to MCP server '{}': {}", server_name, e);
                return Err(e);
            }
        }
        
        // Discover tools
        println!("Discovering tools for MCP server '{}'", server_name);
        let tools = match server.list_tools().await {
            Ok(tools) => {
                println!("Found {} tools for server '{}'", tools.len(), server_name);
                tools
            }
            Err(e) => {
                println!("Failed to list tools for server '{}': {}", server_name, e);
                vec![] // Continue even if tool discovery fails
            }
        };
        
        // Register server and map tools to server
        {
            let mut servers = self.servers.write().await;
            servers.insert(server_name.clone(), server);
            println!("Registered MCP server '{}' in manager", server_name);
        }
        
        {
            let mut tool_mapping = self.tool_to_server.write().await;
            for tool in &tools {
                tool_mapping.insert(tool.name.clone(), server_name.clone());
            }
            println!("Mapped {} tools to server '{}'", tools.len(), server_name);
        }
        
        println!("Added MCP server '{}' with {} tools", server_name, tools.len());
        Ok(())
    }

    /// Get all available tools from all connected MCP servers
    pub async fn get_all_tools(&self) -> Result<Vec<MCPTool>> {
        let servers = self.servers.read().await;
        let mut all_tools = Vec::new();
        
        for server in servers.values() {
            if let Ok(tools) = server.list_tools().await {
                all_tools.extend(tools);
            }
        }
        
        Ok(all_tools)
    }

    /// Convert MCP tools to AI functions
    pub async fn get_functions(&self) -> Result<Vec<Function>> {
        let tools = self.get_all_tools().await?;
        Ok(tools.into_iter().map(Function::from).collect())
    }

    /// Execute a function call by routing it to the appropriate MCP server
    pub async fn execute_function(&self, function_call: FunctionCall) -> Result<String> {
        // Find which server handles this tool
        let server_name = {
            let tool_mapping = self.tool_to_server.read().await;
            tool_mapping.get(&function_call.name)
                .cloned()
                .ok_or_else(|| anyhow!("No MCP server found for tool: {}", function_call.name))?
        };

        // Get the server
        let server = {
            let servers = self.servers.read().await;
            servers.get(&server_name)
                .cloned()
                .ok_or_else(|| anyhow!("MCP server '{}' not found", server_name))?
        };

        // Convert function call to MCP tool call
        let tool_call = MCPToolCall::from(function_call);
        
        // Execute the tool
        let result = server.call_tool(tool_call).await?;
        
        // Convert result to string (simplified for now)
        let content_text = result.content
            .into_iter()
            .filter_map(|content| match content {
                super::MCPContent::Text { text } => Some(text),
                _ => None, // For now, only handle text content
            })
            .collect::<Vec<_>>()
            .join("\n");

        if result.is_error.unwrap_or(false) {
            Err(anyhow!("Tool execution failed: {}", content_text))
        } else {
            Ok(content_text)
        }
    }

    /// Remove an MCP server
    pub async fn remove_server(&self, server_name: &str) -> Result<()> {
        // Remove server
        let server = {
            let mut servers = self.servers.write().await;
            servers.remove(server_name)
        };

        if let Some(server) = server {
            // Disconnect
            server.disconnect().await?;
            
            // Remove tool mappings
            let mut tool_mapping = self.tool_to_server.write().await;
            tool_mapping.retain(|_, name| name != server_name);
            
            tracing::info!("Removed MCP server '{}'", server_name);
        }

        Ok(())
    }

    /// Get status of all connected servers
    pub async fn get_server_status(&self) -> HashMap<String, bool> {
        let servers = self.servers.read().await;
        servers.keys()
            .map(|name| (name.clone(), true)) // Simplified - assume connected if in map
            .collect()
    }
}

impl Default for MCPManager {
    fn default() -> Self {
        Self::new()
    }
}