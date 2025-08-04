use serde::{Deserialize, Serialize};
use crate::providers::{Function, FunctionCall};

pub mod server;
pub mod manager;

/// MCP Tool definition as received from an MCP server
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MCPTool {
    pub name: String,
    pub description: String,
    #[serde(alias = "inputSchema")]
    pub input_schema: serde_json::Value,
}

/// MCP Tool call request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MCPToolCall {
    pub name: String,
    pub arguments: serde_json::Value,
}

/// MCP Tool call result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MCPToolResult {
    pub content: Vec<MCPContent>,
    pub is_error: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum MCPContent {
    #[serde(rename = "text")]
    Text { text: String },
    #[serde(rename = "image")]
    Image { data: String, mime_type: String },
    #[serde(rename = "resource")]
    Resource { resource: MCPResource },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MCPResource {
    pub uri: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub mime_type: Option<String>,
}

/// Convert MCP Tool to OpenAI/Anthropic Function format
impl From<MCPTool> for Function {
    fn from(tool: MCPTool) -> Self {
        Function {
            name: tool.name,
            description: tool.description,
            parameters: tool.input_schema,
        }
    }
}

/// Convert AI Function Call to MCP Tool Call
impl From<FunctionCall> for MCPToolCall {
    fn from(func_call: FunctionCall) -> Self {
        let arguments = serde_json::from_str(&func_call.arguments)
            .unwrap_or_else(|_| serde_json::json!({}));
        
        MCPToolCall {
            name: func_call.name,
            arguments,
        }
    }
}