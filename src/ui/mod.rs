pub mod simple;
pub mod components;
pub mod terminal;

use anyhow::Result;
use crossterm::event::KeyEvent;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LayerType {
    Simple,   // Layer 1: Consumer Simple
    Mission,  // Layer 2: Prosumer Power
    Pro,      // Layer 3: Professional Deep
}

pub trait UILayer: Send {
    fn layer_type(&self) -> LayerType;
    fn render(&mut self) -> Result<()>;
    fn handle_key(&mut self, key: KeyEvent) -> Result<Option<LayerType>>;
}

pub fn create_simple_layer() -> simple::SimpleLayer {
    simple::SimpleLayer::new()
}

pub fn create_terminal_layer() -> terminal::TerminalLayer {
    terminal::TerminalLayer::new()
}