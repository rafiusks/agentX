pub mod blocks;
pub mod command_input;
pub mod command_palette;
pub mod theme;
pub mod layout;

use anyhow::Result;
use crossterm::event::KeyEvent;
use std::collections::VecDeque;
use tokio::sync::mpsc;

use crate::ui::{LayerType, UILayer};

/// Modern terminal interface inspired by Warp
pub struct TerminalLayer {
    /// Command blocks history
    blocks: VecDeque<blocks::CommandBlock>,
    
    /// Current command input
    command_input: command_input::CommandInput,
    
    /// Command palette (Cmd+K)
    command_palette: Option<command_palette::CommandPalette>,
    
    /// Layout manager
    layout: layout::LayoutManager,
    
    /// Theme
    theme: theme::Theme,
    
    /// Channel for sending commands
    tx: Option<mpsc::Sender<String>>,
    
    /// Current focus area
    focus: FocusArea,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum FocusArea {
    CommandInput,
    CommandPalette,
    Blocks,
}

impl TerminalLayer {
    pub fn new() -> Self {
        Self {
            blocks: VecDeque::new(),
            command_input: command_input::CommandInput::new(),
            command_palette: None,
            layout: layout::LayoutManager::new(),
            theme: theme::Theme::dark(),
            tx: None,
            focus: FocusArea::CommandInput,
        }
    }
    
    pub fn set_channel(&mut self, tx: mpsc::Sender<String>) {
        self.tx = Some(tx);
    }
    
    pub fn add_command_result(&mut self, command: String, output: String, success: bool) {
        let block = blocks::CommandBlock::new(command, output, success);
        self.blocks.push_back(block);
        
        // Keep only last 100 blocks
        while self.blocks.len() > 100 {
            self.blocks.pop_front();
        }
    }
    
    fn toggle_command_palette(&mut self) {
        if self.command_palette.is_some() {
            self.command_palette = None;
            self.focus = FocusArea::CommandInput;
        } else {
            self.command_palette = Some(command_palette::CommandPalette::new());
            self.focus = FocusArea::CommandPalette;
        }
    }
}

impl UILayer for TerminalLayer {
    fn layer_type(&self) -> LayerType {
        LayerType::Mission
    }
    
