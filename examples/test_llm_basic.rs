use agentx::providers::{*, demo::DemoProvider, openai::OpenAIProvider, anthropic::AnthropicProvider};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🧪 Testing AgentX LLM Providers");
    println!("═════════════════════════════════════\n");
    
    // Test Demo Provider (always available)
    println!("1️⃣  Testing Demo Provider:");
    test_provider(Box::new(DemoProvider::new())).await?;
    
    // Test OpenAI if API key is available
    if let Ok(api_key) = std::env::var("OPENAI_API_KEY") {
        println!("\n2️⃣  Testing OpenAI Provider:");
        let config = ProviderConfig {
            api_key: Some(api_key),
            base_url: Some("https://api.openai.com/v1".to_string()),
            timeout_secs: Some(30),
            max_retries: Some(3),
        };
        if let Ok(provider) = OpenAIProvider::new(config) {
            test_provider(Box::new(provider)).await?;
        }
    } else {
        println!("\n2️⃣  OpenAI Provider: Skipped (no API key)");
    }
    
    // Test Anthropic if API key is available
    if let Ok(api_key) = std::env::var("ANTHROPIC_API_KEY") {
        println!("\n3️⃣  Testing Anthropic Provider:");
        let config = ProviderConfig {
            api_key: Some(api_key),
            base_url: Some("https://api.anthropic.com/v1".to_string()),
            timeout_secs: Some(30),
            max_retries: Some(3),
        };
        if let Ok(provider) = AnthropicProvider::new(config) {
            test_provider(Box::new(provider)).await?;
        }
    } else {
        println!("\n3️⃣  Anthropic Provider: Skipped (no API key)");
    }
    
    // Test Ollama if available
    println!("\n4️⃣  Testing Ollama Provider:");
    match agentx::providers::openai_compatible::OpenAICompatibleProvider::for_ollama() {
        Ok(provider) => {
            match provider.validate_config().await {
                Ok(()) => test_provider(Box::new(provider)).await?,
                Err(e) => println!("   ⚠️  Ollama not running: {}", e),
            }
        }
        Err(e) => println!("   ❌ Failed to create Ollama provider: {}", e),
    }
    
    Ok(())
}

async fn test_provider(provider: Box<dyn LLMProvider>) -> Result<()> {
    println!("   Provider: {}", provider.name());
    
    // Get capabilities
    let caps = provider.capabilities().await?;
    println!("   Models: {} available", caps.models.len());
    for model in caps.models.iter().take(3) {
        println!("   - {} ({}k context)", model.name, model.context_window / 1000);
    }
    
    // Test completion
    let request = CompletionRequest {
        messages: vec![
            Message {
                role: MessageRole::System,
                content: "You are a helpful assistant. Be very concise.".to_string(),
            },
            Message {
                role: MessageRole::User,
                content: "Say hello in exactly 5 words.".to_string(),
            },
        ],
        model: caps.models.first().unwrap().id.clone(),
        temperature: Some(0.7),
        max_tokens: Some(50),
        stream: false,
    };
    
    println!("   Testing completion...");
    match provider.complete(request.clone()).await {
        Ok(response) => {
            println!("   ✅ Response: {}", response.content.trim());
            if let Some(usage) = response.usage {
                println!("   📊 Tokens: {} prompt, {} completion", 
                    usage.prompt_tokens, usage.completion_tokens);
            }
        }
        Err(e) => println!("   ❌ Error: {}", e),
    }
    
    // Test streaming
    println!("   Testing streaming...");
    let mut stream_request = request;
    stream_request.stream = true;
    
    match provider.stream_complete(stream_request).await {
        Ok(mut stream) => {
            use futures::StreamExt;
            print!("   🌊 Stream: ");
            let mut count = 0;
            while let Some(chunk) = stream.next().await {
                match chunk {
                    Ok(response) => {
                        print!("{}", response.delta);
                        count += 1;
                        if response.finish_reason.is_some() {
                            break;
                        }
                    }
                    Err(e) => {
                        println!("\n   ❌ Stream error: {}", e);
                        break;
                    }
                }
            }
            println!(" ({}chunks)", count);
        }
        Err(e) => println!("   ❌ Stream error: {}", e),
    }
    
    Ok(())
}