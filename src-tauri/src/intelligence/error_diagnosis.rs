use anyhow::Result;
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

/// Intelligent error diagnosis and recovery system
pub struct ErrorDiagnosticEngine {
    error_classifier: ErrorClassifier,
    explanation_generator: ExplanationGenerator,
    fix_suggester: FixSuggester,
    learning_system: Arc<ErrorLearningSystem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorDiagnosis {
    pub error_type: ErrorType,
    pub severity: Severity,
    pub explanation: String,
    pub root_cause: String,
    pub suggested_fixes: Vec<Fix>,
    pub prevention_tips: Vec<String>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum ErrorType {
    CompilationError,
    RuntimeError,
    DependencyError,
    ConfigurationError,
    PermissionError,
    NetworkError,
    ResourceError,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, PartialOrd)]
pub enum Severity {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fix {
    pub description: String,
    pub commands: Vec<String>,
    pub confidence: f32,
    pub side_effects: Vec<String>,
    pub requires_confirmation: bool,
}

#[derive(Debug, Clone)]
struct ErrorPattern {
    regex: Regex,
    error_type: ErrorType,
    explanation_template: String,
    fixes: Vec<FixTemplate>,
}

#[derive(Debug, Clone)]
struct FixTemplate {
    description: String,
    command_template: String,
    confidence: f32,
    side_effects: Vec<String>,
}

pub struct ErrorClassifier {
    patterns: Vec<ErrorPattern>,
    ml_classifier: Option<Arc<dyn MLClassifier>>,
}

trait MLClassifier: Send + Sync {
    fn classify(&self, error_text: &str) -> Result<(ErrorType, f32)>;
}

impl ErrorClassifier {
    pub fn new() -> Self {
        Self {
            patterns: Self::build_patterns(),
            ml_classifier: None, // Would be initialized with actual ML model
        }
    }
    
    fn build_patterns() -> Vec<ErrorPattern> {
        vec![
            // Port in use error
            ErrorPattern {
                regex: Regex::new(r"EADDRINUSE.*:(\d+)").unwrap(),
                error_type: ErrorType::ResourceError,
                explanation_template: "Port {1} is already in use by another process".to_string(),
                fixes: vec![
                    FixTemplate {
                        description: "Kill process using the port".to_string(),
                        command_template: "lsof -ti:{1} | xargs kill -9".to_string(),
                        confidence: 0.9,
                        side_effects: vec!["Will terminate the process using this port".to_string()],
                    },
                    FixTemplate {
                        description: "Use a different port".to_string(),
                        command_template: "Change port configuration to use an available port".to_string(),
                        confidence: 0.8,
                        side_effects: vec![],
                    },
                ],
            },
            
            // Missing module error (Node.js)
            ErrorPattern {
                regex: Regex::new(r"Cannot find module '([^']+)'").unwrap(),
                error_type: ErrorType::DependencyError,
                explanation_template: "Module '{1}' is not installed or cannot be found".to_string(),
                fixes: vec![
                    FixTemplate {
                        description: "Install missing module".to_string(),
                        command_template: "npm install {1}".to_string(),
                        confidence: 0.95,
                        side_effects: vec!["Will modify package.json and package-lock.json".to_string()],
                    },
                    FixTemplate {
                        description: "Install missing module (yarn)".to_string(),
                        command_template: "yarn add {1}".to_string(),
                        confidence: 0.9,
                        side_effects: vec!["Will modify package.json and yarn.lock".to_string()],
                    },
                ],
            },
            
            // Rust compilation error
            ErrorPattern {
                regex: Regex::new(r"error\[E0(\d+)\]: (.+)").unwrap(),
                error_type: ErrorType::CompilationError,
                explanation_template: "Rust compilation error E0{1}: {2}".to_string(),
                fixes: vec![
                    FixTemplate {
                        description: "View Rust error explanation".to_string(),
                        command_template: "rustc --explain E0{1}".to_string(),
                        confidence: 0.7,
                        side_effects: vec![],
                    },
                ],
            },
            
            // Permission denied
            ErrorPattern {
                regex: Regex::new(r"Permission denied|EACCES").unwrap(),
                error_type: ErrorType::PermissionError,
                explanation_template: "Insufficient permissions to perform this operation".to_string(),
                fixes: vec![
                    FixTemplate {
                        description: "Run with elevated permissions".to_string(),
                        command_template: "sudo {original_command}".to_string(),
                        confidence: 0.8,
                        side_effects: vec!["Will run command as superuser".to_string()],
                    },
                    FixTemplate {
                        description: "Change file permissions".to_string(),
                        command_template: "chmod +x {file}".to_string(),
                        confidence: 0.7,
                        side_effects: vec!["Will modify file permissions".to_string()],
                    },
                ],
            },
            
            // Out of memory
            ErrorPattern {
                regex: Regex::new(r"out of memory|OOM|heap out of memory").unwrap(),
                error_type: ErrorType::ResourceError,
                explanation_template: "Process ran out of available memory".to_string(),
                fixes: vec![
                    FixTemplate {
                        description: "Increase Node.js memory limit".to_string(),
                        command_template: "NODE_OPTIONS='--max-old-space-size=4096' {original_command}".to_string(),
                        confidence: 0.8,
                        side_effects: vec!["Will use more system memory".to_string()],
                    },
                    FixTemplate {
                        description: "Close other applications to free memory".to_string(),
                        command_template: "Check system memory usage and close unnecessary applications".to_string(),
                        confidence: 0.6,
                        side_effects: vec![],
                    },
                ],
            },
        ]
    }
    
