use anyhow::Result;
use std::sync::Arc;
use tokio::sync::Mutex;
use std::collections::HashMap;

use super::{Agent, Task, TaskType, TaskStatus, TaskResult, AgentCapabilities, Artifact, ArtifactType};

pub struct Orchestrator {
    agents: HashMap<String, Arc<Mutex<Box<dyn Agent>>>>,
}

impl Orchestrator {
    pub fn new() -> Self {
        let mut orchestrator = Self {
            agents: HashMap::new(),
        };
        
        // Register default agents
        orchestrator.register_default_agents();
        orchestrator
    }
    
    fn register_default_agents(&mut self) {
        // Register mock agents for now
        let architect = Arc::new(Mutex::new(Box::new(MockArchitectAgent::new()) as Box<dyn Agent>));
        let builder = Arc::new(Mutex::new(Box::new(MockBuilderAgent::new()) as Box<dyn Agent>));
        let tester = Arc::new(Mutex::new(Box::new(MockTesterAgent::new()) as Box<dyn Agent>));
        
        self.agents.insert("architect".to_string(), architect);
        self.agents.insert("builder".to_string(), builder);
        self.agents.insert("tester".to_string(), tester);
    }
    
    pub async fn process_request(&mut self, request: &str) -> Result<String> {
        // Step 1: Decompose request into tasks
        let tasks = self.decompose_request(request).await?;
        
        // Step 2: Execute tasks
        let mut results = Vec::new();
        for task in tasks {
            let result = self.execute_task(task).await?;
            results.push(result);
        }
        
        // Step 3: Compile results
        Ok(self.compile_results(results))
    }
    
    async fn decompose_request(&self, request: &str) -> Result<Vec<Task>> {
        // Simple task decomposition for now
        let mut tasks = Vec::new();
        
        // Analyze what the user wants
        let lower_request = request.to_lowercase();
        
        if lower_request.contains("app") || lower_request.contains("build") {
            // Architecture task
            tasks.push(Task {
                id: format!("task-{}", uuid::Uuid::new_v4()),
                description: format!("Design architecture for: {}", request),
                task_type: TaskType::Architecture,
                status: TaskStatus::Pending,
                context: Some(request.to_string()),
            });
            
            // Implementation task
            tasks.push(Task {
                id: format!("task-{}", uuid::Uuid::new_v4()),
                description: format!("Implement: {}", request),
                task_type: TaskType::Implementation,
                status: TaskStatus::Pending,
                context: Some(request.to_string()),
            });
            
            // Testing task
            tasks.push(Task {
                id: format!("task-{}", uuid::Uuid::new_v4()),
                description: format!("Write tests for: {}", request),
                task_type: TaskType::Testing,
                status: TaskStatus::Pending,
                context: Some(request.to_string()),
            });
        } else {
            // Default to a single implementation task
            tasks.push(Task {
                id: format!("task-{}", uuid::Uuid::new_v4()),
                description: request.to_string(),
                task_type: TaskType::Implementation,
                status: TaskStatus::Pending,
                context: Some(request.to_string()),
            });
        }
        
        Ok(tasks)
    }
    
    async fn execute_task(&self, mut task: Task) -> Result<TaskResult> {
        // Find appropriate agent
        let agent_name = match task.task_type {
            TaskType::Architecture => "architect",
            TaskType::Implementation => "builder",
            TaskType::Testing => "tester",
            TaskType::Documentation => "builder", // Use builder for now
        };
        
        if let Some(agent_mutex) = self.agents.get(agent_name) {
            let mut agent = agent_mutex.lock().await;
            task.status = TaskStatus::InProgress;
            let result = agent.execute(task).await?;
            Ok(result)
        } else {
            Err(anyhow::anyhow!("No agent available for task type"))
        }
    }
    
    fn compile_results(&self, results: Vec<TaskResult>) -> String {
        let mut output = String::new();
        output.push_str("🎉 Project created successfully!\n\n");
        
        for result in results {
            if result.success {
                output.push_str(&format!("✅ {}\n", result.output));
                
                // Show artifacts
                for artifact in result.artifacts {
                    output.push_str(&format!("\n📄 {}\n", artifact.name));
                    if artifact.content.len() < 500 {
                        output.push_str(&format!("```\n{}\n```\n", artifact.content));
                    } else {
                        output.push_str(&format!("```\n{}...\n```\n", &artifact.content[..200]));
                    }
                }
            }
        }
        
        output
    }
}

