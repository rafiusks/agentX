pub mod orchestrator;
pub mod llm_agent;

use anyhow::Result;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Task {
    pub id: String,
    pub description: String,
    pub task_type: TaskType,
    pub status: TaskStatus,
    pub context: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TaskType {
    Architecture,
    Implementation,
    Testing,
    Documentation,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TaskStatus {
    Pending,
    InProgress,
    Completed,
    Failed,
}

#[derive(Debug, Clone)]
pub struct AgentCapabilities {
    pub supported_tasks: Vec<TaskType>,
    pub confidence_threshold: f32,
    pub max_context_size: usize,
}

#[async_trait]
pub trait Agent: Send + Sync {
    fn capabilities(&self) -> &AgentCapabilities;
    fn name(&self) -> &str;
    
    async fn can_handle(&self, task: &Task) -> bool {
        self.capabilities().supported_tasks.contains(&task.task_type)
    }
    
    async fn execute(&mut self, task: Task) -> Result<TaskResult>;
}

#[derive(Debug, Clone)]
pub struct TaskResult {
    pub task_id: String,
    pub success: bool,
    pub output: String,
    pub artifacts: Vec<Artifact>,
}

#[derive(Debug, Clone)]
pub struct Artifact {
    pub name: String,
    pub content: String,
    pub artifact_type: ArtifactType,
}

#[derive(Debug, Clone)]
pub enum ArtifactType {
    Code,
    Documentation,
    Diagram,
    TestResults,
}