    pub async fn classify(&self, error_output: &str, _context: &ErrorContext) -> Result<ErrorType> {
        // Try pattern matching first
        for pattern in &self.patterns {
            if pattern.regex.is_match(error_output) {
                return Ok(pattern.error_type.clone());
            }
        }
        
        // Use ML classifier if available
        if let Some(ml) = &self.ml_classifier {
            let (error_type, confidence) = ml.classify(error_output)?;
            if confidence > 0.7 {
                return Ok(error_type);
            }
        }
        
        // Default to unknown
        Ok(ErrorType::Unknown)
    }
    
    pub fn get_pattern_for_error(&self, error_output: &str) -> Option<&ErrorPattern> {
        self.patterns.iter().find(|p| p.regex.is_match(error_output))
    }
}

#[derive(Debug, Clone)]
pub struct ErrorContext {
    pub command: String,
    pub working_directory: String,
    pub environment: HashMap<String, String>,
    pub exit_code: i32,
}

pub struct ExplanationGenerator;

impl ExplanationGenerator {
    pub fn generate(&self, error_type: &ErrorType, error_output: &str, pattern: Option<&ErrorPattern>) -> String {
        if let Some(pattern) = pattern {
            // Use pattern template with captured groups
            if let Some(captures) = pattern.regex.captures(error_output) {
                let mut explanation = pattern.explanation_template.clone();
                for i in 1..captures.len() {
                    if let Some(capture) = captures.get(i) {
                        explanation = explanation.replace(&format!("{{{}}}", i), capture.as_str());
                    }
                }
                return explanation;
            }
        }
        
        // Generic explanations
        match error_type {
            ErrorType::CompilationError => "The code failed to compile due to syntax or type errors".to_string(),
            ErrorType::RuntimeError => "The program crashed during execution".to_string(),
            ErrorType::DependencyError => "Required dependencies are missing or incompatible".to_string(),
            ErrorType::ConfigurationError => "The configuration is invalid or missing required settings".to_string(),
            ErrorType::PermissionError => "Insufficient permissions to perform the requested operation".to_string(),
            ErrorType::NetworkError => "Network connection failed or timed out".to_string(),
            ErrorType::ResourceError => "System resources (memory, disk, ports) are exhausted or unavailable".to_string(),
            ErrorType::Unknown => "An unknown error occurred".to_string(),
        }
    }
}

pub struct FixSuggester;

impl FixSuggester {
    pub fn suggest_fixes(
        &self, 
        error_type: &ErrorType, 
        error_output: &str, 
        pattern: Option<&ErrorPattern>,
        context: &ErrorContext
    ) -> Vec<Fix> {
        let mut fixes = Vec::new();
        
        if let Some(pattern) = pattern {
            // Generate fixes from pattern templates
            if let Some(captures) = pattern.regex.captures(error_output) {
                for fix_template in &pattern.fixes {
                    let mut command = fix_template.command_template.clone();
                    
                    // Replace placeholders
                    command = command.replace("{original_command}", &context.command);
                    
                    // Replace capture groups
                    for i in 1..captures.len() {
                        if let Some(capture) = captures.get(i) {
                            command = command.replace(&format!("{{{}}}", i), capture.as_str());
                        }
                    }
                    
                    fixes.push(Fix {
                        description: fix_template.description.clone(),
                        commands: vec![command],
                        confidence: fix_template.confidence,
                        side_effects: fix_template.side_effects.clone(),
                        requires_confirmation: !fix_template.side_effects.is_empty(),
                    });
                }
            }
        }
        
        // Add generic fixes based on error type
        match error_type {
            ErrorType::DependencyError => {
                fixes.push(Fix {
                    description: "Update all dependencies".to_string(),
                    commands: vec!["npm update".to_string()],
                    confidence: 0.6,
                    side_effects: vec!["May introduce breaking changes".to_string()],
                    requires_confirmation: true,
                });
            }
            ErrorType::PermissionError => {
                fixes.push(Fix {
                    description: "Check file ownership".to_string(),
                    commands: vec!["ls -la".to_string()],
                    confidence: 0.5,
                    side_effects: vec![],
                    requires_confirmation: false,
                });
            }
            _ => {}
        }
        
        // Sort by confidence
        fixes.sort_by(|a, b| b.confidence.partial_cmp(&a.confidence).unwrap());
        
        fixes
    }
}

pub struct ErrorLearningSystem {
    learned_patterns: tokio::sync::RwLock<Vec<LearnedPattern>>,
}

#[derive(Debug, Clone)]
struct LearnedPattern {
    error_signature: String,
    successful_fix: Fix,
    occurrences: u32,
    success_rate: f32,
}

impl ErrorLearningSystem {
    pub fn new() -> Self {
        Self {
            learned_patterns: tokio::sync::RwLock::new(Vec::new()),
        }
    }
    
