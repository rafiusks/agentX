use super::{Agent, AgentCapabilities, Task, TaskResult, TaskType, Artifact, ArtifactType};
use crate::providers::{LLMProvider, CompletionRequest, Message, MessageRole};
use anyhow::Result;
use std::sync::Arc;

/// An agent powered by an LLM provider
pub struct LLMAgent {
    name: String,
    capabilities: AgentCapabilities,
    provider: Arc<dyn LLMProvider>,
    model: String,
    system_prompt: String,
}

impl LLMAgent {
    pub fn new(
        name: String,
        provider: Arc<dyn LLMProvider>,
        model: String,
        supported_tasks: Vec<TaskType>,
        system_prompt: String,
    ) -> Self {
        Self {
            name,
            capabilities: AgentCapabilities {
                supported_tasks,
                confidence_threshold: 0.8,
                max_context_size: 4096, // Will be updated based on model
            },
            provider,
            model,
            system_prompt,
        }
    }
    
    pub fn architect(provider: Arc<dyn LLMProvider>, model: String) -> Self {
        Self::new(
            "LLM Architect".to_string(),
            provider,
            model,
            vec![TaskType::Architecture],
            "You are an expert software architect. Design clean, scalable architectures for the given requirements. Provide clear technical decisions and rationale.".to_string(),
        )
    }
    
    pub fn builder(provider: Arc<dyn LLMProvider>, model: String) -> Self {
        Self::new(
            "LLM Builder".to_string(),
            provider,
            model,
            vec![TaskType::Implementation, TaskType::Documentation],
            "You are an expert software developer. Write clean, efficient, and well-documented code. Follow best practices and modern patterns.".to_string(),
        )
    }
    
    pub fn tester(provider: Arc<dyn LLMProvider>, model: String) -> Self {
        Self::new(
            "LLM Tester".to_string(),
            provider,
            model,
            vec![TaskType::Testing],
            "You are an expert QA engineer. Write comprehensive tests that ensure code quality and catch edge cases. Include unit tests, integration tests, and test documentation.".to_string(),
        )
    }
}

#[async_trait::async_trait]
impl Agent for LLMAgent {
    fn capabilities(&self) -> &AgentCapabilities {
        &self.capabilities
    }
    
    fn name(&self) -> &str {
        &self.name
    }
    
    async fn execute(&mut self, task: Task) -> Result<TaskResult> {
        // Build the prompt based on task type
        let user_prompt = match task.task_type {
            TaskType::Architecture => {
                format!(
                    "Design the software architecture for the following requirement:\n\n{}\n\nProvide:\n1. High-level architecture overview\n2. Technology choices with rationale\n3. Key components and their interactions\n4. Data flow and storage approach",
                    task.context.as_ref().unwrap_or(&task.description)
                )
            },
            TaskType::Implementation => {
                format!(
                    "Implement the following:\n\n{}\n\nProvide clean, working code with:\n1. Clear structure\n2. Error handling\n3. Comments where helpful\n4. Follow modern best practices",
                    task.context.as_ref().unwrap_or(&task.description)
                )
            },
            TaskType::Testing => {
                format!(
                    "Write comprehensive tests for:\n\n{}\n\nInclude:\n1. Unit tests for key functions\n2. Integration tests if applicable\n3. Edge cases\n4. Test documentation",
                    task.context.as_ref().unwrap_or(&task.description)
                )
            },
            TaskType::Documentation => {
                format!(
                    "Create documentation for:\n\n{}\n\nInclude:\n1. Overview and purpose\n2. Usage examples\n3. API documentation if applicable\n4. Setup and configuration",
                    task.context.as_ref().unwrap_or(&task.description)
                )
            },
        };
        
        // Create the completion request
        let request = CompletionRequest {
            messages: vec![
                Message {
                    role: MessageRole::System,
                    content: self.system_prompt.clone(),
                },
                Message {
                    role: MessageRole::User,
                    content: user_prompt,
                },
            ],
            model: self.model.clone(),
            temperature: Some(0.7),
            max_tokens: Some(2000),
            stream: false,
        };
        
        // Get completion from LLM
        let response = self.provider.complete(request).await?;
        
        // Parse the response and create artifacts
        let artifacts = self.create_artifacts(&task, &response.content);
        let summary = self.create_summary(&task, &response.content);
        
        Ok(TaskResult {
            task_id: task.id,
            success: true,
            output: summary,
            artifacts,
        })
    }
}

