use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🚀 AgentX Interactive Chat Test");
    println!("═══════════════════════════════════════════════════");
    println!("This test verifies that the interactive terminal UI works");
    println!("when run in a real terminal environment.");
    println!();
    
    // Check if we're in a real terminal by testing raw mode capability
    let is_interactive = match crossterm::terminal::enable_raw_mode() {
        Ok(_) => {
            let _ = crossterm::terminal::disable_raw_mode();
            true
        }
        Err(_) => false,
    };
    
    if is_interactive {
        println!("✅ Running in interactive terminal - starting full UI");
        println!("   Use Ctrl+Q to quit, Tab to switch models");
        println!("   Ctrl+Shift+M toggles Mission Control mode");
        println!("   Ctrl+Shift+P toggles Pro mode");
        println!();
        
        // Create and run the application
        let mut app = agentx::app::Application::new()?;
        app.run().await?;
    } else {
        println!("⚠️  Not running in interactive terminal");
        println!("   To test the full UI, run this in a regular terminal:");
        println!("   cargo run --example test_chat");
        println!();
        println!("📋 What you would see in interactive mode:");
        println!("• Layer 1 (Simple): Chat interface with streaming responses");
        println!("• Layer 2 (Mission Control): Command blocks like Warp terminal");
        println!("• Layer 3 (Pro): Advanced development environment");
        println!();
        println!("🔧 Available features:");
        println!("• Multi-provider LLM support (OpenAI, Anthropic, Ollama)");
        println!("• Streaming responses with real-time feedback");
        println!("• Model selection with Tab key");
        println!("• Progressive UI disclosure based on usage");
        println!("• Terminal integration with command execution");
        println!();
        println!("🎯 Current project status: Phase 1 complete");
        println!("   ✅ Unified LLM interface");
        println!("   ✅ Terminal UI components");
        println!("   ✅ Streaming and model selection");
        println!("   ✅ Configuration system");
        println!("   📋 Next: MCP integration, advanced features");
    }
    
    Ok(())
}