    pub async fn record_fix_outcome(&self, error: &str, fix: &Fix, success: bool) -> Result<()> {
        let mut patterns = self.learned_patterns.write().await;
        
        // Create error signature (simplified for demo)
        let signature = self.create_signature(error);
        
        // Find or create pattern
        if let Some(pattern) = patterns.iter_mut().find(|p| p.error_signature == signature) {
            pattern.occurrences += 1;
            if success {
                pattern.success_rate = (pattern.success_rate * (pattern.occurrences - 1) as f32 + 1.0) 
                    / pattern.occurrences as f32;
            } else {
                pattern.success_rate = (pattern.success_rate * (pattern.occurrences - 1) as f32) 
                    / pattern.occurrences as f32;
            }
        } else {
            patterns.push(LearnedPattern {
                error_signature: signature,
                successful_fix: fix.clone(),
                occurrences: 1,
                success_rate: if success { 1.0 } else { 0.0 },
            });
        }
        
        Ok(())
    }
    
    pub async fn get_learned_fixes(&self, error: &str) -> Vec<Fix> {
        let patterns = self.learned_patterns.read().await;
        let signature = self.create_signature(error);
        
        patterns.iter()
            .filter(|p| p.error_signature == signature && p.success_rate > 0.7)
            .map(|p| {
                let mut fix = p.successful_fix.clone();
                fix.confidence *= p.success_rate;
                fix
            })
            .collect()
    }
    
    fn create_signature(&self, error: &str) -> String {
        // Simple signature generation - in production would use more sophisticated hashing
        let key_parts: Vec<&str> = error
            .split_whitespace()
            .filter(|word| word.len() > 4 && !word.chars().all(char::is_numeric))
            .take(5)
            .collect();
        
        key_parts.join("_")
    }
}

impl ErrorDiagnosticEngine {
    pub fn new() -> Self {
        Self {
            error_classifier: ErrorClassifier::new(),
            explanation_generator: ExplanationGenerator,
            fix_suggester: FixSuggester,
            learning_system: Arc::new(ErrorLearningSystem::new()),
        }
    }
    
