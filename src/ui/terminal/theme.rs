use ratatui::style::{Color, Modifier, Style};

/// AgentX terminal theme inspired by Warp's modern aesthetic
#[derive(Debug, Clone)]
pub struct Theme {
    // Background colors
    pub bg_primary: Color,
    pub bg_secondary: Color,
    pub bg_tertiary: Color,
    pub bg_accent: Color,
    
    // Foreground colors
    pub fg_primary: Color,
    pub fg_secondary: Color,
    pub fg_muted: Color,
    pub fg_accent: Color,
    
    // Semantic colors
    pub success: Color,
    pub error: Color,
    pub warning: Color,
    pub info: Color,
    
    // Syntax highlighting
    pub syntax_keyword: Color,
    pub syntax_string: Color,
    pub syntax_number: Color,
    pub syntax_comment: Color,
    pub syntax_function: Color,
    pub syntax_variable: Color,
    
    // Special colors
    pub selection: Color,
    pub cursor: Color,
    pub border: Color,
    pub shadow: Color,
}

impl Theme {
    /// Dark theme with sophisticated color palette
    pub fn dark() -> Self {
        Self {
            // Backgrounds - deep, rich tones
            bg_primary: Color::Rgb(13, 17, 23),      // Deep space black
            bg_secondary: Color::Rgb(22, 27, 34),    // Slightly lighter
            bg_tertiary: Color::Rgb(30, 37, 46),     // Panel background
            bg_accent: Color::Rgb(48, 54, 61),       // Hover states
            
            // Foregrounds - high contrast, easy on eyes
            fg_primary: Color::Rgb(201, 209, 217),   // Primary text
            fg_secondary: Color::Rgb(139, 148, 158), // Secondary text
            fg_muted: Color::Rgb(88, 96, 105),      // Muted text
            fg_accent: Color::Rgb(88, 166, 255),    // Accent blue
            
            // Semantic colors - muted but distinguishable
            success: Color::Rgb(87, 171, 90),       // Soft green
            error: Color::Rgb(248, 81, 73),         // Soft red
            warning: Color::Rgb(251, 188, 4),       // Warm yellow
            info: Color::Rgb(88, 166, 255),         // Information blue
            
            // Syntax highlighting - inspired by GitHub's theme
            syntax_keyword: Color::Rgb(255, 123, 114),  // Coral
            syntax_string: Color::Rgb(121, 192, 255),   // Sky blue
            syntax_number: Color::Rgb(255, 166, 87),    // Orange
            syntax_comment: Color::Rgb(88, 96, 105),    // Gray
            syntax_function: Color::Rgb(210, 168, 255), // Purple
            syntax_variable: Color::Rgb(121, 192, 255), // Light blue
            
            // Special UI colors
            selection: Color::Rgb(48, 65, 94),      // Selection blue
            cursor: Color::Rgb(88, 166, 255),       // Bright cursor
            border: Color::Rgb(48, 54, 61),         // Subtle borders
            shadow: Color::Rgb(0, 0, 0),            // Pure black for shadows
        }
    }
    
    /// Get style for different UI elements
    pub fn block_style(&self) -> Style {
        Style::default()
            .bg(self.bg_secondary)
            .fg(self.fg_primary)
    }
    
    pub fn block_border_style(&self) -> Style {
        Style::default()
            .fg(self.border)
            .add_modifier(Modifier::DIM)
    }
    
    pub fn input_style(&self) -> Style {
        Style::default()
            .bg(self.bg_tertiary)
            .fg(self.fg_primary)
    }
    
    pub fn input_cursor_style(&self) -> Style {
        Style::default()
            .bg(self.cursor)
            .fg(self.bg_primary)
            .add_modifier(Modifier::BOLD)
    }
    
    pub fn suggestion_style(&self) -> Style {
        Style::default()
            .fg(self.fg_muted)
            .add_modifier(Modifier::ITALIC)
    }
    
    pub fn selected_style(&self) -> Style {
        Style::default()
            .bg(self.selection)
            .fg(self.fg_primary)
    }
    
    pub fn error_style(&self) -> Style {
        Style::default()
            .fg(self.error)
            .add_modifier(Modifier::BOLD)
    }
    
    pub fn success_style(&self) -> Style {
        Style::default()
            .fg(self.success)
    }
    
    pub fn ai_badge_style(&self) -> Style {
        Style::default()
            .bg(self.syntax_function)
            .fg(self.bg_primary)
            .add_modifier(Modifier::BOLD)
    }
}