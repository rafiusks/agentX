pub mod context;
pub mod error_diagnosis;
pub mod nlp;
pub mod suggestions;

use crate::ui::LayerType;
use anyhow::Result;
use std::sync::Arc;
use tokio::sync::RwLock;

pub use context::{ContextManagementSystem, DeveloperContext};
pub use error_diagnosis::{ErrorDiagnosticEngine, ErrorDiagnosis};
pub use nlp::{NLPEngine, DeveloperIntent};
pub use suggestions::{CommandSuggestionEngine, Suggestion};

/// Main AI intelligence system that coordinates all AI features
pub struct AISystem {
    nlp_engine: Arc<NLPEngine>,
    error_engine: Arc<ErrorDiagnosticEngine>,
    suggestion_engine: Arc<RwLock<CommandSuggestionEngine>>,
    context_system: Arc<ContextManagementSystem>,
    user_context: Arc<RwLock<UserContext>>,
}

impl AISystem {
    pub fn new() -> Self {
        Self {
            nlp_engine: Arc::new(NLPEngine::new()),
            error_engine: Arc::new(ErrorDiagnosticEngine::new()),
            suggestion_engine: Arc::new(RwLock::new(CommandSuggestionEngine::new())),
            context_system: Arc::new(ContextManagementSystem::new()),
            user_context: Arc::new(RwLock::new(UserContext::new())),
        }
    }
    
    /// Process natural language query and return executable commands
    pub async fn process_query(&self, query: &str) -> Result<Vec<nlp::ShellCommand>> {
        self.nlp_engine.process_query(query).await
    }
    
    /// Diagnose an error and suggest fixes
    pub async fn diagnose_error(&self, error_output: &str, context: &error_diagnosis::ErrorContext) -> Result<ErrorDiagnosis> {
        self.error_engine.diagnose(error_output, context).await
    }
    
    /// Get command suggestions based on partial input
    pub async fn get_suggestions(&self, partial_input: &str) -> Vec<Suggestion> {
        let engine = self.suggestion_engine.read().await;
        engine.suggest(partial_input).await
    }
    
    /// Record command execution for learning
    pub async fn record_command(&self, command: String, success: bool, execution_time: std::time::Duration) {
        let mut engine = self.suggestion_engine.write().await;
        engine.record_command(command, success, execution_time);
    }
    
    /// Build context for current directory
    pub async fn build_context(&self, path: &std::path::Path) -> Result<DeveloperContext> {
        self.context_system.build_context(path).await
    }
    
    /// Get current user expertise level
    pub async fn get_user_expertise(&self) -> f32 {
        let user_ctx = self.user_context.read().await;
        user_ctx.expertise_level
    }
    
    /// Record user interaction for adaptive UI
    pub async fn record_interaction(&self) {
        let mut user_ctx = self.user_context.write().await;
        user_ctx.record_interaction();
    }
    
    /// Record UI layer transition
    pub async fn record_layer_transition(&self, new_layer: LayerType) {
        let mut user_ctx = self.user_context.write().await;
        user_ctx.record_transition(new_layer);
    }
}

pub struct UserContext {
    pub interaction_count: usize,
    pub current_layer: LayerType,
    pub expertise_level: f32,
}

impl UserContext {
    pub fn new() -> Self {
        Self {
            interaction_count: 0,
            current_layer: LayerType::Simple,
            expertise_level: 0.0,
        }
    }

    pub fn record_interaction(&mut self) {
        self.interaction_count += 1;
        self.update_expertise();
    }

    pub fn record_transition(&mut self, new_layer: LayerType) {
        self.current_layer = new_layer;
        self.update_expertise();
    }

    fn update_expertise(&mut self) {
        // Progressive expertise calculation
        // 0-10 interactions: Beginner (0.0-0.2)
        // 10-50 interactions: Intermediate (0.2-0.5)
        // 50-100 interactions: Advanced (0.5-0.8)
        // 100+ interactions: Expert (0.8-1.0)
        
        self.expertise_level = match self.interaction_count {
            0..=10 => (self.interaction_count as f32 / 10.0) * 0.2,
            11..=50 => 0.2 + ((self.interaction_count - 10) as f32 / 40.0) * 0.3,
            51..=100 => 0.5 + ((self.interaction_count - 50) as f32 / 50.0) * 0.3,
            _ => 0.8 + ((self.interaction_count - 100) as f32 / 100.0).min(0.2),
        };
        
        // Also consider layer usage
        match self.current_layer {
            LayerType::Simple => {}
            LayerType::Mission => self.expertise_level = self.expertise_level.max(0.3),
            LayerType::Pro => self.expertise_level = self.expertise_level.max(0.6),
        }
    }
}