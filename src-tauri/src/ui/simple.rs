use anyhow::Result;
use crossterm::{
    event::{KeyCode, KeyEvent},
    execute,
    style::{Color, ResetColor, SetForegroundColor},
    terminal::{Clear, ClearType},
    cursor,
};
use std::io::{stdout, Write};
use tokio::sync::mpsc;

use super::{LayerType, UILayer};
use crate::config::{AgentXConfig, ModelConfig, ProviderHealth, ProviderStatus};
use std::collections::HashMap;

pub enum UIMode {
    Input,
    ModelSelection,
}

pub struct SimpleLayer {
    input: String,
    cursor_position: usize,
    is_processing: bool,
    status_message: Option<String>,
    result: Option<String>,
    streaming_content: String,
    tx: Option<mpsc::Sender<String>>,
    
    // Model selection
    mode: UIMode,
    config: AgentXConfig,
    available_models: Vec<(String, ModelConfig)>,
    selected_model_index: usize,
    show_model_selector: bool,
    provider_health: HashMap<String, ProviderHealth>,
}

impl SimpleLayer {
    pub fn new() -> Self {
        let config = AgentXConfig::load().unwrap_or_default();
        let available_models = config.list_available_models();
        
        Self {
            input: String::new(),
            cursor_position: 0,
            is_processing: false,
            status_message: None,
            result: None,
            streaming_content: String::new(),
            tx: None,
            
            mode: UIMode::Input,
            show_model_selector: config.ui.show_model_selector,
            selected_model_index: 0,
            available_models,
            provider_health: HashMap::new(),
            config,
        }
    }
    
    pub fn set_channel(&mut self, tx: mpsc::Sender<String>) {
        self.tx = Some(tx);
    }
    
    pub fn update_result(&mut self, result: String) {
        self.result = Some(result);
        self.is_processing = false;
        self.status_message = Some("✅ Complete! Press Enter to continue.".to_string());
    }
    
    pub fn append_streaming_content(&mut self, chunk: &str) {
        self.streaming_content.push_str(chunk);
    }
    
    pub fn finish_streaming(&mut self) {
        if !self.streaming_content.is_empty() {
            self.result = Some(self.streaming_content.clone());
            self.streaming_content.clear();
            self.is_processing = false;
            self.status_message = Some("✅ Complete! Press Enter to continue.".to_string());
        }
    }
    
    pub fn get_current_model(&self) -> (String, String) {
        (self.config.default_provider.clone(), self.config.default_model.clone())
    }
    
    pub fn get_current_model_display(&self) -> String {
        if let Some(model) = self.config.get_default_model() {
            format!("{}/{}", self.config.default_provider, model.display_name)
        } else {
            format!("{}/{}", self.config.default_provider, self.config.default_model)
        }
    }
    
    fn toggle_model_selector(&mut self) {
        match self.mode {
            UIMode::Input => {
                if self.show_model_selector && !self.available_models.is_empty() {
                    self.mode = UIMode::ModelSelection;
                }
            }
            UIMode::ModelSelection => {
                self.mode = UIMode::Input;
            }
        }
    }
    
    fn select_model(&mut self) -> Result<()> {
        if let Some((model_key, _)) = self.available_models.get(self.selected_model_index) {
            let parts: Vec<&str> = model_key.split('/').collect();
            if parts.len() == 2 {
                let _ = self.config.set_default_model(parts[0].to_string(), parts[1].to_string());
                self.status_message = Some(format!("✅ Switched to {}", model_key));
                self.mode = UIMode::Input;
            }
        }
        Ok(())
    }
    
    pub async fn refresh_provider_health(&mut self) {
        self.provider_health = self.config.check_all_providers_health().await;
    }
    
    fn get_provider_status(&self, provider_name: &str) -> Option<&ProviderStatus> {
        self.provider_health.get(provider_name).map(|h| &h.status)
    }

    fn draw_centered_box(&self, width: u16, height: u16) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        let start_x = (term_size.0 - width) / 2;
        let start_y = (term_size.1 - height) / 2;

        // Clear screen
        execute!(stdout(), Clear(ClearType::All))?;

        // If we have a result, show it instead
        if let Some(result) = &self.result {
            self.draw_result_view(result)?;
            return Ok(());
        }
        
