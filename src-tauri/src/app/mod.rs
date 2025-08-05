use anyhow::Result;
use crossterm::event::{self, Event, KeyCode, KeyEvent};
use std::time::Duration;
use tokio::sync::mpsc;
use std::sync::Arc;

use crate::ui::{UILayer, LayerType, simple::SimpleLayer, terminal::TerminalLayer};
use crate::intelligence::UserContext;
use crate::providers::LLMProvider;
use crate::config::AgentXConfig;

#[derive(Debug, Clone)]
pub enum UIMessage {
    CommandStarted(String),
    StreamingChunk(String),
    StreamingComplete,
    Result(String),
    Error(String),
}

pub struct Application {
    current_layer: LayerType,
    user_context: UserContext,
    simple_layer: SimpleLayer,
    terminal_layer: TerminalLayer,
    should_quit: bool,
    _request_tx: mpsc::Sender<String>,
    request_rx: mpsc::Receiver<String>,
    ui_tx: mpsc::Sender<UIMessage>,
    ui_rx: mpsc::Receiver<UIMessage>,
    current_command: Option<String>,
}

impl Application {
    pub fn new() -> Result<Self> {
        // Create channels for async communication
        let (request_tx, request_rx) = mpsc::channel(100);
        let (ui_tx, ui_rx) = mpsc::channel(100);
        
        // Create layers and give them the sender
        let mut simple_layer = crate::ui::create_simple_layer();
        simple_layer.set_channel(request_tx.clone());
        
        let mut terminal_layer = crate::ui::create_terminal_layer();
        terminal_layer.set_channel(request_tx.clone());
        
        Ok(Self {
            current_layer: LayerType::Mission, // Start with terminal UI
            user_context: UserContext::new(),
            simple_layer,
            terminal_layer,
            should_quit: false,
            _request_tx: request_tx,
            request_rx,
            ui_tx,
            ui_rx,
            current_command: None,
        })
    }

