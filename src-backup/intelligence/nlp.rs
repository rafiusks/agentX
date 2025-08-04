use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

/// Natural Language Processing engine for converting user queries into actionable commands
pub struct NLPEngine {
    intent_classifier: IntentClassifier,
    entity_extractor: EntityExtractor,
    command_generator: CommandGenerator,
    context_manager: Arc<ContextManager>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DeveloperIntent {
    // File operations
    CreateFile { 
        path: String, 
        content: Option<String> 
    },
    EditFile { 
        path: String, 
        changes: Vec<FileChange> 
    },
    SearchCode { 
        pattern: String, 
        scope: SearchScope 
    },
    
    // Project operations
    BuildProject { 
        target: Option<String> 
    },
    RunTests { 
        filter: Option<String> 
    },
    Deploy { 
        environment: String 
    },
    
    // Git operations
    CommitChanges { 
        message: String 
    },
    CreateBranch { 
        name: String 
    },
    ReviewChanges,
    
    // Complex workflows
    RefactorCode { 
        pattern: String, 
        replacement: String 
    },
    DebugIssue { 
        description: String 
    },
    OptimizePerformance { 
        target: String 
    },
    
    // Ambiguous or unknown
    Unknown { 
        query: String 
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileChange {
    pub line_range: Option<(usize, usize)>,
    pub old_content: Option<String>,
    pub new_content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SearchScope {
    CurrentFile,
    CurrentDirectory,
    Project,
    Workspace,
}

#[derive(Debug, Clone)]
pub struct IntentCandidate {
    pub intent: DeveloperIntent,
    pub confidence: f32,
    pub entities: HashMap<String, String>,
}

#[derive(Debug, Clone)]
pub struct ShellCommand {
    pub command: String,
    pub args: Vec<String>,
    pub description: String,
}

impl ShellCommand {
    pub fn new(command: &str, args: Vec<&str>) -> Self {
        Self {
            command: command.to_string(),
            args: args.into_iter().map(|s| s.to_string()).collect(),
            description: String::new(),
        }
    }
    
    pub fn with_description(mut self, desc: &str) -> Self {
        self.description = desc.to_string();
        self
    }
}

pub struct IntentClassifier {
    patterns: Vec<IntentPattern>,
}

#[derive(Debug, Clone)]
struct IntentPattern {
    keywords: Vec<String>,
    intent_type: IntentType,
    confidence_boost: f32,
}

#[derive(Debug, Clone, PartialEq)]
enum IntentType {
    FileCreation,
    FileEdit,
    CodeSearch,
    Build,
    Test,
    Deploy,
    GitCommit,
    GitBranch,
    Refactor,
    Debug,
}

impl IntentClassifier {
    pub fn new() -> Self {
        Self {
            patterns: vec![
                IntentPattern {
                    keywords: vec!["create".into(), "new".into(), "file".into()],
                    intent_type: IntentType::FileCreation,
                    confidence_boost: 0.8,
                },
                IntentPattern {
                    keywords: vec!["edit".into(), "modify".into(), "change".into()],
                    intent_type: IntentType::FileEdit,
                    confidence_boost: 0.7,
                },
                IntentPattern {
                    keywords: vec!["search".into(), "find".into(), "grep".into()],
                    intent_type: IntentType::CodeSearch,
                    confidence_boost: 0.9,
                },
                IntentPattern {
                    keywords: vec!["build".into(), "compile".into(), "make".into()],
                    intent_type: IntentType::Build,
                    confidence_boost: 0.8,
                },
                IntentPattern {
                    keywords: vec!["test".into(), "run tests".into(), "check".into()],
                    intent_type: IntentType::Test,
                    confidence_boost: 0.85,
                },
                IntentPattern {
                    keywords: vec!["deploy".into(), "release".into(), "publish".into()],
                    intent_type: IntentType::Deploy,
                    confidence_boost: 0.9,
                },
                IntentPattern {
                    keywords: vec!["commit".into(), "save changes".into()],
                    intent_type: IntentType::GitCommit,
                    confidence_boost: 0.9,
                },
                IntentPattern {
                    keywords: vec!["branch".into(), "checkout".into()],
                    intent_type: IntentType::GitBranch,
                    confidence_boost: 0.85,
                },
            ],
        }
    }
    
    pub async fn classify(&self, query: &str) -> Vec<IntentCandidate> {
        let query_lower = query.to_lowercase();
        let mut candidates = Vec::new();
        
        // Pattern-based classification
        for pattern in &self.patterns {
            let match_count = pattern.keywords.iter()
                .filter(|keyword| query_lower.contains(keyword.as_str()))
                .count();
            
            if match_count > 0 {
                let confidence = (match_count as f32 / pattern.keywords.len() as f32) 
                    * pattern.confidence_boost;
                
                let intent = self.create_intent(&pattern.intent_type, query);
                candidates.push(IntentCandidate {
                    intent,
                    confidence,
                    entities: HashMap::new(),
                });
            }
        }
        
        // Sort by confidence
        candidates.sort_by(|a, b| b.confidence.partial_cmp(&a.confidence).unwrap());
        
        // If no matches, return unknown intent
        if candidates.is_empty() {
            candidates.push(IntentCandidate {
                intent: DeveloperIntent::Unknown { query: query.to_string() },
                confidence: 0.0,
                entities: HashMap::new(),
            });
        }
        
        candidates
    }
    
    fn create_intent(&self, intent_type: &IntentType, query: &str) -> DeveloperIntent {
        match intent_type {
            IntentType::FileCreation => DeveloperIntent::CreateFile {
                path: String::new(), // Will be filled by entity extractor
                content: None,
            },
            IntentType::FileEdit => DeveloperIntent::EditFile {
                path: String::new(),
                changes: vec![],
            },
            IntentType::CodeSearch => DeveloperIntent::SearchCode {
                pattern: String::new(),
                scope: SearchScope::Project,
            },
            IntentType::Build => DeveloperIntent::BuildProject {
                target: None,
            },
            IntentType::Test => DeveloperIntent::RunTests {
                filter: None,
            },
            IntentType::Deploy => DeveloperIntent::Deploy {
                environment: "production".to_string(),
            },
            IntentType::GitCommit => DeveloperIntent::CommitChanges {
                message: String::new(),
            },
            IntentType::GitBranch => DeveloperIntent::CreateBranch {
                name: String::new(),
            },
            IntentType::Refactor => DeveloperIntent::RefactorCode {
                pattern: String::new(),
                replacement: String::new(),
            },
            IntentType::Debug => DeveloperIntent::DebugIssue {
                description: query.to_string(),
            },
        }
    }
}

pub struct EntityExtractor;

impl EntityExtractor {
    pub fn new() -> Self {
        Self
    }
    
    pub async fn extract(&self, query: &str, intent: &mut DeveloperIntent) -> Result<()> {
        // Simple regex-based extraction for now
        // In production, this would use a proper NER model
        
        match intent {
            DeveloperIntent::CreateFile { path, .. } => {
                // Extract file path from query
                if let Some(extracted_path) = self.extract_path(query) {
                    *path = extracted_path;
                }
            }
            DeveloperIntent::RunTests { filter } => {
                // Extract test filter
                if let Some(test_pattern) = self.extract_test_pattern(query) {
                    *filter = Some(test_pattern);
                }
            }
            DeveloperIntent::CommitChanges { message } => {
                // Extract commit message
                if let Some(msg) = self.extract_quoted_string(query) {
                    *message = msg;
                }
            }
            _ => {}
        }
        
        Ok(())
    }
    
    fn extract_path(&self, query: &str) -> Option<String> {
        // Look for patterns like "file.rs" or "src/main.rs"
        let path_regex = regex::Regex::new(r"[\w\-_/]+\.\w+").ok()?;
        path_regex.find(query).map(|m| m.as_str().to_string())
    }
    
    fn extract_test_pattern(&self, query: &str) -> Option<String> {
        // Extract test patterns like "test_*" or specific test names
        if query.contains("test") {
            let parts: Vec<&str> = query.split_whitespace().collect();
            for (i, part) in parts.iter().enumerate() {
                if *part == "test" && i + 1 < parts.len() {
                    return Some(parts[i + 1].to_string());
                }
            }
        }
        None
    }
    
    fn extract_quoted_string(&self, query: &str) -> Option<String> {
        // Extract content between quotes
        let quote_regex = regex::Regex::new(r#""([^"]+)""#).ok()?;
        quote_regex.captures(query)
            .and_then(|cap| cap.get(1))
            .map(|m| m.as_str().to_string())
    }
}

pub struct CommandGenerator {
    context_manager: Arc<ContextManager>,
}

impl CommandGenerator {
    pub fn new(context_manager: Arc<ContextManager>) -> Self {
        Self { context_manager }
    }
    
    pub async fn generate(&self, intent: &DeveloperIntent) -> Result<Vec<ShellCommand>> {
        match intent {
            DeveloperIntent::CreateFile { path, content } => {
                let mut commands = vec![];
                
                // Create directory if needed
                if path.contains('/') {
                    let dir = std::path::Path::new(path)
                        .parent()
                        .ok_or_else(|| anyhow!("Invalid path"))?;
                    commands.push(
                        ShellCommand::new("mkdir", vec!["-p", dir.to_str().unwrap()])
                            .with_description("Create parent directory")
                    );
                }
                
                // Create file
                commands.push(
                    ShellCommand::new("touch", vec![path])
                        .with_description("Create empty file")
                );
                
                // Add content if provided
                if let Some(content) = content {
                    commands.push(
                        ShellCommand::new("echo", vec![content, ">", path])
                            .with_description("Add initial content")
                    );
                }
                
                Ok(commands)
            }
            
            DeveloperIntent::RunTests { filter } => {
                // Detect project type from context
                let project_type = self.context_manager.detect_project_type().await?;
                
                let command = match project_type {
                    ProjectType::Rust => {
                        let mut args = vec!["test"];
                        if let Some(f) = filter {
                            args.push(f);
                        }
                        ShellCommand::new("cargo", args)
                            .with_description("Run Rust tests")
                    }
                    ProjectType::JavaScript => {
                        let mut args = vec!["test"];
                        if let Some(f) = filter {
                            args.push("--");
                            args.push(f);
                        }
                        ShellCommand::new("npm", args)
                            .with_description("Run JavaScript tests")
                    }
                    ProjectType::Python => {
                        let args = if let Some(f) = filter {
                            vec![f.as_str()]
                        } else {
                            vec![]
                        };
                        ShellCommand::new("pytest", args)
                            .with_description("Run Python tests")
                    }
                    _ => return Err(anyhow!("Unknown project type")),
                };
                
                Ok(vec![command])
            }
            
            DeveloperIntent::BuildProject { target } => {
                let project_type = self.context_manager.detect_project_type().await?;
                
                let command = match project_type {
                    ProjectType::Rust => {
                        let mut args = vec!["build"];
                        if let Some(t) = target {
                            args.push("--target");
                            args.push(t);
                        }
                        ShellCommand::new("cargo", args)
                            .with_description("Build Rust project")
                    }
                    ProjectType::JavaScript => {
                        ShellCommand::new("npm", vec!["run", "build"])
                            .with_description("Build JavaScript project")
                    }
                    _ => return Err(anyhow!("Unknown project type")),
                };
                
                Ok(vec![command])
            }
            
            DeveloperIntent::CommitChanges { message } => {
                Ok(vec![
                    ShellCommand::new("git", vec!["add", "."])
                        .with_description("Stage all changes"),
                    ShellCommand::new("git", vec!["commit", "-m", message])
                        .with_description("Create commit"),
                ])
            }
            
            _ => Err(anyhow!("Intent not yet implemented")),
        }
    }
}

pub struct ContextManager {
    project_root: Option<std::path::PathBuf>,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ProjectType {
    Rust,
    JavaScript,
    Python,
    Go,
    Unknown,
}

impl ContextManager {
    pub fn new() -> Self {
        Self {
            project_root: None,
        }
    }
    
    pub async fn detect_project_type(&self) -> Result<ProjectType> {
        // Check for project-specific files
        let current_dir = std::env::current_dir()?;
        
        if current_dir.join("Cargo.toml").exists() {
            Ok(ProjectType::Rust)
        } else if current_dir.join("package.json").exists() {
            Ok(ProjectType::JavaScript)
        } else if current_dir.join("requirements.txt").exists() 
            || current_dir.join("setup.py").exists() 
            || current_dir.join("pyproject.toml").exists() {
            Ok(ProjectType::Python)
        } else if current_dir.join("go.mod").exists() {
            Ok(ProjectType::Go)
        } else {
            Ok(ProjectType::Unknown)
        }
    }
}

impl NLPEngine {
    pub fn new() -> Self {
        let context_manager = Arc::new(ContextManager::new());
        
        Self {
            intent_classifier: IntentClassifier::new(),
            entity_extractor: EntityExtractor::new(),
            command_generator: CommandGenerator::new(context_manager.clone()),
            context_manager,
        }
    }
    
    pub async fn process_query(&self, query: &str) -> Result<Vec<ShellCommand>> {
        // Classify intent
        let candidates = self.intent_classifier.classify(query).await;
        
        // For now, take the highest confidence candidate
        let mut best_candidate = candidates.into_iter()
            .next()
            .ok_or_else(|| anyhow!("No intent detected"))?;
        
        // Extract entities
        self.entity_extractor.extract(query, &mut best_candidate.intent).await?;
        
        // Generate commands
        let commands = self.command_generator.generate(&best_candidate.intent).await?;
        
        Ok(commands)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_intent_classification() {
        let classifier = IntentClassifier::new();
        
        let candidates = classifier.classify("create a new file called test.rs").await;
        assert!(!candidates.is_empty());
        assert!(matches!(candidates[0].intent, DeveloperIntent::CreateFile { .. }));
        
        let candidates = classifier.classify("run all tests").await;
        assert!(!candidates.is_empty());
        assert!(matches!(candidates[0].intent, DeveloperIntent::RunTests { .. }));
    }
    
    #[tokio::test]
    async fn test_entity_extraction() {
        let extractor = EntityExtractor::new();
        
        let mut intent = DeveloperIntent::CreateFile {
            path: String::new(),
            content: None,
        };
        
        extractor.extract("create a new file called src/main.rs", &mut intent).await.unwrap();
        
        if let DeveloperIntent::CreateFile { path, .. } = intent {
            assert_eq!(path, "src/main.rs");
        } else {
            panic!("Intent type changed unexpectedly");
        }
    }
}