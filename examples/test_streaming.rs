use agentx::providers::{
    openai_compatible::OpenAICompatibleProvider,
    CompletionRequest, LLMProvider, Message, MessageRole,
};
use anyhow::Result;
use futures::StreamExt;

#[tokio::main]
async fn main() -> Result<()> {
    println!("Testing AgentX streaming functionality...\n");
    
    // Create provider
    let provider = OpenAICompatibleProvider::for_ollama()?;
    
    // Test if Ollama is running
    println!("Checking if Ollama is available...");
    match provider.validate_config().await {
        Ok(_) => println!("✅ Ollama is running!\n"),
        Err(e) => {
            println!("❌ Ollama is not available: {}", e);
            println!("\n📝 To test streaming:");
            println!("   1. Install Ollama from https://ollama.ai");
            println!("   2. Run: ollama serve");
            println!("   3. Pull a model: ollama pull llama2");
            return Ok(());
        }
    }
    
    // Test streaming with a prompt that should generate a longer response
    let prompt = "Write a short story about a robot learning to paint. Include dialogue and descriptive passages.";
    
    println!("👤 User: {}\n", prompt);
    println!("🤖 AgentX (streaming):");
    println!("{}", "=".repeat(50));
    
    let request = CompletionRequest {
        messages: vec![
            Message {
                role: MessageRole::System,
                content: "You are AgentX, a creative AI assistant. Write engaging stories with vivid descriptions.".to_string(),
            },
            Message {
                role: MessageRole::User,
                content: prompt.to_string(),
            },
        ],
        model: "llama2".to_string(),
        temperature: Some(0.8),
        max_tokens: Some(1000),
        stream: true,
    };
    
    // Stream the response
    let mut stream = provider.stream_complete(request).await?;
    let mut full_response = String::new();
    let mut token_count = 0;
    
    while let Some(chunk_result) = stream.next().await {
        match chunk_result {
            Ok(chunk) => {
                if !chunk.delta.is_empty() {
                    print!("{}", chunk.delta);
                    use std::io::{self, Write};
                    io::stdout().flush()?;
                    full_response.push_str(&chunk.delta);
                    token_count += 1;
                }
                
                if chunk.finish_reason.is_some() {
                    println!("\n");
                    println!("{}", "=".repeat(50));
                    println!("✅ Streaming complete!");
                    println!("📊 Streamed {} chunks", token_count);
                    println!("📝 Total response length: {} characters", full_response.len());
                    break;
                }
            }
            Err(e) => {
                println!("\n❌ Stream error: {}", e);
                break;
            }
        }
    }
    
    // Test non-streaming for comparison
    println!("\n\nTesting non-streaming mode for comparison...");
    let request_non_stream = CompletionRequest {
        messages: vec![
            Message {
                role: MessageRole::System,
                content: "You are AgentX. Provide a brief summary.".to_string(),
            },
            Message {
                role: MessageRole::User,
                content: "Summarize the benefits of streaming responses in one sentence.".to_string(),
            },
        ],
        model: "llama2".to_string(),
        temperature: Some(0.7),
        max_tokens: Some(100),
        stream: false,
    };
    
    match provider.complete(request_non_stream).await {
        Ok(response) => {
            println!("🤖 AgentX (non-streaming):");
            println!("{}", response.content);
            
            if let Some(usage) = response.usage {
                println!("\n[Tokens: {} prompt + {} completion = {} total]",
                    usage.prompt_tokens, usage.completion_tokens, usage.total_tokens);
            }
        }
        Err(e) => {
            println!("❌ Error: {}", e);
        }
    }
    
    Ok(())
}