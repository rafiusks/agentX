use ratatui::prelude::*;

/// Manages the terminal layout with responsive design
pub struct LayoutManager {
    /// Current terminal size
    terminal_size: (u16, u16),
    
    /// Layout mode based on terminal size
    layout_mode: LayoutMode,
    
    /// Whether to show sidebar
    show_sidebar: bool,
    
    /// Split ratio for main/sidebar
    sidebar_ratio: f32,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum LayoutMode {
    /// Narrow terminal - stack vertically
    Compact,
    
    /// Medium width - standard layout
    Standard,
    
    /// Wide terminal - show additional panels
    Wide,
}

impl LayoutManager {
    pub fn new() -> Self {
        Self {
            terminal_size: (80, 24),
            layout_mode: LayoutMode::Standard,
            show_sidebar: false,
            sidebar_ratio: 0.3,
        }
    }
    
    pub fn update_size(&mut self, width: u16, height: u16) {
        self.terminal_size = (width, height);
        
        // Determine layout mode based on width
        self.layout_mode = match width {
            0..=80 => LayoutMode::Compact,
            81..=120 => LayoutMode::Standard,
            _ => LayoutMode::Wide,
        };
        
        // Auto-show sidebar in wide mode
        if self.layout_mode == LayoutMode::Wide {
            self.show_sidebar = true;
        }
    }
    
    pub fn toggle_sidebar(&mut self) {
        self.show_sidebar = !self.show_sidebar;
    }
    
    /// Get the main layout chunks
    pub fn get_layout(&self, area: Rect) -> TerminalLayout {
        match self.layout_mode {
            LayoutMode::Compact => self.compact_layout(area),
            LayoutMode::Standard => self.standard_layout(area),
            LayoutMode::Wide => self.wide_layout(area),
        }
    }
    
    fn compact_layout(&self, area: Rect) -> TerminalLayout {
        // In compact mode, everything is stacked vertically
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Min(10),      // Blocks area
                Constraint::Length(5),    // Command input
                Constraint::Length(1),    // Status bar
            ])
            .split(area);
        
        TerminalLayout {
            blocks_area: chunks[0],
            input_area: chunks[1],
            status_area: chunks[2],
            sidebar_area: None,
            ai_panel_area: None,
        }
    }
    
    fn standard_layout(&self, area: Rect) -> TerminalLayout {
        if self.show_sidebar {
            // Split horizontally first
            let main_chunks = Layout::default()
                .direction(Direction::Horizontal)
                .constraints([
                    Constraint::Percentage((100.0 * (1.0 - self.sidebar_ratio)) as u16),
                    Constraint::Percentage((100.0 * self.sidebar_ratio) as u16),
                ])
                .split(area);
            
            // Then split the main area vertically
            let left_chunks = Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Min(10),      // Blocks area
                    Constraint::Length(5),    // Command input
                    Constraint::Length(1),    // Status bar
                ])
                .split(main_chunks[0]);
            
            TerminalLayout {
                blocks_area: left_chunks[0],
                input_area: left_chunks[1],
                status_area: left_chunks[2],
                sidebar_area: Some(main_chunks[1]),
                ai_panel_area: None,
            }
        } else {
            // No sidebar - use full width
            self.compact_layout(area)
        }
    }
    
    fn wide_layout(&self, area: Rect) -> TerminalLayout {
        // In wide mode, we can show AI panel on the right
        let main_chunks = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage(70),  // Main area
                Constraint::Percentage(30),  // AI panel
            ])
            .split(area);
        
        // Split main area
        let left_chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Min(10),      // Blocks area
                Constraint::Length(5),    // Command input
                Constraint::Length(1),    // Status bar
            ])
            .split(main_chunks[0]);
        
        TerminalLayout {
            blocks_area: left_chunks[0],
            input_area: left_chunks[1],
            status_area: left_chunks[2],
            sidebar_area: None,
            ai_panel_area: Some(main_chunks[1]),
        }
    }
}

/// The calculated layout areas
pub struct TerminalLayout {
    /// Main area for command blocks
    pub blocks_area: Rect,
    
    /// Command input area
    pub input_area: Rect,
    
    /// Status bar area
    pub status_area: Rect,
    
    /// Optional sidebar area
    pub sidebar_area: Option<Rect>,
    
    /// Optional AI assistant panel
    pub ai_panel_area: Option<Rect>,
}

/// Status bar component
pub struct StatusBar {
    mode: String,
    branch: Option<String>,
    notifications: Vec<String>,
}

impl StatusBar {
    pub fn new() -> Self {
        Self {
            mode: "Normal".to_string(),
            branch: None,
            notifications: Vec::new(),
        }
    }
    
    pub fn set_git_branch(&mut self, branch: String) {
        self.branch = Some(branch);
    }
    
    pub fn add_notification(&mut self, msg: String) {
        self.notifications.push(msg);
        // Keep only last 5 notifications
        if self.notifications.len() > 5 {
            self.notifications.remove(0);
        }
    }
    
    pub fn render(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Clear the status bar area
        for x in area.x..area.x + area.width {
            if x < buf.area.width && area.y < buf.area.height {
                buf[(x, area.y)].set_style(Style::default().bg(theme.bg_accent).fg(theme.fg_primary));
            }
        }
        
        // Left side: mode and branch
        let mut left_content = vec![self.mode.clone()];
        if let Some(branch) = &self.branch {
            left_content.push(format!("git:{}", branch));
        }
        let left_text = left_content.join(" │ ");
        buf.set_string(area.x + 1, area.y, &left_text, Style::default().fg(theme.fg_secondary));
        
        // Right side: notifications
        if !self.notifications.is_empty() {
            let notif = &self.notifications.last().unwrap();
            let right_x = area.x + area.width - notif.len() as u16 - 2;
            buf.set_string(right_x, area.y, notif, Style::default().fg(theme.info));
        }
    }
}