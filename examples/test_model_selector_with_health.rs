use agentx::config::{AgentXConfig, ProviderStatus};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🔍 Testing Model Selector with Health Checks...\n");
    
    let mut config = AgentXConfig::load()?;
    
    // Check provider health
    println!("🏥 Checking provider health...");
    let health_results = config.check_all_providers_health().await;
    
    println!("\n📋 Model Selection with Health Status:");
    println!("{}", "=".repeat(70));
    
    let models = config.list_available_models();
    for (i, (model_key, model)) in models.iter().enumerate() {
        let provider_name = model_key.split('/').next().unwrap_or("unknown");
        let health = health_results.get(provider_name);
        
        let status_info = if let Some(h) = health {
            match &h.status {
                ProviderStatus::Online => {
                    let response_time = h.response_time
                        .map(|d| format!("{}ms", d.as_millis()))
                        .unwrap_or_else(|| "?ms".to_string());
                    format!("🟢 Online ({})", response_time)
                }
                ProviderStatus::Offline => "🔴 Offline".to_string(),
                ProviderStatus::ConfigError(msg) => format!("🟡 Config: {}", msg),
                ProviderStatus::Unknown => "⚪ Unknown".to_string(),
            }
        } else {
            "❓ No health data".to_string()
        };
        
        let current_marker = if model_key == &format!("{}/{}", config.default_provider, config.default_model) {
            " ← CURRENT"
        } else {
            ""
        };
        
        println!("\n{}. {} - {}{}", 
            i + 1, 
            model.display_name,
            model.description.as_deref().unwrap_or("No description"),
            current_marker
        );
        println!("   Provider: {} / Model: {}", provider_name, model.name);
        println!("   Status: {}", status_info);
        println!("   Context: {} tokens | Streaming: {}", 
            model.context_size,
            if model.supports_streaming { "✅" } else { "❌" }
        );
        
        // Show available models from server if online
        if let Some(h) = health {
            if matches!(h.status, ProviderStatus::Online) {
                let available_on_server = h.available_models.contains(&model.name);
                println!("   Server: {}", 
                    if available_on_server { 
                        "✅ Available" 
                    } else { 
                        "❓ Not found (may need to pull/install)" 
                    }
                );
            }
        }
    }
    
    println!("\n{}", "=".repeat(70));
    
    // Simulate model selection UI behavior
    println!("\n🖱️  Model Selection UI Simulation:");
    println!("In the actual UI, you would see:");
    println!("┌────────────────────────── Select Model ──────────────────────────┐");
    println!("│                   ↑↓ Navigate  Enter Select  R Refresh  Esc Cancel              │");
    println!("│                                                                  │");
    
    for (i, (model_key, model)) in models.iter().enumerate().take(5) {
        let provider_name = model_key.split('/').next().unwrap_or("unknown");
        let status_icon = if let Some(health) = health_results.get(provider_name) {
            match health.status {
                ProviderStatus::Online => "🟢",
                ProviderStatus::Offline => "🔴",
                ProviderStatus::ConfigError(_) => "🟡",
                ProviderStatus::Unknown => "⚪",
            }
        } else {
            "❓"
        };
        
        let is_current = model_key == &format!("{}/{}", config.default_provider, config.default_model);
        let selection = if is_current { "→ " } else { "  " };
        
        println!("│{}{}  {} {} - {}{}│", 
            selection,
            status_icon,
            model.display_name,
            if model.description.is_some() { "..." } else { "" },
            if model.description.is_some() { "..." } else { &model.display_name },
            " ".repeat(20 - model.display_name.len().min(20))
        );
    }
    
    if models.len() > 5 {
        println!("│  ... and {} more models                                           │", models.len() - 5);
    }
    
    println!("│                                                                  │");
    println!("│  Context: 4096  Streaming: Yes                                  │");
    println!("└──────────────────────────────────────────────────────────────────┘");
    
    // Provide guidance
    println!("\n💡 How to use the model selector:");
    println!("   • Press Tab in the main UI to open model selection");
    println!("   • Use ↑↓ arrow keys to navigate models");
    println!("   • Green 🟢 = Provider online and ready");
    println!("   • Red 🔴 = Provider offline or unreachable");
    println!("   • Yellow 🟡 = Configuration issue (e.g., missing API key)");
    println!("   • Press Enter to select a model");
    println!("   • Press R (when implemented) to refresh health status");
    
    // Test model switching
    if let Some(online_model) = models.iter().find(|(key, _)| {
        let provider = key.split('/').next().unwrap_or("");
        health_results.get(provider)
            .map(|h| matches!(h.status, ProviderStatus::Online))
            .unwrap_or(false)
    }) {
        println!("\n🔄 Found online provider model: {}", online_model.0);
        println!("   You could switch to this model for immediate use!");
    } else {
        println!("\n⚠️  No online providers found.");
        println!("   AgentX will use mock responses until a provider is available.");
        println!("   Try starting Ollama with: ollama serve");
    }
    
    Ok(())
}