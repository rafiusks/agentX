use agentx::config::{AgentXConfig, ProviderType};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("Testing AgentX configuration system...\n");
    
    // Load or create default configuration
    let mut config = AgentXConfig::load()?;
    
    println!("📋 Current Configuration:");
    println!("  Default Provider: {}", config.default_provider);
    println!("  Default Model: {}", config.default_model);
    println!("  Show Model Selector: {}", config.ui.show_model_selector);
    println!("  Auto Streaming: {}", config.ui.auto_stream);
    
    println!("\n🤖 Available Models:");
    let models = config.list_available_models();
    for (i, (model_key, model)) in models.iter().enumerate() {
        let current = if model_key == &format!("{}/{}", config.default_provider, config.default_model) {
            " ← CURRENT"
        } else {
            ""
        };
        
        println!("  {}. {} - {}{}", 
            i + 1, 
            model.display_name,
            model.description.as_deref().unwrap_or("No description"),
            current
        );
        println!("     Provider: {}", model_key.split('/').next().unwrap_or("unknown"));
        println!("     Context Size: {} tokens", model.context_size);
        println!("     Streaming: {}", if model.supports_streaming { "✅" } else { "❌" });
        println!();
    }
    
    println!("🔧 Provider Details:");
    for (name, provider) in &config.providers {
        let status = match provider.provider_type {
            ProviderType::Ollama => {
                if provider.base_url.contains("localhost") {
                    "🟡 Local (may not be running)"
                } else {
                    "🟢 Remote"
                }
            }
            ProviderType::OpenAI | ProviderType::Anthropic => {
                if provider.api_key.is_some() {
                    "🟢 Configured"
                } else {
                    "🔴 API key required"
                }
            }
            ProviderType::OpenAICompatible => "🟡 Compatible endpoint"
        };
        
        println!("  {} ({})", provider.name, name);
        println!("    URL: {}", provider.base_url);
        println!("    Status: {}", status);
        println!("    Models: {}", provider.models.len());
        println!();
    }
    
    // Test model switching
    println!("🔄 Testing model switching...");
    let original_model = config.default_model.clone();
    
    // Find a different model to switch to
    if let Some((test_key, _)) = models.iter().find(|(key, _)| !key.ends_with(&original_model)) {
        let parts: Vec<&str> = test_key.split('/').collect();
        if parts.len() == 2 {
            println!("  Switching to: {}", test_key);
            config.set_default_model(parts[0].to_string(), parts[1].to_string())?;
            println!("  ✅ Switched successfully!");
            
            // Switch back
            let original_key = format!("{}/{}", config.default_provider, original_model);
            let parts: Vec<&str> = original_key.split('/').collect();
            config.set_default_model(parts[0].to_string(), parts[1].to_string())?;
            println!("  ↩️  Switched back to original model");
        }
    }
    
    println!("\n📁 Configuration saved to: ~/.config/agentx/config.toml");
    println!("💡 You can edit this file to customize providers and models");
    
    Ok(())
}