    fn render(&mut self) -> Result<()> {
        use crossterm::{
            execute,
            style::{Color, ResetColor, SetForegroundColor},
            terminal::{Clear, ClearType},
            cursor,
        };
        use std::io::{stdout, Write};
        
        let term_size = crossterm::terminal::size()?;
        
        // Clear screen
        execute!(stdout(), Clear(ClearType::All))?;
        
        // Draw header
        execute!(stdout(), cursor::MoveTo(0, 0))?;
        execute!(stdout(), SetForegroundColor(Color::Cyan))?;
        print!("🚀 AgentX Terminal - Mission Control");
        execute!(stdout(), ResetColor)?;
        
        // Draw command blocks (last few)
        let mut y = 2;
        let max_blocks = (term_size.1 as usize).saturating_sub(6);
        let blocks_to_show = self.blocks.iter().rev().take(max_blocks).collect::<Vec<_>>();
        
        for block in blocks_to_show.iter().rev() {
            if y >= term_size.1.saturating_sub(4) {
                break;
            }
            
            // Draw command
            execute!(stdout(), cursor::MoveTo(0, y))?;
            let status_icon = if block.success { "✅" } else { "❌" };
            execute!(stdout(), SetForegroundColor(Color::Green))?;
            print!("{} $ {}", status_icon, block.command);
            execute!(stdout(), ResetColor)?;
            y += 1;
            
            // Draw output (first few lines)
            let output_lines: Vec<&str> = block.output.lines().take(3).collect();
            for line in output_lines {
                if y >= term_size.1.saturating_sub(4) {
                    break;
                }
                execute!(stdout(), cursor::MoveTo(2, y))?;
                execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
                print!("{}", line);
                execute!(stdout(), ResetColor)?;
                y += 1;
            }
            y += 1; // Add spacing
        }
        
        // Draw command input area
        let input_y = term_size.1.saturating_sub(3);
        execute!(stdout(), cursor::MoveTo(0, input_y))?;
        execute!(stdout(), SetForegroundColor(Color::White))?;
        print!("┌{}┐", "─".repeat((term_size.0 - 2) as usize));
        
        execute!(stdout(), cursor::MoveTo(0, input_y + 1))?;
        print!("│ $ ");
        let current_input = self.command_input.get_current_input();
        print!("{}", current_input);
        let remaining_space = (term_size.0 as usize).saturating_sub(current_input.len() + 4);
        print!("{}│", " ".repeat(remaining_space));
        
        execute!(stdout(), cursor::MoveTo(0, input_y + 2))?;
        print!("└{}┘", "─".repeat((term_size.0 - 2) as usize));
        execute!(stdout(), ResetColor)?;
        
        // Draw command palette if active
        if let Some(palette) = &self.command_palette {
            let palette_y = term_size.1 / 4;
            let palette_height = term_size.1 / 2;
            let palette_width = (term_size.0 * 3) / 4;
            let palette_x = (term_size.0 - palette_width) / 2;
            
            // Draw palette box
            execute!(stdout(), cursor::MoveTo(palette_x, palette_y))?;
            execute!(stdout(), SetForegroundColor(Color::Yellow))?;
            print!("┌{}┐", "─".repeat((palette_width - 2) as usize));
            
            for i in 1..palette_height - 1 {
                execute!(stdout(), cursor::MoveTo(palette_x, palette_y + i))?;
                print!("│{}│", " ".repeat((palette_width - 2) as usize));
            }
            
            execute!(stdout(), cursor::MoveTo(palette_x, palette_y + palette_height - 1))?;
            print!("└{}┘", "─".repeat((palette_width - 2) as usize));
            
            // Draw palette title
            execute!(stdout(), cursor::MoveTo(palette_x + 2, palette_y + 1))?;
            execute!(stdout(), SetForegroundColor(Color::White))?;
            print!("Command Palette (Ctrl+K to close)");
            execute!(stdout(), ResetColor)?;
        }
        
        // Draw status line
        execute!(stdout(), cursor::MoveTo(0, term_size.1 - 1))?;
        execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
        print!("Ctrl+K: Command Palette | Ctrl+L: Clear | Ctrl+Shift+P: Toggle Pro Mode");
        execute!(stdout(), ResetColor)?;
        
        // Position cursor in input
        let cursor_x = 4 + self.command_input.get_cursor_position() as u16;
        execute!(stdout(), cursor::MoveTo(cursor_x, input_y + 1))?;
        execute!(stdout(), cursor::Show)?;
        
        stdout().flush()?;
        Ok(())
    }
    
    fn handle_key(&mut self, key: KeyEvent) -> Result<Option<LayerType>> {
        use crossterm::event::{KeyCode, KeyModifiers};
        
        // Global keybindings
        match (key.modifiers, key.code) {
            (KeyModifiers::CONTROL, KeyCode::Char('k')) => {
                self.toggle_command_palette();
                return Ok(None);
            }
            (KeyModifiers::CONTROL, KeyCode::Char('l')) => {
                // Clear blocks
                self.blocks.clear();
                return Ok(None);
            }
            _ => {}
        }
        
        // Handle input based on focus
        match self.focus {
            FocusArea::CommandInput => {
                if let Some(command) = self.command_input.handle_key(key)? {
                    // Send command for execution
                    if let Some(tx) = &self.tx {
                        let _ = tx.try_send(command.clone());
                        self.add_command_result(command, "Processing...".to_string(), true);
                    }
                }
            }
            FocusArea::CommandPalette => {
                if let Some(palette) = &mut self.command_palette {
                    if let Some(action) = palette.handle_key(key)? {
                        match action {
                            command_palette::PaletteAction::Close => {
                                self.command_palette = None;
                                self.focus = FocusArea::CommandInput;
                            }
                            command_palette::PaletteAction::ExecuteCommand(cmd) => {
                                self.command_input.set_text(&cmd);
                                self.command_palette = None;
                                self.focus = FocusArea::CommandInput;
                            }
                        }
                    }
                }
            }
            FocusArea::Blocks => {
                // Handle block navigation
            }
        }
        
        Ok(None)
    }
}