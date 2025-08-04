use chrono::{DateTime, Local};
use ratatui::{
    prelude::*,
    widgets::{Block, Borders, BorderType, Paragraph, Wrap},
};

/// Individual command block that contains command and output
#[derive(Debug, Clone)]
pub struct CommandBlock {
    /// Unique ID for this block
    pub id: uuid::Uuid,
    
    /// The command that was executed
    pub command: String,
    
    /// The output of the command
    pub output: String,
    
    /// Whether the command succeeded
    pub success: bool,
    
    /// When the command was executed
    pub timestamp: DateTime<Local>,
    
    /// Execution duration in milliseconds
    pub duration: Option<u64>,
    
    /// Whether this block is selected
    pub selected: bool,
    
    /// Whether this block is collapsed
    pub collapsed: bool,
    
    /// AI-generated explanation (if any)
    pub ai_explanation: Option<String>,
}

impl CommandBlock {
    pub fn new(command: String, output: String, success: bool) -> Self {
        Self {
            id: uuid::Uuid::new_v4(),
            command,
            output,
            success,
            timestamp: Local::now(),
            duration: None,
            selected: false,
            collapsed: false,
            ai_explanation: None,
        }
    }
    
    pub fn with_duration(mut self, duration_ms: u64) -> Self {
        self.duration = Some(duration_ms);
        self
    }
    
    pub fn with_ai_explanation(mut self, explanation: String) -> Self {
        self.ai_explanation = Some(explanation);
        self
    }
    
    /// Toggle collapsed state
    pub fn toggle_collapse(&mut self) {
        self.collapsed = !self.collapsed;
    }
    
    /// Render this block as a widget
    pub fn render(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Block container with rounded corners
        let block = Block::default()
            .borders(Borders::ALL)
            .border_style(if self.selected {
                theme.selected_style()
            } else {
                theme.block_border_style()
            })
            .border_type(BorderType::Rounded)
            .style(theme.block_style());
        
        let inner = block.inner(area);
        block.render(area, buf);
        
        // Layout: Header, Command, Output
        let chunks = if self.collapsed {
            vec![Rect::new(inner.x, inner.y, inner.width, 2)]
        } else {
            Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Length(2),  // Header
                    Constraint::Length(3),  // Command
                    Constraint::Min(3),     // Output
                ])
                .split(inner).to_vec()
        };
        
        // Header with timestamp and status
        self.render_header(chunks[0], buf, theme);
        
        if !self.collapsed && chunks.len() > 1 {
            // Command section
            self.render_command(chunks[1], buf, theme);
            
            // Output section
            self.render_output(chunks[2], buf, theme);
        }
    }
    
    fn render_header(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        let status_icon = if self.success { "✓" } else { "✗" };
        let status_style = if self.success {
            theme.success_style()
        } else {
            theme.error_style()
        };
        
        // Left side: status + timestamp
        let timestamp = self.timestamp.format("%H:%M:%S").to_string();
        let left_content = format!("{} {}", status_icon, timestamp);
        
        // Right side: duration + controls
        let mut right_parts = vec![];
        if let Some(duration) = self.duration {
            right_parts.push(format!("{}ms", duration));
        }
        if self.ai_explanation.is_some() {
            right_parts.push("AI".to_string());
        }
        right_parts.push(if self.collapsed { "▶" } else { "▼" }.to_string());
        
        let right_content = right_parts.join(" │ ");
        
        // Render left side
        buf.set_string(area.x + 1, area.y, &left_content, status_style);
        
        // Render right side
        let right_x = area.x + area.width - right_content.len() as u16 - 1;
        buf.set_string(right_x, area.y, &right_content, Style::default().fg(theme.fg_muted));
    }
    
    fn render_command(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Apply syntax highlighting to command
        let highlighted_command = self.highlight_command(&self.command);
        
        let command_widget = Paragraph::new(highlighted_command)
            .style(Style::default().fg(theme.fg_primary))
            .wrap(Wrap { trim: false });
        
        command_widget.render(area, buf);
    }
    
    fn render_output(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        let output_lines: Vec<Line> = self.output
            .lines()
            .map(|line| Line::from(line.to_string()))
            .collect();
        
        let output_widget = Paragraph::new(output_lines)
            .style(Style::default().fg(theme.fg_secondary))
            .wrap(Wrap { trim: false });
        
        output_widget.render(area, buf);
    }
    
    fn highlight_command(&self, command: &str) -> Text {
        // Simple syntax highlighting for common shell patterns
        // This would be expanded with proper lexing in production
        let mut spans = vec![];
        
        for word in command.split_whitespace() {
            if word.starts_with('-') {
                // Flags
                spans.push(Span::styled(
                    format!("{} ", word),
                    Style::default().fg(Color::Rgb(255, 123, 114))
                ));
            } else if word.contains('/') || word.contains('.') {
                // Paths
                spans.push(Span::styled(
                    format!("{} ", word),
                    Style::default().fg(Color::Rgb(121, 192, 255))
                ));
            } else if spans.is_empty() {
                // First word is likely the command
                spans.push(Span::styled(
                    format!("{} ", word),
                    Style::default().fg(Color::Rgb(210, 168, 255))
                ));
            } else {
                // Regular arguments
                spans.push(Span::raw(format!("{} ", word)));
            }
        }
        
        Text::from(Line::from(spans))
    }
}

/// Manages the collection of command blocks
pub struct BlockManager {
    blocks: Vec<CommandBlock>,
    selected_index: Option<usize>,
    scroll_offset: usize,
}

impl BlockManager {
    pub fn new() -> Self {
        Self {
            blocks: Vec::new(),
            selected_index: None,
            scroll_offset: 0,
        }
    }
    
    pub fn add_block(&mut self, block: CommandBlock) {
        self.blocks.push(block);
    }
    
    pub fn select_previous(&mut self) {
        if let Some(idx) = self.selected_index {
            if idx > 0 {
                self.selected_index = Some(idx - 1);
            }
        } else if !self.blocks.is_empty() {
            self.selected_index = Some(self.blocks.len() - 1);
        }
    }
    
    pub fn select_next(&mut self) {
        if let Some(idx) = self.selected_index {
            if idx < self.blocks.len() - 1 {
                self.selected_index = Some(idx + 1);
            }
        } else if !self.blocks.is_empty() {
            self.selected_index = Some(0);
        }
    }
    
    pub fn toggle_selected_collapse(&mut self) {
        if let Some(idx) = self.selected_index {
            if let Some(block) = self.blocks.get_mut(idx) {
                block.toggle_collapse();
            }
        }
    }
    
    pub fn get_selected_block(&self) -> Option<&CommandBlock> {
        self.selected_index.and_then(|idx| self.blocks.get(idx))
    }
    
    pub fn render(&mut self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Update selected state
        for (idx, block) in self.blocks.iter_mut().enumerate() {
            block.selected = self.selected_index == Some(idx);
        }
        
        // Calculate visible blocks based on scroll
        let mut y_offset = 0;
        for block in &self.blocks[self.scroll_offset..] {
            if y_offset >= area.height {
                break;
            }
            
            let block_height = if block.collapsed { 4 } else { 12 };
            if y_offset + block_height > area.height {
                break;
            }
            
            let block_area = Rect::new(
                area.x,
                area.y + y_offset,
                area.width,
                block_height.min(area.height - y_offset),
            );
            
            block.render(block_area, buf, theme);
            y_offset += block_height + 1; // 1 line spacing between blocks
        }
    }
}