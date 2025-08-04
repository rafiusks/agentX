use crossterm::{
    style::{Color, SetForegroundColor, ResetColor},
    execute,
    cursor,
    terminal::{Clear, ClearType},
};
use std::io::{stdout, Write};
use anyhow::Result;

pub struct HelpDisplay;

impl HelpDisplay {
    pub fn show_help() -> Result<()> {
        execute!(stdout(), Clear(ClearType::All), cursor::MoveTo(0, 0))?;
        
        // Header
        execute!(stdout(), SetForegroundColor(Color::Cyan))?;
        println!("🚀 AgentX Keyboard Shortcuts");
        execute!(stdout(), ResetColor)?;
        println!("═══════════════════════════════════════════════════════════════");
        println!();
        
        // Global shortcuts
        Self::print_section("Global Shortcuts", vec![
            ("Ctrl+Q", "Quit", "Exit AgentX"),
            ("Ctrl+Shift+M", "Mission Control", "Toggle Terminal UI mode"),
            ("Ctrl+Shift+P", "Pro Mode", "Toggle Advanced mode"),
            ("Tab", "Model Selection", "Choose AI provider/model"),
            ("F1", "Help", "Show this help screen"),
        ])?;
        
        // Current mode shortcuts
        Self::print_section("Current Mode", vec![
            ("Enter", "Send/Execute", "Submit query or command"),
            ("Esc", "Cancel/Clear", "Cancel operation or clear input"),
            ("↑/↓", "Navigate", "Move through history or items"),
            ("Ctrl+L", "Clear", "Clear the screen"),
        ])?;
        
        // Mission Control specific
        Self::print_section("Mission Control Mode", vec![
            ("Ctrl+K", "Command Palette", "Open command palette"),
            ("Ctrl+R", "Search", "Search command history"),
            ("Ctrl+E", "Export", "Export session"),
            ("Ctrl+↑/↓", "Jump Blocks", "Navigate command blocks"),
        ])?;
        
        // Tips
        execute!(stdout(), SetForegroundColor(Color::Yellow))?;
        println!("💡 Tips:");
        execute!(stdout(), ResetColor)?;
        println!("• Press Tab to quickly switch between AI models");
        println!("• Use Ctrl+Shift+M to toggle between UI modes");
        println!("• Type naturally - AgentX understands plain English");
        println!("• Configure shortcuts in ~/.agentx/config.toml");
        println!();
        
        execute!(stdout(), SetForegroundColor(Color::Green))?;
        println!("Press any key to return...");
        execute!(stdout(), ResetColor)?;
        
        stdout().flush()?;
        Ok(())
    }
    
    fn print_section(title: &str, shortcuts: Vec<(&str, &str, &str)>) -> Result<()> {
        execute!(stdout(), SetForegroundColor(Color::Green))?;
        println!("📋 {}", title);
        execute!(stdout(), ResetColor)?;
        
        for (key, action, desc) in shortcuts {
            execute!(stdout(), SetForegroundColor(Color::Blue))?;
            print!("  {:15}", key);
            execute!(stdout(), SetForegroundColor(Color::White))?;
            print!("{:20}", action);
            execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
            println!("{}", desc);
            execute!(stdout(), ResetColor)?;
        }
        println!();
        
        Ok(())
    }
    
    pub fn show_inline_help() -> String {
        "Ctrl+Q: Quit | Tab: Models | F1: Help | Ctrl+Shift+M: Terminal Mode".to_string()
    }
}