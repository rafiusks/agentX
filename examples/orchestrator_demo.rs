use agentx::agents::orchestrator::Orchestrator;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    println!("AgentX Orchestrator Demo");
    println!("========================\n");
    
    let mut orchestrator = Orchestrator::new();
    
    // Test different requests
    let requests = vec![
        "Build a todo app",
        "Create a REST API",
        "Write documentation for my project",
    ];
    
    for request in requests {
        println!("📝 Request: {}", request);
        println!("⏳ Processing...\n");
        
        match orchestrator.process_request(request).await {
            Ok(result) => {
                println!("{}", result);
                println!("\n{}\n", "=".repeat(60));
            }
            Err(e) => {
                println!("❌ Error: {}", e);
            }
        }
    }
    
    Ok(())
}