        // If we're streaming, show streaming view
        if self.is_processing && !self.streaming_content.is_empty() {
            self.draw_streaming_view()?;
            return Ok(());
        }
        
        // If we're in model selection mode, show model selector
        if matches!(self.mode, UIMode::ModelSelection) {
            self.draw_model_selector()?;
            return Ok(());
        }

        // Draw box
        execute!(stdout(), cursor::MoveTo(start_x, start_y))?;
        println!("┌{}┐", "─".repeat((width - 2) as usize));

        // Empty lines
        for i in 1..height-1 {
            execute!(stdout(), cursor::MoveTo(start_x, start_y + i))?;
            println!("│{}│", " ".repeat((width - 2) as usize));
        }

        // Bottom border
        execute!(stdout(), cursor::MoveTo(start_x, start_y + height - 1))?;
        println!("└{}┘", "─".repeat((width - 2) as usize));

        // Draw content
        let content_y = start_y + (height / 2) - 2;
        
        // Title
        let title = "What would you like to create?";
        let title_x = start_x + (width - title.len() as u16) / 2;
        execute!(stdout(), cursor::MoveTo(title_x, content_y))?;
        execute!(stdout(), SetForegroundColor(Color::White))?;
        print!("{}", title);
        
        // Model info (if model selector is enabled)
        if self.show_model_selector {
            let model_info = format!("Model: {} (Press Tab to change)", self.get_current_model_display());
            let model_x = start_x + (width - model_info.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(model_x, content_y - 1))?;
            execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
            print!("{}", model_info);
            execute!(stdout(), ResetColor)?;
        }

        // Input field
        let input_y = content_y + 2;
        let input_x = start_x + 4;
        let input_width = (width - 12) as usize;
        
        execute!(stdout(), cursor::MoveTo(input_x, input_y))?;
        print!("[");
        
        // Show input or status
        if self.is_processing {
            execute!(stdout(), SetForegroundColor(Color::Yellow))?;
            print!("{:^width$}", "✨ Working on it...", width = input_width);
        } else {
            execute!(stdout(), SetForegroundColor(Color::White))?;
            print!("{:<width$}", self.input, width = input_width);
        }
        
        execute!(stdout(), ResetColor)?;
        print!("] ");
        
        // Microphone emoji
        execute!(stdout(), SetForegroundColor(Color::Cyan))?;
        print!("🎤");
        execute!(stdout(), ResetColor)?;

        // Quick ideas (if not processing)
        if !self.is_processing && self.input.is_empty() {
            let ideas_y = input_y + 2;
            let ideas = "Quick ideas:";
            let ideas_x = start_x + (width - ideas.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(ideas_x, ideas_y))?;
            execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
            print!("{}", ideas);

            let examples = "• Build a web app  • Analyze data  • Write docs";
            let examples_x = start_x + (width - examples.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(examples_x, ideas_y + 1))?;
            print!("{}", examples);
            execute!(stdout(), ResetColor)?;
        }

        // Status message
        if let Some(status) = &self.status_message {
            let status_y = start_y + height + 1;
            let status_x = start_x + (width - status.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(status_x, status_y))?;
            execute!(stdout(), SetForegroundColor(Color::Green))?;
            print!("{}", status);
            execute!(stdout(), ResetColor)?;
        }
        
        // Shortcuts help
        let term_size = crossterm::terminal::size()?;
        let help_text = "Ctrl+Shift+M: Terminal Mode | Tab: Models | Ctrl+Q: Quit";
        let help_x = (term_size.0 - help_text.len() as u16) / 2;
        execute!(stdout(), cursor::MoveTo(help_x, term_size.1 - 1))?;
        execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
        print!("{}", help_text);
        execute!(stdout(), ResetColor)?;

        // Position cursor
        if !self.is_processing {
            let cursor_x = input_x + 1 + self.cursor_position as u16;
            execute!(stdout(), cursor::MoveTo(cursor_x, input_y))?;
            execute!(stdout(), cursor::Show)?;
        } else {
            execute!(stdout(), cursor::Hide)?;
        }

