use crossterm::event::{KeyCode, KeyEvent};
use fuzzy_matcher::FuzzyMatcher;
use fuzzy_matcher::skim::SkimMatcherV2;
use ratatui::{
    prelude::*,
    widgets::{Block, Borders, BorderType, Clear, List, ListItem, ListState},
};
use anyhow::Result;

/// Command palette for quick access to commands and workflows
pub struct CommandPalette {
    /// Search input
    search: String,
    
    /// Cursor position in search
    cursor_pos: usize,
    
    /// All available items
    items: Vec<PaletteItem>,
    
    /// Filtered items based on search
    filtered_items: Vec<usize>,
    
    /// Currently selected item
    selected: usize,
    
    /// List state for scrolling
    list_state: ListState,
    
    /// Fuzzy matcher
    matcher: SkimMatcherV2,
}

#[derive(Debug, Clone)]
pub struct PaletteItem {
    /// Display title
    pub title: String,
    
    /// Optional description
    pub description: Option<String>,
    
    /// The action to perform
    pub action: PaletteItemAction,
    
    /// Category for grouping
    pub category: PaletteCategory,
    
    /// Optional keyboard shortcut
    pub shortcut: Option<String>,
}

#[derive(Debug, Clone)]
pub enum PaletteItemAction {
    /// Execute a shell command
    Command(String),
    
    /// Run an AI workflow
    Workflow(String),
    
    /// Open a saved command
    SavedCommand(String),
    
    /// Insert a template
    Template(String),
    
    /// Navigate to a location
    Navigate(String),
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum PaletteCategory {
    RecentCommands,
    SavedCommands,
    Workflows,
    Templates,
    Actions,
    AI,
}

#[derive(Debug, Clone)]
pub enum PaletteAction {
    Close,
    ExecuteCommand(String),
}

impl CommandPalette {
    pub fn new() -> Self {
        let mut palette = Self {
            search: String::new(),
            cursor_pos: 0,
            items: Vec::new(),
            filtered_items: Vec::new(),
            selected: 0,
            list_state: ListState::default(),
            matcher: SkimMatcherV2::default(),
        };
        
        palette.load_default_items();
        palette.update_filter();
        palette
    }
    
    fn load_default_items(&mut self) {
        // Recent commands
        self.items.extend([
            PaletteItem {
                title: "git status".to_string(),
                description: Some("Show working tree status".to_string()),
                action: PaletteItemAction::Command("git status".to_string()),
                category: PaletteCategory::RecentCommands,
                shortcut: None,
            },
            PaletteItem {
                title: "cargo build".to_string(),
                description: Some("Build the current project".to_string()),
                action: PaletteItemAction::Command("cargo build".to_string()),
                category: PaletteCategory::RecentCommands,
                shortcut: None,
            },
        ]);
        
        // AI Workflows
        self.items.extend([
            PaletteItem {
                title: "Explain this error".to_string(),
                description: Some("Get AI help with the last error".to_string()),
                action: PaletteItemAction::Workflow("explain_error".to_string()),
                category: PaletteCategory::AI,
                shortcut: Some("⌘E".to_string()),
            },
            PaletteItem {
                title: "Generate test cases".to_string(),
                description: Some("AI creates tests for selected code".to_string()),
                action: PaletteItemAction::Workflow("generate_tests".to_string()),
                category: PaletteCategory::AI,
                shortcut: Some("⌘T".to_string()),
            },
            PaletteItem {
                title: "Refactor code".to_string(),
                description: Some("AI suggests refactoring improvements".to_string()),
                action: PaletteItemAction::Workflow("refactor".to_string()),
                category: PaletteCategory::AI,
                shortcut: None,
            },
        ]);
        
        // Templates
        self.items.extend([
            PaletteItem {
                title: "Docker compose template".to_string(),
                description: Some("Basic docker-compose.yml structure".to_string()),
                action: PaletteItemAction::Template("docker_compose".to_string()),
                category: PaletteCategory::Templates,
                shortcut: None,
            },
            PaletteItem {
                title: "GitHub Actions workflow".to_string(),
                description: Some("CI/CD workflow template".to_string()),
                action: PaletteItemAction::Template("github_actions".to_string()),
                category: PaletteCategory::Templates,
                shortcut: None,
            },
        ]);
        
        // Actions
        self.items.extend([
            PaletteItem {
                title: "Clear terminal".to_string(),
                description: Some("Clear all command blocks".to_string()),
                action: PaletteItemAction::Command("clear".to_string()),
                category: PaletteCategory::Actions,
                shortcut: Some("⌘L".to_string()),
            },
            PaletteItem {
                title: "Share session".to_string(),
                description: Some("Generate shareable link for this session".to_string()),
                action: PaletteItemAction::Workflow("share_session".to_string()),
                category: PaletteCategory::Actions,
                shortcut: None,
            },
        ]);
    }
    