// Mock agents for testing
struct MockArchitectAgent {
    capabilities: AgentCapabilities,
}

impl MockArchitectAgent {
    fn new() -> Self {
        Self {
            capabilities: AgentCapabilities {
                supported_tasks: vec![TaskType::Architecture],
                confidence_threshold: 0.8,
                max_context_size: 8000,
            },
        }
    }
}

#[async_trait::async_trait]
impl Agent for MockArchitectAgent {
    fn capabilities(&self) -> &AgentCapabilities {
        &self.capabilities
    }
    
    fn name(&self) -> &str {
        "Mock Architect"
    }
    
    async fn execute(&mut self, task: Task) -> Result<TaskResult> {
        // Simulate thinking
        tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;
        
        Ok(TaskResult {
            task_id: task.id,
            success: true,
            output: "Designed clean architecture with separation of concerns".to_string(),
            artifacts: vec![
                Artifact {
                    name: "architecture.md".to_string(),
                    content: "# Architecture\n\n- Frontend: React\n- Backend: REST API\n- Database: PostgreSQL".to_string(),
                    artifact_type: ArtifactType::Documentation,
                }
            ],
        })
    }
}

struct MockBuilderAgent {
    capabilities: AgentCapabilities,
}

impl MockBuilderAgent {
    fn new() -> Self {
        Self {
            capabilities: AgentCapabilities {
                supported_tasks: vec![TaskType::Implementation, TaskType::Documentation],
                confidence_threshold: 0.9,
                max_context_size: 16000,
            },
        }
    }
}

#[async_trait::async_trait]
impl Agent for MockBuilderAgent {
    fn capabilities(&self) -> &AgentCapabilities {
        &self.capabilities
    }
    
    fn name(&self) -> &str {
        "Mock Builder"
    }
    
    async fn execute(&mut self, task: Task) -> Result<TaskResult> {
        // Simulate building
        tokio::time::sleep(tokio::time::Duration::from_millis(800)).await;
        
        let code = if task.description.to_lowercase().contains("todo") {
            r#"import React, { useState } from 'react';

function TodoApp() {
  const [todos, setTodos] = useState([]);
  const [input, setInput] = useState('');

  const addTodo = () => {
    if (input.trim()) {
      setTodos([...todos, { id: Date.now(), text: input }]);
      setInput('');
    }
  };

  return (
    <div className="todo-app">
      <h1>Todo List</h1>
      <input 
        value={input} 
        onChange={(e) => setInput(e.target.value)}
        onKeyPress={(e) => e.key === 'Enter' && addTodo()}
      />
      <button onClick={addTodo}>Add</button>
      <ul>
        {todos.map(todo => (
          <li key={todo.id}>{todo.text}</li>
        ))}
      </ul>
    </div>
  );
}"#
        } else {
            r#"function main() {
    console.log("Hello from AgentX!");
}"#
        };
        
        Ok(TaskResult {
            task_id: task.id,
            success: true,
            output: "Implementation complete".to_string(),
            artifacts: vec![
                Artifact {
                    name: "app.js".to_string(),
                    content: code.to_string(),
                    artifact_type: ArtifactType::Code,
                }
            ],
        })
    }
}

struct MockTesterAgent {
    capabilities: AgentCapabilities,
}

impl MockTesterAgent {
    fn new() -> Self {
        Self {
            capabilities: AgentCapabilities {
                supported_tasks: vec![TaskType::Testing],
                confidence_threshold: 0.95,
                max_context_size: 8000,
            },
        }
    }
}

#[async_trait::async_trait]
impl Agent for MockTesterAgent {
    fn capabilities(&self) -> &AgentCapabilities {
        &self.capabilities
    }
    
    fn name(&self) -> &str {
        "Mock Tester"
    }
    
    async fn execute(&mut self, task: Task) -> Result<TaskResult> {
        // Simulate testing
        tokio::time::sleep(tokio::time::Duration::from_millis(600)).await;
        
        Ok(TaskResult {
            task_id: task.id,
            success: true,
            output: "All tests passed (3/3)".to_string(),
            artifacts: vec![
                Artifact {
                    name: "test_results.txt".to_string(),
                    content: "✓ Component renders\n✓ Can add items\n✓ Input clears after adding".to_string(),
                    artifact_type: ArtifactType::TestResults,
                }
            ],
        })
    }
}