    pub async fn run(&mut self) -> Result<()> {
        println!("🚀 AgentX starting up...");
        
        // Check if we can use interactive terminal
        if !self.check_terminal_capability() {
            println!("⚠️  Running in non-interactive mode (terminal features limited)");
            println!("   For full terminal UI, run in a regular terminal with TTY support");
            return self.run_fallback_mode().await;
        }
        
        // Initialize terminal
        self.setup_terminal()?;
        
        // Start background task processor with direct LLM chat
        let ui_tx = self.ui_tx.clone();
        let mut request_rx = std::mem::replace(&mut self.request_rx, mpsc::channel(1).1);
        let _app_tx = self.ui_tx.clone(); // For updating current command
        
        tokio::spawn(async move {
            // Load configuration
            let mut config = AgentXConfig::load().unwrap_or_default();
            
            // Check environment variables and update config
            if let Ok(key) = std::env::var("OPENAI_API_KEY") {
                if let Some(provider_config) = config.providers.get_mut("openai") {
                    provider_config.api_key = Some(key);
                }
            }
            if let Ok(key) = std::env::var("ANTHROPIC_API_KEY") {
                if let Some(provider_config) = config.providers.get_mut("anthropic") {
                    provider_config.api_key = Some(key);
                }
            }
            
            // Try to create a provider with fallback chain
            let provider: Arc<dyn crate::providers::LLMProvider + Send + Sync> = {
                // Try OpenAI first if API key is available
                if let Some(openai_config) = config.providers.get("openai") {
                    if openai_config.api_key.is_some() {
                        let provider_config = crate::providers::ProviderConfig {
                            api_key: openai_config.api_key.clone(),
                            base_url: Some(openai_config.base_url.clone()),
                            timeout_secs: Some(30),
                            max_retries: Some(3),
                        };
                        if let Ok(p) = crate::providers::openai::OpenAIProvider::new(provider_config) {
                            println!("🔌 Using OpenAI provider");
                            config.default_provider = "openai".to_string();
                            config.default_model = "gpt-3.5-turbo".to_string();
                            Arc::new(p) as Arc<dyn crate::providers::LLMProvider + Send + Sync>
                        } else {
                            println!("⚠️  OpenAI provider failed to initialize");
                            create_fallback_provider(&config)
                        }
                    } else {
                        create_fallback_provider(&config)
                    }
                } else {
                    create_fallback_provider(&config)
                }
            };
            
            fn create_fallback_provider(config: &AgentXConfig) -> Arc<dyn crate::providers::LLMProvider + Send + Sync> {
                // Try Anthropic if available
                if let Some(anthropic_config) = config.providers.get("anthropic") {
                    if anthropic_config.api_key.is_some() {
                        let provider_config = crate::providers::ProviderConfig {
                            api_key: anthropic_config.api_key.clone(),
                            base_url: Some(anthropic_config.base_url.clone()),
                            timeout_secs: Some(30),
                            max_retries: Some(3),
                        };
                        if let Ok(p) = crate::providers::anthropic::AnthropicProvider::new(provider_config) {
                            println!("🔌 Using Anthropic provider");
                            return Arc::new(p) as Arc<dyn crate::providers::LLMProvider + Send + Sync>;
                        }
                    }
                }
                
                // Try local provider (Ollama, LM Studio, etc.)
                if let Some(local_config) = config.providers.get("local").or_else(|| config.providers.get("ollama")) {
                    let prov_config = crate::providers::ProviderConfig {
                        api_key: None,
                        base_url: Some(local_config.base_url.clone()),
                        timeout_secs: Some(300),
                        max_retries: Some(3),
                    };
                    if let Ok(p) = crate::providers::openai_compatible::OpenAICompatibleProvider::new(prov_config) {
                        // Check if provider is actually running
                        if let Ok(()) = futures::executor::block_on(p.validate_config()) {
                            println!("🔌 Using local OpenAI-compatible provider");
                            return Arc::new(p) as Arc<dyn crate::providers::LLMProvider + Send + Sync>;
                        }
                    }
                }
                
                // Fallback to demo provider
                println!("🎮 Using Demo provider (no API keys found)");
                println!("   💡 Set OPENAI_API_KEY or ANTHROPIC_API_KEY for full AI capabilities");
                Arc::new(crate::providers::demo::DemoProvider::new()) as Arc<dyn crate::providers::LLMProvider + Send + Sync>
            }
            
            while let Some(request) = request_rx.recv().await {
                // Notify that command started
                let _ = ui_tx.send(UIMessage::CommandStarted(request.clone())).await;
                
                // Direct LLM chat mode
                let messages = vec![
                    crate::providers::Message {
                        role: crate::providers::MessageRole::System,
                        content: "You are AgentX, a helpful AI assistant for software development. Be concise and helpful.".to_string(),
                        function_call: None,
                    },
                    crate::providers::Message {
                        role: crate::providers::MessageRole::User,
                        content: request.clone(),
                        function_call: None,
                    },
                ];
                
                let completion_request = crate::providers::CompletionRequest {
                    messages,
                    model: config.default_model.clone(),
                    temperature: Some(0.7),
                    max_tokens: Some(1000),
                    stream: config.ui.auto_stream,
                    functions: None,
                    tool_choice: None,
                };
                
                // Try streaming first
                match provider.stream_complete(completion_request).await {
                    Ok(mut stream) => {
                        use futures::StreamExt;
                        while let Some(chunk_result) = stream.next().await {
                            match chunk_result {
                                Ok(chunk) => {
                                    if !chunk.delta.is_empty() {
                                        let _ = ui_tx.send(UIMessage::StreamingChunk(chunk.delta)).await;
                                    }
                                    if chunk.finish_reason.is_some() {
                                        let _ = ui_tx.send(UIMessage::StreamingComplete).await;
                                        break;
                                    }
                                }
                                Err(e) => {
                                    let _ = ui_tx.send(UIMessage::Error(format!("Stream error: {}", e))).await;
                                    break;
                                }
                            }
                        }
                    }
                    Err(e) => {
                        let _ = ui_tx.send(UIMessage::Error(format!("❌ Error: {}", e))).await;
                    }
                }
            }
        });
        
        // Main event loop
        let result = self.main_loop().await;
        
        // Cleanup
        self.restore_terminal()?;
        
        result
    }