    fn update_filter(&mut self) {
        if self.search.is_empty() {
            // Show all items grouped by category when no search
            self.filtered_items = (0..self.items.len()).collect();
        } else {
            // Fuzzy match items
            let mut matches: Vec<(usize, i64)> = self.items
                .iter()
                .enumerate()
                .filter_map(|(idx, item)| {
                    let score = self.matcher.fuzzy_match(&item.title, &self.search)
                        .or_else(|| {
                            item.description.as_ref()
                                .and_then(|desc| self.matcher.fuzzy_match(desc, &self.search))
                        });
                    score.map(|s| (idx, s))
                })
                .collect();
            
            // Sort by score (highest first)
            matches.sort_by_key(|(_, score)| -score);
            
            self.filtered_items = matches.into_iter().map(|(idx, _)| idx).collect();
        }
        
        // Reset selection
        self.selected = 0;
        self.list_state.select(Some(0));
    }
    
    pub fn handle_key(&mut self, key: KeyEvent) -> Result<Option<PaletteAction>> {
        match key.code {
            KeyCode::Esc => {
                return Ok(Some(PaletteAction::Close));
            }
            
            KeyCode::Enter => {
                if let Some(&idx) = self.filtered_items.get(self.selected) {
                    let item = &self.items[idx];
                    match &item.action {
                        PaletteItemAction::Command(cmd) => {
                            return Ok(Some(PaletteAction::ExecuteCommand(cmd.clone())));
                        }
                        PaletteItemAction::Workflow(workflow) => {
                            // In real implementation, this would trigger the workflow
                            return Ok(Some(PaletteAction::ExecuteCommand(
                                format!("agentx workflow {}", workflow)
                            )));
                        }
                        PaletteItemAction::Template(template) => {
                            // In real implementation, this would insert the template
                            return Ok(Some(PaletteAction::ExecuteCommand(
                                format!("agentx template {}", template)
                            )));
                        }
                        _ => {}
                    }
                }
            }
            
            KeyCode::Up => {
                if self.selected > 0 {
                    self.selected -= 1;
                    self.list_state.select(Some(self.selected));
                }
            }
            
            KeyCode::Down => {
                if self.selected < self.filtered_items.len().saturating_sub(1) {
                    self.selected += 1;
                    self.list_state.select(Some(self.selected));
                }
            }
            
            KeyCode::Char(c) => {
                self.search.insert(self.cursor_pos, c);
                self.cursor_pos += 1;
                self.update_filter();
            }
            
            KeyCode::Backspace => {
                if self.cursor_pos > 0 {
                    self.search.remove(self.cursor_pos - 1);
                    self.cursor_pos -= 1;
                    self.update_filter();
                }
            }
            
            KeyCode::Left => {
                if self.cursor_pos > 0 {
                    self.cursor_pos -= 1;
                }
            }
            
            KeyCode::Right => {
                if self.cursor_pos < self.search.len() {
                    self.cursor_pos += 1;
                }
            }
            
            _ => {}
        }
        
        Ok(None)
    }
    