    pub async fn diagnose(&self, error_output: &str, context: &ErrorContext) -> Result<ErrorDiagnosis> {
        // Classify the error
        let error_type = self.error_classifier.classify(error_output, context).await?;
        
        // Get pattern if available
        let pattern = self.error_classifier.get_pattern_for_error(error_output);
        
        // Generate explanation
        let explanation = self.explanation_generator.generate(&error_type, error_output, pattern);
        
        // Determine severity
        let severity = self.determine_severity(&error_type, context);
        
        // Get suggested fixes
        let mut suggested_fixes = self.fix_suggester.suggest_fixes(&error_type, error_output, pattern, context);
        
        // Add learned fixes
        let learned_fixes = self.learning_system.get_learned_fixes(error_output).await;
        suggested_fixes.extend(learned_fixes);
        
        // Generate prevention tips
        let prevention_tips = self.generate_prevention_tips(&error_type);
        
        Ok(ErrorDiagnosis {
            error_type,
            severity,
            explanation,
            root_cause: self.identify_root_cause(error_output, &error_type),
            suggested_fixes,
            prevention_tips,
        })
    }
    
    fn determine_severity(&self, error_type: &ErrorType, context: &ErrorContext) -> Severity {
        match error_type {
            ErrorType::PermissionError => {
                if context.command.contains("sudo") {
                    Severity::Critical
                } else {
                    Severity::Medium
                }
            }
            ErrorType::ResourceError => Severity::High,
            ErrorType::CompilationError => Severity::Medium,
            ErrorType::DependencyError => Severity::Medium,
            ErrorType::ConfigurationError => Severity::Low,
            ErrorType::NetworkError => Severity::Medium,
            ErrorType::RuntimeError => Severity::High,
            ErrorType::Unknown => Severity::Low,
        }
    }
    
    fn identify_root_cause(&self, _error_output: &str, error_type: &ErrorType) -> String {
        match error_type {
            ErrorType::DependencyError => "Missing or incompatible package versions".to_string(),
            ErrorType::PermissionError => "User lacks necessary permissions".to_string(),
            ErrorType::ResourceError => "System resources exhausted or unavailable".to_string(),
            _ => "Root cause analysis pending".to_string(),
        }
    }
    
    fn generate_prevention_tips(&self, error_type: &ErrorType) -> Vec<String> {
        match error_type {
            ErrorType::DependencyError => vec![
                "Keep dependencies up to date with regular updates".to_string(),
                "Use lock files to ensure consistent installations".to_string(),
                "Test dependency updates in a separate branch first".to_string(),
            ],
            ErrorType::ResourceError => vec![
                "Monitor system resources during development".to_string(),
                "Set resource limits for processes".to_string(),
                "Use resource pooling for efficient usage".to_string(),
            ],
            ErrorType::PermissionError => vec![
                "Run with minimal required permissions".to_string(),
                "Use proper file ownership and permissions".to_string(),
                "Avoid using sudo unless absolutely necessary".to_string(),
            ],
            _ => vec![
                "Implement proper error handling".to_string(),
                "Add comprehensive logging".to_string(),
                "Write tests to catch errors early".to_string(),
            ],
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_port_in_use_diagnosis() {
        let engine = ErrorDiagnosticEngine::new();
        let error = "Error: listen EADDRINUSE: address already in use :::3000";
        let context = ErrorContext {
            command: "npm start".to_string(),
            working_directory: "/project".to_string(),
            environment: HashMap::new(),
            exit_code: 1,
        };
        
        let diagnosis = engine.diagnose(error, &context).await.unwrap();
        
        assert_eq!(diagnosis.error_type, ErrorType::ResourceError);
        assert!(!diagnosis.suggested_fixes.is_empty());
        assert!(diagnosis.explanation.contains("Port 3000"));
    }
    
    #[tokio::test]
    async fn test_missing_module_diagnosis() {
        let engine = ErrorDiagnosticEngine::new();
        let error = "Error: Cannot find module 'express'";
        let context = ErrorContext {
            command: "node index.js".to_string(),
            working_directory: "/project".to_string(),
            environment: HashMap::new(),
            exit_code: 1,
        };
        
        let diagnosis = engine.diagnose(error, &context).await.unwrap();
        
        assert_eq!(diagnosis.error_type, ErrorType::DependencyError);
        assert!(diagnosis.suggested_fixes.iter().any(|f| f.commands[0].contains("npm install express")));
    }
}