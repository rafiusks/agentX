// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use crate::{
    providers::{*, demo::DemoProvider, openai::OpenAIProvider, anthropic::AnthropicProvider, openai_compatible::OpenAICompatibleProvider},
    config::AgentXConfig,
    mcp::{manager::MCPManager, server::MCPServerConfig}
};
use std::sync::Arc;
use tauri::{State, Emitter};
use tokio::sync::Mutex;
use futures::StreamExt;

// State to hold our provider and MCP manager
struct AppState {
    provider: Arc<Mutex<Option<Arc<dyn LLMProvider + Send + Sync>>>>,
    config: Arc<Mutex<AgentXConfig>>,
    mcp_manager: Arc<MCPManager>,
}

#[tauri::command]
async fn get_providers(state: State<'_, AppState>) -> Result<Vec<ProviderInfo>, String> {
    let config = state.config.lock().await;
    
    let providers = vec![
        ProviderInfo {
            id: "openai".to_string(),
            name: "OpenAI".to_string(),
            enabled: config.providers.get("openai")
                .and_then(|p| p.api_key.as_ref())
                .is_some(),
            status: "Ready".to_string(),
        },
        ProviderInfo {
            id: "anthropic".to_string(),
            name: "Anthropic".to_string(),
            enabled: config.providers.get("anthropic")
                .and_then(|p| p.api_key.as_ref())
                .is_some(),
            status: "Ready".to_string(),
        },
        ProviderInfo {
            id: "ollama".to_string(),
            name: "Ollama".to_string(),
            enabled: true,
            status: "Ready".to_string(),
        },
        ProviderInfo {
            id: "demo".to_string(),
            name: "Demo".to_string(),
            enabled: true,
            status: "Always Available".to_string(),
        },
    ];
    
    Ok(providers)
}

#[tauri::command]
async fn send_message(
    state: State<'_, AppState>,
    message: String,
    provider_id: Option<String>,
) -> Result<String, String> {
    // Create provider based on provider_id
    let config = state.config.lock().await;
    let provider = create_provider(&config, provider_id).await
        .map_err(|e| format!("Failed to create provider: {}", e))?;
    
    // Create completion request
    let messages = vec![
        Message {
            role: MessageRole::System,
            content: "You are AgentX, a helpful AI assistant for software development. Be concise and helpful.".to_string(),
            function_call: None,
        },
        Message {
            role: MessageRole::User,
            content: message,
            function_call: None,
        },
    ];
    
    let request = CompletionRequest {
        messages,
        model: "gpt-3.5-turbo".to_string(), // Will be overridden by provider
        temperature: Some(0.7),
        max_tokens: Some(1000),
        stream: false,
        functions: None,
        tool_choice: None,
    };
    
    // Get completion
    let response = provider.complete(request).await
        .map_err(|e| format!("Completion error: {}", e))?;
    
    Ok(response.content)
}