        stdout().flush()?;
        Ok(())
    }

    fn draw_result_view(&self, result: &str) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        
        // Clear and show result
        execute!(stdout(), Clear(ClearType::All))?;
        execute!(stdout(), cursor::MoveTo(2, 2))?;
        
        // Split result into lines and display
        for (i, line) in result.lines().enumerate() {
            execute!(stdout(), cursor::MoveTo(2, 2 + i as u16))?;
            print!("{}", line);
        }
        
        // Show continue prompt at bottom
        let prompt = "Press Enter to create something else, or Ctrl+Q to quit";
        execute!(stdout(), cursor::MoveTo(2, term_size.1 - 2))?;
        execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
        print!("{}", prompt);
        execute!(stdout(), ResetColor)?;
        
        execute!(stdout(), cursor::Hide)?;
        stdout().flush()?;
        Ok(())
    }
    
    fn draw_streaming_view(&self) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        
        // Clear and show header
        execute!(stdout(), Clear(ClearType::All))?;
        execute!(stdout(), cursor::MoveTo(2, 2))?;
        execute!(stdout(), SetForegroundColor(Color::Cyan))?;
        print!("🤖 AgentX is thinking...");
        execute!(stdout(), ResetColor)?;
        
        // Show streaming content
        execute!(stdout(), cursor::MoveTo(2, 4))?;
        
        // Split streaming content into lines and display
        let max_lines = (term_size.1 - 6) as usize;
        let lines: Vec<&str> = self.streaming_content.lines().collect();
        let start_line = if lines.len() > max_lines {
            lines.len() - max_lines
        } else {
            0
        };
        
        for (i, line) in lines.iter().skip(start_line).enumerate() {
            execute!(stdout(), cursor::MoveTo(2, 4 + i as u16))?;
            print!("{}", line);
        }
        
        // Show cursor animation
        execute!(stdout(), cursor::MoveTo(2 + self.streaming_content.len() as u16 % (term_size.0 - 4), 4 + lines.len() as u16))?;
        execute!(stdout(), SetForegroundColor(Color::Yellow))?;
        print!("▊");
        execute!(stdout(), ResetColor)?;
        
        execute!(stdout(), cursor::Hide)?;
        stdout().flush()?;
        Ok(())
    }
    
    fn draw_model_selector(&self) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        let width = 70u16;
        let height = (self.available_models.len() + 6).min(term_size.1 as usize - 4) as u16;
        let start_x = (term_size.0 - width) / 2;
        let start_y = (term_size.1 - height) / 2;
        
        // Clear and draw box
        execute!(stdout(), Clear(ClearType::All))?;
        execute!(stdout(), cursor::MoveTo(start_x, start_y))?;
        println!("┌{}┐", "─".repeat((width - 2) as usize));
        
        for i in 1..height-1 {
            execute!(stdout(), cursor::MoveTo(start_x, start_y + i))?;
            println!("│{}│", " ".repeat((width - 2) as usize));
        }
        
        execute!(stdout(), cursor::MoveTo(start_x, start_y + height - 1))?;
        println!("└{}┘", "─".repeat((width - 2) as usize));
        
        // Title
        let title = "Select Model";
        let title_x = start_x + (width - title.len() as u16) / 2;
        execute!(stdout(), cursor::MoveTo(title_x, start_y + 1))?;
        execute!(stdout(), SetForegroundColor(Color::White))?;
        print!("{}", title);
        execute!(stdout(), ResetColor)?;
        
        // Instructions
        let instructions = "↑↓ Navigate  Enter Select  R Refresh  Esc Cancel";
        let instr_x = start_x + (width - instructions.len() as u16) / 2;
        execute!(stdout(), cursor::MoveTo(instr_x, start_y + 2))?;
        execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
        print!("{}", instructions);
        execute!(stdout(), ResetColor)?;
        
        // Model list
        let list_start_y = start_y + 4;
        let visible_count = (height - 6) as usize;
        let start_index = if self.selected_model_index >= visible_count {
            self.selected_model_index - visible_count + 1
        } else {
            0
        };
        
        for (i, (model_key, model)) in self.available_models.iter().enumerate().skip(start_index).take(visible_count) {
            let y = list_start_y + (i - start_index) as u16;
            execute!(stdout(), cursor::MoveTo(start_x + 2, y))?;
            
            if i == self.selected_model_index {
                execute!(stdout(), SetForegroundColor(Color::Black))?;
                execute!(stdout(), crossterm::style::SetBackgroundColor(crossterm::style::Color::White))?;
                print!("→ ");
            } else {
                print!("  ");
            }
            
            // Get provider status
            let provider_name = model_key.split('/').next().unwrap_or("unknown");
            let status_icon = if let Some(status) = self.get_provider_status(provider_name) {
                match status {
                    ProviderStatus::Online => "🟢",
                    ProviderStatus::Offline => "🔴",
                    ProviderStatus::ConfigError(_) => "🟡",
                    ProviderStatus::Unknown => "⚪",
                }
            } else {
                "❓"
            };
            
            // Model name and description with status
            let display_text = if let Some(desc) = &model.description {
                format!("{} {} - {}", status_icon, model.display_name, desc)
            } else {
                format!("{} {}", status_icon, model.display_name)
            };
            
            let max_width = (width - 6) as usize;
            let truncated = if display_text.len() > max_width {
                format!("{}...", &display_text[..max_width-3])
            } else {
                display_text
            };
            
            print!("{:<width$}", truncated, width = max_width);
            execute!(stdout(), ResetColor)?;
        }
        
        // Current selection info
        if let Some((_, model)) = self.available_models.get(self.selected_model_index) {
            let info_y = start_y + height - 2;
            execute!(stdout(), cursor::MoveTo(start_x + 2, info_y))?;
            execute!(stdout(), SetForegroundColor(Color::Cyan))?;
            let info = format!("Context: {}  Streaming: {}", 
                model.context_size, 
                if model.supports_streaming { "Yes" } else { "No" }
            );
            print!("{}", info);
            execute!(stdout(), ResetColor)?;
        }
        
        execute!(stdout(), cursor::Hide)?;
        stdout().flush()?;
        Ok(())
    }
}

