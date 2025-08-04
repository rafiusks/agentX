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
        // Terminal UI rendering will be implemented
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