    async fn main_loop(&mut self) -> Result<()> {
        loop {
            // Check for UI messages from background processing
            if let Ok(msg) = self.ui_rx.try_recv() {
                match msg {
                    UIMessage::CommandStarted(command) => {
                        self.current_command = Some(command);
                    }
                    UIMessage::StreamingChunk(chunk) => {
                        self.simple_layer.append_streaming_content(&chunk);
                    }
                    UIMessage::StreamingComplete => {
                        self.simple_layer.finish_streaming();
                        self.user_context.record_interaction();
                        
                        // Add to terminal layer if we have a command
                        if let Some(cmd) = self.current_command.take() {
                            self.terminal_layer.add_command_result(cmd, "✅ Streaming completed".to_string(), true);
                        }
                    }
                    UIMessage::Result(result) => {
                        self.simple_layer.update_result(result.clone());
                        self.user_context.record_interaction();
                        
                        // Add to terminal layer if we have a command
                        if let Some(cmd) = self.current_command.take() {
                            let success = !result.contains("❌");
                            self.terminal_layer.add_command_result(cmd, result, success);
                        }
                    }
                    UIMessage::Error(error) => {
                        self.simple_layer.update_result(error.clone());
                        
                        // Add to terminal layer if we have a command
                        if let Some(cmd) = self.current_command.take() {
                            self.terminal_layer.add_command_result(cmd, error, false);
                        }
                    }
                }
            }
            
            // Render current layer
            self.render()?;
            
            // Handle events with timeout to allow checking for results
            if self.handle_events_with_timeout().await? {
                break;
            }
            
            // Check if we should transition layers
            if let Some(new_layer) = self.check_layer_transition() {
                self.transition_to(new_layer);
            }
        }
        
        Ok(())
    }

    fn render(&mut self) -> Result<()> {
        match self.current_layer {
            LayerType::Simple => self.simple_layer.render()?,
            LayerType::Mission => self.terminal_layer.render()?,
            LayerType::Pro => {
                // Pro layer not yet implemented, fall back to terminal
                self.terminal_layer.render()?
            }
        }
        Ok(())
    }

    async fn handle_events_with_timeout(&mut self) -> Result<bool> {
        // Poll for events with small timeout to stay responsive
        if event::poll(Duration::from_millis(50))? {
            if let Event::Key(key) = event::read()? {
                // Global shortcuts
                if self.handle_global_shortcuts(&key) {
                    return Ok(self.should_quit);
                }
                
                // Pass to current layer
                match self.current_layer {
                    LayerType::Simple => {
                        if let Some(transition) = self.simple_layer.handle_key(key)? {
                            self.transition_to(transition);
                        }
                    }
                    LayerType::Mission => {
                        if let Some(transition) = self.terminal_layer.handle_key(key)? {
                            self.transition_to(transition);
                        }
                    }
                    LayerType::Pro => {
                        // Pro layer not yet implemented, use terminal layer
                        if let Some(transition) = self.terminal_layer.handle_key(key)? {
                            self.transition_to(transition);
                        }
                    }
                }
            }
        }
        
        Ok(self.should_quit)
    }