impl LLMAgent {
    fn create_artifacts(&self, task: &Task, content: &str) -> Vec<Artifact> {
        let mut artifacts = Vec::new();
        
        match task.task_type {
            TaskType::Architecture => {
                artifacts.push(Artifact {
                    name: "architecture.md".to_string(),
                    content: content.to_string(),
                    artifact_type: ArtifactType::Documentation,
                });
            },
            TaskType::Implementation => {
                // Try to extract code blocks
                let mut in_code_block = false;
                let mut code_content = String::new();
                let mut language = String::new();
                
                for line in content.lines() {
                    if line.starts_with("```") {
                        if in_code_block {
                            // End of code block
                            if !code_content.is_empty() {
                                let filename = self.guess_filename(&language, &task.description);
                                artifacts.push(Artifact {
                                    name: filename,
                                    content: code_content.clone(),
                                    artifact_type: ArtifactType::Code,
                                });
                                code_content.clear();
                            }
                            in_code_block = false;
                        } else {
                            // Start of code block
                            in_code_block = true;
                            language = line.trim_start_matches("```").to_string();
                        }
                    } else if in_code_block {
                        code_content.push_str(line);
                        code_content.push('\n');
                    }
                }
                
                // If no code blocks found, treat entire content as code
                if artifacts.is_empty() {
                    artifacts.push(Artifact {
                        name: "implementation.txt".to_string(),
                        content: content.to_string(),
                        artifact_type: ArtifactType::Code,
                    });
                }
            },
            TaskType::Testing => {
                artifacts.push(Artifact {
                    name: "tests.txt".to_string(),
                    content: content.to_string(),
                    artifact_type: ArtifactType::TestResults,
                });
            },
            TaskType::Documentation => {
                artifacts.push(Artifact {
                    name: "documentation.md".to_string(),
                    content: content.to_string(),
                    artifact_type: ArtifactType::Documentation,
                });
            },
        }
        
        artifacts
    }
    
    fn create_summary(&self, task: &Task, content: &str) -> String {
        // Create a brief summary of what was accomplished
        let first_line = content.lines().next().unwrap_or("Task completed");
        
        match task.task_type {
            TaskType::Architecture => format!("✅ Architecture designed: {}", first_line),
            TaskType::Implementation => format!("✅ Implementation complete: {}", first_line),
            TaskType::Testing => format!("✅ Tests created: {}", first_line),
            TaskType::Documentation => format!("✅ Documentation written: {}", first_line),
        }
    }
    
    fn guess_filename(&self, language: &str, description: &str) -> String {
        let extension = match language.to_lowercase().as_str() {
            "javascript" | "js" => "js",
            "typescript" | "ts" => "ts",
            "python" | "py" => "py",
            "rust" | "rs" => "rs",
            "go" => "go",
            "java" => "java",
            "cpp" | "c++" => "cpp",
            "c" => "c",
            "html" => "html",
            "css" => "css",
            "json" => "json",
            "yaml" | "yml" => "yaml",
            _ => "txt",
        };
        
        // Try to extract a meaningful name from the description
        let name = if description.to_lowercase().contains("todo") {
            "todo"
        } else if description.to_lowercase().contains("app") {
            "app"
        } else if description.to_lowercase().contains("main") {
            "main"
        } else {
            "code"
        };
        
        format!("{}.{}", name, extension)
    }
}