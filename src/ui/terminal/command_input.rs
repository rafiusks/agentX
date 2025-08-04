use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
use ratatui::{
    prelude::*,
    widgets::{Block, Borders, BorderType},
};
use anyhow::Result;

/// Sophisticated command input with multi-line editing and AI features
pub struct CommandInput {
    /// Current input buffer
    buffer: Vec<String>,
    
    /// Current line index
    current_line: usize,
    
    /// Cursor position in current line
    cursor_pos: usize,
    
    /// Command history
    history: Vec<String>,
    
    /// Current history index
    history_index: Option<usize>,
    
    /// AI suggestion
    ai_suggestion: Option<String>,
    
    /// Whether we're in natural language mode
    natural_language_mode: bool,
    
    /// Syntax highlighter
    syntax_mode: SyntaxMode,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum SyntaxMode {
    Shell,
    Python,
    JavaScript,
    Natural,
}

impl CommandInput {
    pub fn new() -> Self {
        Self {
            buffer: vec![String::new()],
            current_line: 0,
            cursor_pos: 0,
            history: Vec::new(),
            history_index: None,
            ai_suggestion: None,
            natural_language_mode: false,
            syntax_mode: SyntaxMode::Shell,
        }
    }
    
    pub fn set_text(&mut self, text: &str) {
        self.buffer = text.lines().map(|s| s.to_string()).collect();
        if self.buffer.is_empty() {
            self.buffer.push(String::new());
        }
        self.current_line = self.buffer.len() - 1;
        self.cursor_pos = self.buffer[self.current_line].len();
    }
    
    pub fn get_text(&self) -> String {
        self.buffer.join("\n")
    }
    
    pub fn clear(&mut self) {
        self.buffer = vec![String::new()];
        self.current_line = 0;
        self.cursor_pos = 0;
        self.ai_suggestion = None;
    }
    
    pub fn toggle_natural_language(&mut self) {
        self.natural_language_mode = !self.natural_language_mode;
        self.syntax_mode = if self.natural_language_mode {
            SyntaxMode::Natural
        } else {
            SyntaxMode::Shell
        };
    }
    
    pub fn set_ai_suggestion(&mut self, suggestion: String) {
        self.ai_suggestion = Some(suggestion);
    }
    
    pub fn accept_suggestion(&mut self) {
        if let Some(suggestion) = self.ai_suggestion.take() {
            self.set_text(&suggestion);
        }
    }
    
    pub fn handle_key(&mut self, key: KeyEvent) -> Result<Option<String>> {
        match (key.modifiers, key.code) {
            // Submit command
            (KeyModifiers::NONE, KeyCode::Enter) if !key.modifiers.contains(KeyModifiers::SHIFT) => {
                let command = self.get_text();
                if !command.trim().is_empty() {
                    self.history.push(command.clone());
                    self.clear();
                    return Ok(Some(command));
                }
            }
            
            // Multi-line: Shift+Enter
            (KeyModifiers::SHIFT, KeyCode::Enter) => {
                // Insert new line at cursor
                let current = &mut self.buffer[self.current_line];
                let rest = current.split_off(self.cursor_pos);
                self.buffer.insert(self.current_line + 1, rest);
                self.current_line += 1;
                self.cursor_pos = 0;
            }
            
            // Accept AI suggestion: Tab
            (KeyModifiers::NONE, KeyCode::Tab) if self.ai_suggestion.is_some() => {
                self.accept_suggestion();
            }
            
            // Toggle natural language: Ctrl+N
            (KeyModifiers::CONTROL, KeyCode::Char('n')) => {
                self.toggle_natural_language();
            }
            
            // History navigation
            (KeyModifiers::NONE, KeyCode::Up) => {
                if let Some(idx) = self.history_index {
                    if idx > 0 {
                        self.history_index = Some(idx - 1);
                        let text = self.history[idx - 1].clone();
                        self.set_text(&text);
                    }
                } else if !self.history.is_empty() {
                    self.history_index = Some(self.history.len() - 1);
                    let text = self.history[self.history.len() - 1].clone();
                    self.set_text(&text);
                }
            }
            
            (KeyModifiers::NONE, KeyCode::Down) => {
                if let Some(idx) = self.history_index {
                    if idx < self.history.len() - 1 {
                        self.history_index = Some(idx + 1);
                        let text = self.history[idx + 1].clone();
                        self.set_text(&text);
                    } else {
                        self.history_index = None;
                        self.clear();
                    }
                }
            }
            
            // Text navigation
            (KeyModifiers::NONE, KeyCode::Left) => {
                if self.cursor_pos > 0 {
                    self.cursor_pos -= 1;
                } else if self.current_line > 0 {
                    self.current_line -= 1;
                    self.cursor_pos = self.buffer[self.current_line].len();
                }
            }
            
            (KeyModifiers::NONE, KeyCode::Right) => {
                let current_len = self.buffer[self.current_line].len();
                if self.cursor_pos < current_len {
                    self.cursor_pos += 1;
                } else if self.current_line < self.buffer.len() - 1 {
                    self.current_line += 1;
                    self.cursor_pos = 0;
                }
            }
            
            // Word navigation: Ctrl+Left/Right
            (KeyModifiers::CONTROL, KeyCode::Left) => {
                self.move_word_left();
            }
            
            (KeyModifiers::CONTROL, KeyCode::Right) => {
                self.move_word_right();
            }
            
            // Character input
            (KeyModifiers::NONE | KeyModifiers::SHIFT, KeyCode::Char(c)) => {
                self.buffer[self.current_line].insert(self.cursor_pos, c);
                self.cursor_pos += 1;
                self.history_index = None;
            }
            
            // Backspace
            (KeyModifiers::NONE, KeyCode::Backspace) => {
                if self.cursor_pos > 0 {
                    self.buffer[self.current_line].remove(self.cursor_pos - 1);
                    self.cursor_pos -= 1;
                } else if self.current_line > 0 {
                    // Merge with previous line
                    let current = self.buffer.remove(self.current_line);
                    self.current_line -= 1;
                    self.cursor_pos = self.buffer[self.current_line].len();
                    self.buffer[self.current_line].push_str(&current);
                }
            }
            
            // Delete
            (KeyModifiers::NONE, KeyCode::Delete) => {
                let current_len = self.buffer[self.current_line].len();
                if self.cursor_pos < current_len {
                    self.buffer[self.current_line].remove(self.cursor_pos);
                } else if self.current_line < self.buffer.len() - 1 {
                    // Merge with next line
                    let next = self.buffer.remove(self.current_line + 1);
                    self.buffer[self.current_line].push_str(&next);
                }
            }
            
            _ => {}
        }
        
        Ok(None)
    }
    
