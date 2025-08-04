use agentx::providers::{
    openai_compatible::OpenAICompatibleProvider,
    CompletionRequest, LLMProvider, Message, MessageRole,
};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    println!("Testing OpenAI-compatible provider with local models...\n");
    
    // Try Ollama first
    println!("1. Testing Ollama connection...");
    let ollama = OpenAICompatibleProvider::for_ollama()?;
    
    match ollama.validate_config().await {
        Ok(_) => println!("✅ Ollama is running!"),
        Err(e) => println!("❌ Ollama not available: {}", e),
    }
    
    // List available models
    println!("\n2. Available models:");
    match ollama.capabilities().await {
        Ok(caps) => {
            for model in &caps.models {
                println!("   - {} (context: {} tokens)", model.id, model.context_window);
            }
        }
        Err(e) => println!("   Failed to get models: {}", e),
    }
    
    // Test a simple completion
    println!("\n3. Testing completion...");
    let request = CompletionRequest {
        messages: vec![
            Message {
                role: MessageRole::System,
                content: "You are a helpful AI assistant.".to_string(),
            },
            Message {
                role: MessageRole::User,
                content: "Write a haiku about Rust programming.".to_string(),
            },
        ],
        model: "llama2".to_string(), // Change to your model
        temperature: Some(0.7),
        max_tokens: Some(100),
        stream: false,
    };
    
    match ollama.complete(request).await {
        Ok(response) => {
            println!("✅ Response from {}:", response.model);
            println!("{}", response.content);
            if let Some(usage) = response.usage {
                println!("\nTokens used: {} prompt + {} completion = {} total",
                    usage.prompt_tokens, usage.completion_tokens, usage.total_tokens);
            }
        }
        Err(e) => println!("❌ Completion failed: {}", e),
    }
    
    // Test streaming
    println!("\n4. Testing streaming completion...");
    let stream_request = CompletionRequest {
        messages: vec![
            Message {
                role: MessageRole::User,
                content: "Count from 1 to 5 slowly.".to_string(),
            },
        ],
        model: "llama2".to_string(),
        temperature: Some(0.7),
        max_tokens: Some(50),
        stream: true,
    };
    
    use futures::StreamExt;
    match ollama.stream_complete(stream_request).await {
        Ok(mut stream) => {
            print!("✅ Streaming response: ");
            while let Some(chunk_result) = stream.next().await {
                match chunk_result {
                    Ok(chunk) => {
                        print!("{}", chunk.delta);
                        if chunk.finish_reason.is_some() {
                            println!("\n✅ Stream completed!");
                            break;
                        }
                    }
                    Err(e) => {
                        println!("\n❌ Stream error: {}", e);
                        break;
                    }
                }
            }
        }
        Err(e) => println!("❌ Streaming failed: {}", e),
    }
    
    Ok(())
}