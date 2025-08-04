use agentx::ui::{create_terminal_layer, UILayer};
use anyhow::Result;
use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::io;
use tokio::sync::mpsc;

#[tokio::main]
async fn main() -> Result<()> {
    // Setup terminal
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    
    // Create terminal layer
    let mut terminal_ui = create_terminal_layer();
    
    // Create channel for commands
    let (tx, mut rx) = mpsc::channel::<String>(100);
    terminal_ui.set_channel(tx);
    
    // Demo: Add some example command blocks
    terminal_ui.add_command_result(
        "git status --porcelain".to_string(),
        "M  src/main.rs\n?? docs/design.md".to_string(),
        true,
    );
    
    terminal_ui.add_command_result(
        "cargo build --release".to_string(),
        "   Compiling agentx v0.1.0\n    Finished release [optimized] target(s) in 12.5s".to_string(),
        true,
    );
    
    terminal_ui.add_command_result(
        "cargo test".to_string(),
        "test result: FAILED. 1 passed; 1 failed; 0 ignored".to_string(),
        false,
    );
    
    // Event loop
    loop {
        terminal_ui.render()?;
        
        if event::poll(std::time::Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if key.code == KeyCode::Char('q') && key.modifiers.contains(event::KeyModifiers::CONTROL) {
                    break;
                }
                
                terminal_ui.handle_key(key)?;
            }
        }
        
        // Check for command results
        if let Ok(command) = rx.try_recv() {
            // Simulate command execution
            tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;
            
            let output = match command.as_str() {
                cmd if cmd.starts_with("echo") => {
                    cmd.strip_prefix("echo ").unwrap_or("").to_string()
                }
                "ls" => {
                    "Cargo.toml\nCargo.lock\nsrc/\ntarget/\nexamples/\ndocs/".to_string()
                }
                "pwd" => {
                    "/Users/Rafael.Vidal/Code/agentX".to_string()
                }
                _ => {
                    format!("Executed: {}", command)
                }
            };
            
            terminal_ui.add_command_result(command, output, true);
        }
    }
    
    // Restore terminal
    disable_raw_mode()?;
    execute!(stdout, LeaveAlternateScreen)?;
    
    Ok(())
}