impl UILayer for SimpleLayer {
    fn layer_type(&self) -> LayerType {
        LayerType::Simple
    }

    fn render(&mut self) -> Result<()> {
        self.draw_centered_box(60, 10)
    }

    fn handle_key(&mut self, key: KeyEvent) -> Result<Option<LayerType>> {
        // If showing result, Enter clears it
        if self.result.is_some() && key.code == KeyCode::Enter {
            self.result = None;
            self.status_message = None;
            return Ok(None);
        }

        if self.is_processing {
            return Ok(None);
        }

        // Handle model selection mode
        if matches!(self.mode, UIMode::ModelSelection) {
            match key.code {
                KeyCode::Up => {
                    if self.selected_model_index > 0 {
                        self.selected_model_index -= 1;
                    }
                }
                KeyCode::Down => {
                    if self.selected_model_index < self.available_models.len().saturating_sub(1) {
                        self.selected_model_index += 1;
                    }
                }
                KeyCode::Enter => {
                    self.select_model()?;
                }
                KeyCode::Esc | KeyCode::Tab => {
                    self.mode = UIMode::Input;
                }
                _ => {}
            }
            return Ok(None);
        }

        // Handle input mode
        match key.code {
            KeyCode::Char(c) => {
                self.input.insert(self.cursor_position, c);
                self.cursor_position += 1;
            }
            KeyCode::Backspace => {
                if self.cursor_position > 0 {
                    self.cursor_position -= 1;
                    self.input.remove(self.cursor_position);
                }
            }
            KeyCode::Left => {
                if self.cursor_position > 0 {
                    self.cursor_position -= 1;
                }
            }
            KeyCode::Right => {
                if self.cursor_position < self.input.len() {
                    self.cursor_position += 1;
                }
            }
            KeyCode::Enter => {
                if !self.input.is_empty() {
                    self.is_processing = true;
                    self.streaming_content.clear(); // Clear any previous streaming content
                    
                    // Send request to process
                    if let Some(tx) = &self.tx {
                        let _ = tx.try_send(self.input.clone());
                        self.input.clear();
                        self.cursor_position = 0;
                    }
                }
            }
            KeyCode::Tab => {
                self.toggle_model_selector();
            }
            _ => {}
        }

        Ok(None)
    }
}