#[tauri::command]
async fn stream_message(
    state: State<'_, AppState>,
    message: String,
    provider_id: Option<String>,
    window: tauri::Window,
) -> Result<(), String> {
    // Create provider based on provider_id
    let config = state.config.lock().await;
    let provider = create_provider(&config, provider_id.clone()).await
        .map_err(|e| format!("Failed to create provider: {}", e))?;
    
    // Create completion request
    let messages = vec![
        Message {
            role: MessageRole::System,
            content: "You are AgentX, a helpful AI assistant for software development. Be concise and helpful.".to_string(),
            function_call: None,
        },
        Message {
            role: MessageRole::User,
            content: message,
            function_call: None,
        },
    ];
    
    // Get available MCP tools
    let functions = match state.mcp_manager.get_functions().await {
        Ok(funcs) => {
            if !funcs.is_empty() {
                println!("Adding {} MCP functions to completion request", funcs.len());
                Some(funcs)
            } else {
                None
            }
        }
        Err(e) => {
            println!("Failed to get MCP functions: {}", e);
            None
        }
    };

    let has_functions = functions.is_some();
    
    // Use appropriate model based on provider
    let model = match provider_id.as_deref() {
        Some("ollama") => "deepseek/deepseek-r1-0528-qwen3-8b".to_string(), // Available model
        Some("openai") => "gpt-3.5-turbo".to_string(),
        Some("anthropic") => "claude-3-sonnet-20240229".to_string(),
        _ => "gpt-3.5-turbo".to_string(),
    };
    
    let request = CompletionRequest {
        messages,
        model,
        temperature: Some(0.7),
        max_tokens: Some(1000),
        stream: true,
        functions,
        tool_choice: if has_functions { Some(crate::providers::ToolChoice::Auto) } else { None },
    };
    
    // Stream completion
    match provider.stream_complete(request).await {
        Ok(mut stream) => {
            let mut chunk_count = 0;
            let mut function_call_buffer: Option<String> = None;
            let mut function_name: Option<String> = None;
            
            while let Some(chunk_result) = stream.next().await {
                match chunk_result {
                    Ok(chunk) => {
                        // Handle function calls if present
                        if let Some(function_call_delta) = &chunk.function_call_delta {
                            if let Some(name) = &function_call_delta.name {
                                function_name = Some(name.clone());
                            }
                            if let Some(args) = &function_call_delta.arguments {
                                if let Some(buffer) = &mut function_call_buffer {
                                    buffer.push_str(args);
                                } else {
                                    function_call_buffer = Some(args.clone());
                                }
                            }
                        }
                        
                        if !chunk.delta.is_empty() {
                            chunk_count += 1;
                            println!("Emitting chunk {}: {:?}", chunk_count, chunk.delta);
                            window.emit("stream-chunk", &chunk.delta)
                                .map_err(|e| format!("Failed to emit: {}", e))?;
                        }
                        if let Some(reason) = &chunk.finish_reason {
                            println!("Finish reason detected: {}", reason);
                            
                            // If we have a complete function call, execute it
                            if let (Some(name), Some(args)) = (&function_name, &function_call_buffer) {
                                println!("Executing function call: {} with args: {}", name, args);
                                let function_call = crate::providers::FunctionCall {
                                    name: name.clone(),
                                    arguments: args.clone(),
                                };
                                
                                match state.mcp_manager.execute_function(function_call).await {
                                    Ok(result) => {
                                        println!("Function result: {}", result);
                                        window.emit("stream-chunk", &format!("\n\n**Function Result:**\n{}", result))
                                            .map_err(|e| format!("Failed to emit function result: {}", e))?;
                                    }
                                    Err(e) => {
                                        println!("Function execution failed: {}", e);
                                        window.emit("stream-chunk", &format!("\n\n**Function Error:** {}", e))
                                            .map_err(|e| format!("Failed to emit function error: {}", e))?;
                                    }
                                }
                            }
                            break;
                        }
                    }
                    Err(e) => {
                        println!("Stream error: {}", e);
                        window.emit("stream-error", format!("Stream error: {}", e))
                            .map_err(|e| format!("Failed to emit error: {}", e))?;
                        break;
                    }
                }
            }
            // Emit stream-end after the loop completes
            println!("Emitting stream-end event");
            window.emit("stream-end", ())
                .map_err(|e| format!("Failed to emit end: {}", e))?;
        }
        Err(e) => {
            return Err(format!("Stream error: {}", e));
        }
    }
    
    Ok(())
}

#[tauri::command]
async fn update_api_key(
    state: State<'_, AppState>,
    provider_id: String,
    api_key: String,
) -> Result<(), String> {
    let mut config = state.config.lock().await;
    
    if let Some(provider_config) = config.providers.get_mut(&provider_id) {
        provider_config.api_key = Some(api_key);
        config.save()
            .map_err(|e| format!("Failed to save config: {}", e))?;
        
        // Clear current provider to force recreation
        let mut provider_lock = state.provider.lock().await;
        *provider_lock = None;
        
        Ok(())
    } else {
        Err(format!("Provider {} not found", provider_id))
    }
}

