use agentx::config::{AgentXConfig, ProviderStatus};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🏥 Testing provider health checks...\n");
    
    let config = AgentXConfig::load()?;
    
    println!("🔍 Checking all providers...");
    let health_results = config.check_all_providers_health().await;
    
    println!("\n📊 Provider Health Report:");
    println!("{}", "=".repeat(60));
    
    for (provider_name, health) in &health_results {
        let provider_config = config.get_provider(provider_name).unwrap();
        
        println!("\n🏷️  Provider: {} ({})", provider_config.name, provider_name);
        print!("   Status: {} {}", health.status.icon(), health.status.description());
        
        if let Some(response_time) = health.response_time {
            println!(" ({}ms)", response_time.as_millis());
        } else {
            println!();
        }
        
        println!("   URL: {}", provider_config.base_url);
        
        match &health.status {
            ProviderStatus::Online => {
                println!("   ✅ Available models from server: {}", health.available_models.len());
                if !health.available_models.is_empty() {
                    let display_count = 3.min(health.available_models.len());
                    for model in health.available_models.iter().take(display_count) {
                        println!("      • {}", model);
                    }
                    if health.available_models.len() > display_count {
                        println!("      ... and {} more", health.available_models.len() - display_count);
                    }
                }
                
                println!("   📋 Configured models: {}", provider_config.models.len());
                for model in &provider_config.models {
                    let available = if health.available_models.contains(&model.name) {
                        "✅"
                    } else {
                        "❓"
                    };
                    println!("      {} {} ({})", available, model.display_name, model.name);
                }
            }
            ProviderStatus::Offline => {
                println!("   ❌ Cannot connect to provider");
                if provider_name == "ollama" {
                    println!("   💡 To start Ollama:");
                    println!("      1. Install from https://ollama.ai");
                    println!("      2. Run: ollama serve");
                    println!("      3. Pull models: ollama pull llama2");
                }
            }
            ProviderStatus::ConfigError(msg) => {
                println!("   ⚠️  Configuration issue: {}", msg);
                if provider_name == "openai" || provider_name == "anthropic" {
                    println!("   💡 Add API key to config file or environment variable");
                }
            }
            ProviderStatus::Unknown => {
                println!("   ❓ Could not determine status");
            }
        }
    }
    
    // Summary
    let online_count = health_results.values()
        .filter(|h| matches!(h.status, ProviderStatus::Online))
        .count();
        
    let total_count = health_results.len();
    
    println!("\n{}", "=".repeat(60));
    println!("📈 Summary: {}/{} providers online", online_count, total_count);
    
    if online_count == 0 {
        println!("⚠️  No providers are currently available!");
        println!("   AgentX will run in mock mode until providers are configured.");
    } else if online_count < total_count {
        println!("💡 Some providers need configuration or are offline.");
        println!("   You can still use available providers for AI interactions.");
    } else {
        println!("🎉 All providers are online and ready!");
    }
    
    // Test individual provider check
    println!("\n🔍 Testing individual provider check...");
    if let Some(default_provider) = health_results.get(&config.default_provider) {
        println!("Default provider '{}' status: {} {}", 
            config.default_provider, 
            default_provider.status.icon(), 
            default_provider.status.description()
        );
        
        if matches!(default_provider.status, ProviderStatus::Online) {
            println!("✅ Default provider is ready for AI interactions!");
        } else {
            println!("⚠️  Default provider is not available. Consider:");
            println!("   • Starting the service (for local providers)");
            println!("   • Adding API keys (for cloud providers)");
            println!("   • Switching to an available provider with 'Tab' in the UI");
        }
    }
    
    Ok(())
}