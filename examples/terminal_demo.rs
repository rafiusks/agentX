use agentx::ui::terminal::TerminalLayer;
use agentx::ui::{UILayer, LayerType};
use crossterm::event::{self, Event, KeyCode, KeyModifiers};
use crossterm::terminal::{self, EnterAlternateScreen, LeaveAlternateScreen};
use crossterm::execute;
use anyhow::Result;
use std::io::stdout;
use std::time::Duration;
use tokio::sync::mpsc;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🚀 AgentX Terminal Demo");
    println!("Press any key to start the terminal UI...");
    println!("(Use Ctrl+C to exit once started)");
    
    // Wait for user input
    loop {
        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(_) = event::read()? {
                break;
            }
        }
    }
    
    // Setup terminal
    terminal::enable_raw_mode()?;
    execute!(stdout(), EnterAlternateScreen)?;
    
    // Create terminal layer
    let (tx, mut rx) = mpsc::channel(100);
    let mut terminal_layer = TerminalLayer::new();
    terminal_layer.set_channel(tx);
    
    // Add some demo command blocks
    terminal_layer.add_command_result(
        "hello world".to_string(),
        "Hello, World from AgentX!".to_string(),
        true
    );
    terminal_layer.add_command_result(
        "list files".to_string(),
        "README.md\nCargo.toml\nsrc/\nexamples/".to_string(),
        true
    );
    terminal_layer.add_command_result(
        "broken command".to_string(),
        "Error: Command not found".to_string(),
        false
    );
    
    println!("\n🎯 Terminal UI Active!");
    println!("Try typing commands and pressing Enter");
    println!("Press Ctrl+C to exit");
    
    // Main loop
    loop {
        // Handle any background messages
        while let Ok(command) = rx.try_recv() {
            println!("Received command: {}", command);
            // Simulate processing
            terminal_layer.add_command_result(
                command.clone(),
                format!("✅ Processed: {}", command),
                true
            );
        }
        
        // Render
        if let Err(e) = terminal_layer.render() {
            eprintln!("Render error: {}", e);
            break;
        }
        
        // Handle events
        if event::poll(Duration::from_millis(50))? {
            if let Event::Key(key) = event::read()? {
                // Check for exit
                if key.modifiers.contains(KeyModifiers::CONTROL) && key.code == KeyCode::Char('c') {
                    break;
                }
                
                // Pass to terminal layer
                if let Ok(Some(_transition)) = terminal_layer.handle_key(key) {
                    // Handle layer transitions if needed
                }
            }
        }
    }
    
    // Cleanup
    terminal::disable_raw_mode()?;
    execute!(stdout(), LeaveAlternateScreen)?;
    
    println!("\n👋 Terminal demo ended!");
    Ok(())
}