async fn create_provider(
    config: &AgentXConfig,
    provider_id: Option<String>,
) -> Result<Arc<dyn LLMProvider + Send + Sync>, String> {
    let provider_id = provider_id.unwrap_or_else(|| config.default_provider.clone());
    
    match provider_id.as_str() {
        "openai" => {
            if let Some(provider_config) = config.providers.get("openai") {
                if let Some(api_key) = &provider_config.api_key {
                    let prov_config = ProviderConfig {
                        api_key: Some(api_key.clone()),
                        base_url: Some(provider_config.base_url.clone()),
                        timeout_secs: Some(30),
                        max_retries: Some(3),
                    };
                    return OpenAIProvider::new(prov_config)
                        .map(|p| Arc::new(p) as Arc<dyn LLMProvider + Send + Sync>)
                        .map_err(|e| e.to_string());
                }
            }
        }
        "anthropic" => {
            if let Some(provider_config) = config.providers.get("anthropic") {
                if let Some(api_key) = &provider_config.api_key {
                    let prov_config = ProviderConfig {
                        api_key: Some(api_key.clone()),
                        base_url: Some(provider_config.base_url.clone()),
                        timeout_secs: Some(30),
                        max_retries: Some(3),
                    };
                    return AnthropicProvider::new(prov_config)
                        .map(|p| Arc::new(p) as Arc<dyn LLMProvider + Send + Sync>)
                        .map_err(|e| e.to_string());
                }
            }
        }
        "ollama" => {
            return OpenAICompatibleProvider::for_ollama()
                .map(|p| Arc::new(p) as Arc<dyn LLMProvider + Send + Sync>)
                .map_err(|e| e.to_string());
        }
        _ => {}
    }
    
    // Fallback to demo provider
    Ok(Arc::new(DemoProvider::new()) as Arc<dyn LLMProvider + Send + Sync>)
}

#[tauri::command]
async fn add_mcp_server(
    state: State<'_, AppState>,
    name: String,
    command: String,
    args: Vec<String>,
) -> Result<(), String> {
    println!("add_mcp_server called with: name={}, command={}, args={:?}", name, command, args);
    
    let config = MCPServerConfig {
        name: name.clone(),
        command,
        args,
        env: None,
        capabilities: vec!["tools".to_string()],
    };

    println!("Created MCP config: {:?}", config);
    
    match state.mcp_manager.add_server(config).await {
        Ok(()) => {
            println!("MCP server '{}' added successfully", name);
            Ok(())
        }
        Err(e) => {
            println!("Failed to add MCP server '{}': {}", name, e);
            Err(format!("Failed to add MCP server: {}", e))
        }
    }
}

#[tauri::command]
async fn list_mcp_tools(state: State<'_, AppState>) -> Result<Vec<String>, String> {
    println!("list_mcp_tools called");
    
    match state.mcp_manager.get_all_tools().await {
        Ok(tools) => {
            println!("Found {} MCP tools total", tools.len());
            for tool in &tools {
                println!("Tool: {} - {}", tool.name, tool.description);
            }
            Ok(tools.into_iter().map(|t| format!("{}: {}", t.name, t.description)).collect())
        }
        Err(e) => {
            println!("Failed to get MCP tools: {}", e);
            Err(format!("Failed to get MCP tools: {}", e))
        }
    }
}

#[tauri::command]
async fn get_mcp_servers(state: State<'_, AppState>) -> Result<Vec<String>, String> {
    let status = state.mcp_manager.get_server_status().await;
    Ok(status.keys().cloned().collect())
}

#[tauri::command]
async fn remove_mcp_server(
    state: State<'_, AppState>,
    name: String,
) -> Result<(), String> {
    println!("remove_mcp_server called with name: {}", name);
    
    match state.mcp_manager.remove_server(&name).await {
        Ok(()) => {
            println!("MCP server '{}' removed successfully", name);
            Ok(())
        }
        Err(e) => {
            println!("Failed to remove MCP server '{}': {}", name, e);
            Err(format!("Failed to remove MCP server: {}", e))
        }
    }
}

#[derive(serde::Serialize)]
struct ProviderInfo {
    id: String,
    name: String,
    enabled: bool,
    status: String,
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let config = AgentXConfig::load().unwrap_or_default();
    
    let app_state = AppState {
        provider: Arc::new(Mutex::new(None)),
        config: Arc::new(Mutex::new(config)),
        mcp_manager: Arc::new(MCPManager::new()),
    };
    
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(app_state)
        .invoke_handler(tauri::generate_handler![
            get_providers,
            send_message,
            stream_message,
            update_api_key,
            add_mcp_server,
            remove_mcp_server,
            list_mcp_tools,
            get_mcp_servers,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}