    fn handle_global_shortcuts(&mut self, key: &KeyEvent) -> bool {
        match key.code {
            KeyCode::Char('q') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL) => {
                self.should_quit = true;
                true
            }
            KeyCode::Char('p') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL | crossterm::event::KeyModifiers::SHIFT) => {
                // Toggle pro mode
                self.current_layer = if self.current_layer == LayerType::Pro {
                    LayerType::Simple
                } else {
                    LayerType::Pro
                };
                true
            }
            KeyCode::Char('m') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL | crossterm::event::KeyModifiers::SHIFT) => {
                // Toggle mission control mode
                self.current_layer = if self.current_layer == LayerType::Mission {
                    LayerType::Simple
                } else {
                    LayerType::Mission
                };
                true
            }
            KeyCode::F(1) => {
                // Show help
                let _ = crate::ui::help::HelpDisplay::show_help();
                // Wait for a key press
                let _ = event::read();
                true
            }
            _ => false
        }
    }

    fn check_layer_transition(&self) -> Option<LayerType> {
        // Progressive disclosure logic
        if self.user_context.interaction_count > 5 && self.current_layer == LayerType::Simple {
            return Some(LayerType::Mission);
        }
        None
    }

    fn transition_to(&mut self, new_layer: LayerType) {
        self.current_layer = new_layer;
        self.user_context.record_transition(new_layer);
    }

    fn setup_terminal(&self) -> Result<()> {
        crossterm::terminal::enable_raw_mode()?;
        crossterm::execute!(
            std::io::stdout(),
            crossterm::terminal::EnterAlternateScreen,
            crossterm::event::EnableMouseCapture
        )?;
        Ok(())
    }

    fn restore_terminal(&self) -> Result<()> {
        crossterm::terminal::disable_raw_mode()?;
        crossterm::execute!(
            std::io::stdout(),
            crossterm::terminal::LeaveAlternateScreen,
            crossterm::event::DisableMouseCapture
        )?;
        Ok(())
    }
    
    fn check_terminal_capability(&self) -> bool {
        // Try to enable raw mode briefly to check if terminal is interactive
        match crossterm::terminal::enable_raw_mode() {
            Ok(_) => {
                // Successfully enabled, now disable it
                let _ = crossterm::terminal::disable_raw_mode();
                true
            }
            Err(_) => false,
        }
    }
    
    async fn run_fallback_mode(&mut self) -> Result<()> {
        println!("\n🎯 AgentX Demo Mode");
        println!("════════════════════════════════════════════════════════════");
        println!("✨ This is what AgentX would look like in interactive mode:");
        println!();
        
        // Show a demo conversation
        println!("🤔 You: How do I create a Rust function?");
        println!();
        println!("🤖 AgentX: Here's how to create a basic Rust function:");
        println!();
        println!("```rust");
        println!("fn greet(name: &str) -> String {{");
        println!("    format!(\"Hello, {{}}!\", name)");
        println!("}}");
        println!();
        println!("fn main() {{");
        println!("    let message = greet(\"World\");");
        println!("    println!(\"{{}}\", message);");
        println!("}}");
        println!("```");
        println!();
        println!("Key points:");
        println!("• Functions start with `fn` keyword");
        println!("• Parameters specify type with `name: type`");
        println!("• Return type comes after `->`");
        println!("• Last expression is returned (no semicolon)");
        println!();
        
        // Load configuration status
        let config = AgentXConfig::load().unwrap_or_default();
        let provider_status = if config.providers.contains_key("local") || config.providers.contains_key("ollama") {
            "✅ Local provider configured"
        } else {
            "⚠️  No LLM provider configured"
        };
        
        println!("📊 System Status:");
        println!("• Default model: {}", config.default_model);
        println!("• Provider status: {}", provider_status);
        println!("• Streaming enabled: {}", config.ui.auto_stream);
        println!();
        
        println!("🚀 To use AgentX interactively:");
        println!("1. Run in a regular terminal (not through Claude Code)");
        println!("2. Install Ollama: curl -fsSL https://ollama.ai/install.sh | sh");
        println!("3. Pull a model: ollama pull llama2");
        println!("4. Run: cargo run");
        println!();
        
        println!("💡 In interactive mode, you would see:");
        println!("• Terminal UI: Mission Control with command blocks (Warp-style)");
        println!("• Keyboard shortcuts: Ctrl+Shift+M (Simple), Ctrl+Shift+P (Pro)");  
        println!("• Layer switching: Tab for models, command history, palette");
        println!();
        
        println!("👋 Demo complete! AgentX is ready for interactive use.");
        
        Ok(())
    }
}