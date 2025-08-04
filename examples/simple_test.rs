use agentx::agents::{Task, TaskType, TaskStatus};

fn main() {
    println!("AgentX - AI IDE Test");
    println!("====================\n");
    
    // Simulate creating a task
    let task = Task {
        id: "task-001".to_string(),
        description: "Build a todo app".to_string(),
        task_type: TaskType::Implementation,
        status: TaskStatus::Pending,
        context: Some("User wants a simple todo app with add/remove functionality".to_string()),
    };
    
    println!("✨ Created task: {}", task.description);
    println!("📋 Type: {:?}", task.task_type);
    println!("🎯 Status: {:?}", task.status);
    
    println!("\n🤖 Agent would now:");
    println!("  1. Analyze the request");
    println!("  2. Create a project structure");
    println!("  3. Generate the code");
    println!("  4. Run tests");
    println!("  5. Present the result");
    
    println!("\n✅ AgentX is ready to build!");
}