    pub fn render(&mut self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        // Calculate centered modal size
        let modal_width = 80.min(area.width - 4);
        let modal_height = 20.min(area.height - 4);
        
        let modal_area = Rect {
            x: (area.width - modal_width) / 2,
            y: (area.height - modal_height) / 2,
            width: modal_width,
            height: modal_height,
        };
        
        // Clear background
        Clear.render(modal_area, buf);
        
        // Modal container with shadow effect
        let shadow_area = Rect {
            x: modal_area.x + 1,
            y: modal_area.y + 1,
            width: modal_area.width,
            height: modal_area.height,
        };
        
        // Render shadow
        for y in shadow_area.y..shadow_area.y + shadow_area.height {
            for x in shadow_area.x..shadow_area.x + shadow_area.width {
                if x < buf.area.width && y < buf.area.height {
                    buf[(x, y)].set_style(Style::default().bg(theme.shadow));
                }
            }
        }
        
        // Main modal
        let block = Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(Style::default().fg(theme.border))
            .style(Style::default().bg(theme.bg_tertiary));
        
        let inner = block.inner(modal_area);
        block.render(modal_area, buf);
        
        // Layout: search bar at top, list below
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Length(3),  // Search
                Constraint::Min(5),     // Results
            ])
            .split(inner);
        
        // Search input
        self.render_search(chunks[0], buf, theme);
        
        // Results list
        self.render_results(chunks[1], buf, theme);
    }
    
    fn render_search(&self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        let search_block = Block::default()
            .borders(Borders::BOTTOM)
            .border_style(Style::default().fg(theme.border));
        
        let inner = search_block.inner(area);
        search_block.render(area, buf);
        
        // Search icon and input
        buf.set_string(inner.x, inner.y, "🔍 ", Style::default().fg(theme.fg_accent));
        
        let search_text = if self.search.is_empty() {
            "Search commands, workflows, and actions...".to_string()
        } else {
            self.search.clone()
        };
        
        let search_style = if self.search.is_empty() {
            Style::default().fg(theme.fg_muted).add_modifier(Modifier::ITALIC)
        } else {
            Style::default().fg(theme.fg_primary)
        };
        
        buf.set_string(inner.x + 3, inner.y, &search_text, search_style);
        
        // Cursor
        if !self.search.is_empty() {
            let cursor_x = inner.x + 3 + self.cursor_pos as u16;
            buf.set_string(cursor_x, inner.y, "█", theme.input_cursor_style());
        }
    }
    
    fn render_results(&mut self, area: Rect, buf: &mut Buffer, theme: &crate::ui::terminal::theme::Theme) {
        let items: Vec<ListItem> = self.filtered_items
            .iter()
            .map(|&idx| {
                let item = &self.items[idx];
                let icon = Self::get_category_icon_static(item.category);
                let mut lines = vec![
                    Line::from(vec![
                        Span::raw(icon),
                        Span::raw(" "),
                        Span::styled(&item.title, Style::default().fg(theme.fg_primary)),
                    ])
                ];
                
                if let Some(desc) = &item.description {
                    lines.push(Line::from(vec![
                        Span::raw("   "),
                        Span::styled(desc, Style::default().fg(theme.fg_muted)),
                    ]));
                }
                
                if let Some(shortcut) = &item.shortcut {
                    lines[0].spans.push(Span::raw(" "));
                    lines[0].spans.push(Span::styled(
                        shortcut,
                        Style::default().fg(theme.fg_muted).add_modifier(Modifier::DIM),
                    ));
                }
                
                ListItem::new(lines)
            })
            .collect();
        
        let list = List::new(items)
            .highlight_style(theme.selected_style())
            .highlight_symbol("▶ ");
        
        StatefulWidget::render(list, area, buf, &mut self.list_state);
    }
    
    fn get_category_icon_static(category: PaletteCategory) -> &'static str {
        match category {
            PaletteCategory::RecentCommands => "⏱",
            PaletteCategory::SavedCommands => "⭐",
            PaletteCategory::Workflows => "⚡",
            PaletteCategory::Templates => "📄",
            PaletteCategory::Actions => "⚙️",
            PaletteCategory::AI => "🤖",
        }
    }
    
    fn get_category_icon(&self, category: PaletteCategory) -> &'static str {
        match category {
            PaletteCategory::RecentCommands => "⏱",
            PaletteCategory::SavedCommands => "⭐",
            PaletteCategory::Workflows => "⚡",
            PaletteCategory::Templates => "📄",
            PaletteCategory::Actions => "⚙️",
            PaletteCategory::AI => "🤖",
        }
    }
}