    fn move_word_left(&mut self) {
        let line = &self.buffer[self.current_line];
        if self.cursor_pos == 0 {
            return;
        }
        
        // Skip spaces
        while self.cursor_pos > 0 && line.chars().nth(self.cursor_pos - 1) == Some(' ') {
            self.cursor_pos -= 1;
        }
        
        // Skip word
        while self.cursor_pos > 0 && line.chars().nth(self.cursor_pos - 1) != Some(' ') {
            self.cursor_pos -= 1;
        }
    }
    
    fn move_word_right(&mut self) {
        let line = &self.buffer[self.current_line];
        let len = line.len();
        
        if self.cursor_pos >= len {
            return;
        }
        
        // Skip word
        while self.cursor_pos < len && line.chars().nth(self.cursor_pos) != Some(' ') {
            self.cursor_pos += 1;
        }
        
        // Skip spaces
        while self.cursor_pos < len && line.chars().nth(self.cursor_pos) == Some(' ') {
            self.cursor_pos += 1;
        }
    }
    
    pub fn render(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Input container with rounded corners
        let block = Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(if self.natural_language_mode {
                Style::default().fg(theme.syntax_function)
            } else {
                theme.block_border_style()
            })
            .style(theme.input_style());
        
        let inner = block.inner(area);
        block.render(area, buf);
        
        // Mode indicator
        let mode_text = if self.natural_language_mode {
            " AI "
        } else {
            match self.syntax_mode {
                SyntaxMode::Shell => " $ ",
                SyntaxMode::Python => " >>> ",
                SyntaxMode::JavaScript => " > ",
                SyntaxMode::Natural => " AI ",
            }
        };
        
        buf.set_string(
            inner.x,
            inner.y,
            mode_text,
            if self.natural_language_mode {
                theme.ai_badge_style()
            } else {
                Style::default().fg(theme.fg_accent)
            },
        );
        
        // Render input text with syntax highlighting
        let text_start_x = inner.x + mode_text.len() as u16;
        let _text_width = inner.width.saturating_sub(mode_text.len() as u16);
        
        // Calculate visible lines and cursor position
        let visible_lines = (inner.height as usize).min(self.buffer.len());
        let start_line = if self.current_line >= visible_lines {
            self.current_line - visible_lines + 1
        } else {
            0
        };
        
        // Render each visible line
        for (i, line_idx) in (start_line..start_line + visible_lines).enumerate() {
            if line_idx >= self.buffer.len() {
                break;
            }
            
            let line = &self.buffer[line_idx];
            let y = inner.y + i as u16;
            
            // Apply syntax highlighting
            let highlighted = self.highlight_line(line);
            buf.set_string(text_start_x, y, highlighted, Style::default());
            
            // Render cursor on current line
            if line_idx == self.current_line {
                let cursor_x = text_start_x + self.cursor_pos as u16;
                if cursor_x < inner.x + inner.width {
                    buf.set_string(
                        cursor_x,
                        y,
                        "█",
                        theme.input_cursor_style(),
                    );
                }
            }
        }
        
        // Render AI suggestion if available
        if let Some(suggestion) = &self.ai_suggestion {
            let suggestion_y = inner.y + inner.height.saturating_sub(1);
            let suggestion_text = format!("→ {}", suggestion);
            buf.set_string(
                inner.x,
                suggestion_y,
                &suggestion_text,
                theme.suggestion_style(),
            );
        }
    }
    
    fn highlight_line(&self, line: &str) -> String {
        // In a real implementation, this would use proper syntax highlighting
        // For now, we'll just return the line as-is
        line.to_string()
    }
}