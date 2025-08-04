use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, VecDeque};
use std::time::{Duration, Instant};
use std::sync::Arc;

/// Intelligent command suggestion engine
pub struct CommandSuggestionEngine {
    history_analyzer: HistoryAnalyzer,
    context_detector: ContextDetector,
    pattern_matcher: PatternMatcher,
    workflow_predictor: WorkflowPredictor,
    suggestion_cache: Arc<SuggestionCache>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Suggestion {
    pub command: String,
    pub confidence: f32,
    pub explanation: String,
    pub source: SuggestionSource,
    pub next_commands: Option<Vec<String>>,
    pub keyboard_shortcut: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SuggestionSource {
    History { 
        frequency: u32, 
        recency: Duration 
    },
    ProjectContext { 
        relevance: f32 
    },
    Pattern { 
        similar_tasks: Vec<String> 
    },
    Workflow { 
        step: u32, 
        total_steps: u32 
    },
}

pub struct HistoryAnalyzer {
    command_history: VecDeque<TimestampedCommand>,
    frequency_map: HashMap<String, u32>,
    max_history_size: usize,
}

#[derive(Debug, Clone)]
struct TimestampedCommand {
    command: String,
    timestamp: Instant,
    success: bool,
    execution_time: Duration,
}

impl HistoryAnalyzer {
    pub fn new(max_history_size: usize) -> Self {
        Self {
            command_history: VecDeque::with_capacity(max_history_size),
            frequency_map: HashMap::new(),
            max_history_size,
        }
    }
    
    pub fn add_command(&mut self, command: String, success: bool, execution_time: Duration) {
        // Update frequency map
        *self.frequency_map.entry(command.clone()).or_insert(0) += 1;
        
        // Add to history
        let timestamped = TimestampedCommand {
            command,
            timestamp: Instant::now(),
            success,
            execution_time,
        };
        
        self.command_history.push_back(timestamped);
        
        // Maintain max size
        if self.command_history.len() > self.max_history_size {
            if let Some(old) = self.command_history.pop_front() {
                if let Some(freq) = self.frequency_map.get_mut(&old.command) {
                    *freq = freq.saturating_sub(1);
                }
            }
        }
    }
    
    pub async fn suggest(&self, partial_input: &str) -> Vec<Suggestion> {
        let mut suggestions = Vec::new();
        let now = Instant::now();
        
        // Find matching commands from history
        for cmd in self.command_history.iter().rev() {
            if cmd.command.starts_with(partial_input) && cmd.success {
                let recency = now.duration_since(cmd.timestamp);
                let frequency = self.frequency_map.get(&cmd.command).copied().unwrap_or(0);
                
                // Calculate confidence based on frequency and recency
                let recency_score = 1.0 / (1.0 + recency.as_secs() as f32 / 3600.0); // Decay over hours
                let frequency_score = (frequency as f32).log2() / 10.0;
                let confidence = (recency_score * 0.6 + frequency_score * 0.4).min(1.0);
                
                suggestions.push(Suggestion {
                    command: cmd.command.clone(),
                    confidence,
                    explanation: format!("Used {} times, last used {:?} ago", frequency, recency),
                    source: SuggestionSource::History { frequency, recency },
                    next_commands: None,
                    keyboard_shortcut: None,
                });
            }
        }
        
        // Sort by confidence and deduplicate
        suggestions.sort_by(|a, b| b.confidence.partial_cmp(&a.confidence).unwrap());
        suggestions.dedup_by(|a, b| a.command == b.command);
        suggestions.truncate(5);
        
        suggestions
    }
}

pub struct ContextDetector {
    project_analyzer: ProjectAnalyzer,
    git_analyzer: GitAnalyzer,
}

#[derive(Debug, Clone)]
pub struct ProjectContext {
    pub project_type: ProjectType,
    pub git_status: GitStatus,
    pub recent_files: Vec<String>,
    pub open_ports: Vec<u16>,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ProjectType {
    Rust,
    JavaScript,
    Python,
    Go,
    Unknown,
}

#[derive(Debug, Clone)]
pub enum GitStatus {
    Clean,
    HasUncommittedChanges,
    BehindRemote(u32),
    AheadRemote(u32),
    Diverged { ahead: u32, behind: u32 },
}

pub struct ProjectAnalyzer;

impl ProjectAnalyzer {
    pub async fn analyze(&self) -> Result<ProjectType> {
        let current_dir = std::env::current_dir()?;
        
        if current_dir.join("Cargo.toml").exists() {
            Ok(ProjectType::Rust)
        } else if current_dir.join("package.json").exists() {
            Ok(ProjectType::JavaScript)
        } else if current_dir.join("requirements.txt").exists() 
            || current_dir.join("setup.py").exists() {
            Ok(ProjectType::Python)
        } else if current_dir.join("go.mod").exists() {
            Ok(ProjectType::Go)
        } else {
            Ok(ProjectType::Unknown)
        }
    }
}

pub struct GitAnalyzer;

impl GitAnalyzer {
    pub async fn get_status(&self) -> Result<GitStatus> {
        // This would actually run git commands
        // For now, return mock status
        Ok(GitStatus::HasUncommittedChanges)
    }
}

impl ContextDetector {
    pub fn new() -> Self {
        Self {
            project_analyzer: ProjectAnalyzer,
            git_analyzer: GitAnalyzer,
        }
    }
    
    pub async fn suggest(&self, input: &str, _context: &ProjectContext) -> Vec<Suggestion> {
        let mut suggestions = vec![];
        
        // Git-aware suggestions
        if let Ok(git_status) = self.git_analyzer.get_status().await {
            match git_status {
                GitStatus::HasUncommittedChanges => {
                    if input.is_empty() || "git commit".starts_with(input) {
                        suggestions.push(Suggestion {
                            command: "git add . && git commit -m \"\"".to_string(),
                            confidence: 0.9,
                            explanation: "You have uncommitted changes".to_string(),
                            source: SuggestionSource::ProjectContext { relevance: 0.9 },
                            next_commands: Some(vec!["git push".to_string()]),
                            keyboard_shortcut: Some("Ctrl+G C".to_string()),
                        });
                    }
                }
                GitStatus::BehindRemote(n) => {
                    if input.is_empty() || "git pull".starts_with(input) {
                        suggestions.push(Suggestion {
                            command: "git pull".to_string(),
                            confidence: 0.95,
                            explanation: format!("Your branch is {} commits behind", n),
                            source: SuggestionSource::ProjectContext { relevance: 0.95 },
                            next_commands: None,
                            keyboard_shortcut: Some("Ctrl+G P".to_string()),
                        });
                    }
                }
                _ => {}
            }
        }
        
        // Project-specific suggestions
        if let Ok(project_type) = self.project_analyzer.analyze().await {
            match project_type {
                ProjectType::Rust => {
                    if input.starts_with("test") || input.is_empty() {
                        suggestions.extend(vec![
                            Suggestion {
                                command: "cargo test".to_string(),
                                confidence: 0.8,
                                explanation: "Run all tests".to_string(),
                                source: SuggestionSource::ProjectContext { relevance: 0.8 },
                                next_commands: Some(vec!["cargo test -- --nocapture".to_string()]),
                                keyboard_shortcut: Some("Ctrl+T".to_string()),
                            },
                            Suggestion {
                                command: "cargo test --lib".to_string(),
                                confidence: 0.7,
                                explanation: "Run library tests only".to_string(),
                                source: SuggestionSource::ProjectContext { relevance: 0.7 },
                                next_commands: None,
                                keyboard_shortcut: None,
                            },
                        ]);
                    }
                    
                    if input.starts_with("build") || input.is_empty() {
                        suggestions.push(Suggestion {
                            command: "cargo build --release".to_string(),
                            confidence: 0.75,
                            explanation: "Build optimized binary".to_string(),
                            source: SuggestionSource::ProjectContext { relevance: 0.75 },
                            next_commands: Some(vec!["cargo run --release".to_string()]),
                            keyboard_shortcut: Some("Ctrl+B".to_string()),
                        });
                    }
                }
                ProjectType::JavaScript => {
                    if input.starts_with("test") || input.is_empty() {
                        suggestions.push(Suggestion {
                            command: "npm test".to_string(),
                            confidence: 0.8,
                            explanation: "Run test suite".to_string(),
                            source: SuggestionSource::ProjectContext { relevance: 0.8 },
                            next_commands: Some(vec!["npm run coverage".to_string()]),
                            keyboard_shortcut: Some("Ctrl+T".to_string()),
                        });
                    }
                    
                    if input.starts_with("dev") || input.is_empty() {
                        suggestions.push(Suggestion {
                            command: "npm run dev".to_string(),
                            confidence: 0.85,
                            explanation: "Start development server".to_string(),
                            source: SuggestionSource::ProjectContext { relevance: 0.85 },
                            next_commands: None,
                            keyboard_shortcut: Some("Ctrl+D".to_string()),
                        });
                    }
                }
                _ => {}
            }
        }
        
        suggestions
    }
}

pub struct PatternMatcher {
    patterns: Vec<CommandPattern>,
}

#[derive(Debug, Clone)]
struct CommandPattern {
    pattern: String,
    category: String,
    variations: Vec<String>,
}

impl PatternMatcher {
    pub fn new() -> Self {
        Self {
            patterns: vec![
                CommandPattern {
                    pattern: "find files".to_string(),
                    category: "search".to_string(),
                    variations: vec![
                        "find . -name \"*.rs\"".to_string(),
                        "fd -e rs".to_string(),
                        "rg -l pattern".to_string(),
                    ],
                },
                CommandPattern {
                    pattern: "search code".to_string(),
                    category: "search".to_string(),
                    variations: vec![
                        "rg \"TODO\"".to_string(),
                        "grep -r \"pattern\" .".to_string(),
                        "ag \"pattern\"".to_string(),
                    ],
                },
                CommandPattern {
                    pattern: "list processes".to_string(),
                    category: "system".to_string(),
                    variations: vec![
                        "ps aux | grep".to_string(),
                        "top".to_string(),
                        "htop".to_string(),
                    ],
                },
            ],
        }
    }
    
    pub async fn suggest(&self, partial_input: &str) -> Vec<Suggestion> {
        let mut suggestions = vec![];
        
        for pattern in &self.patterns {
            if pattern.pattern.contains(partial_input) || partial_input.contains(&pattern.category) {
                for (i, variation) in pattern.variations.iter().enumerate() {
                    if variation.starts_with(partial_input) || partial_input.is_empty() {
                        suggestions.push(Suggestion {
                            command: variation.clone(),
                            confidence: 0.7 - (i as f32 * 0.1),
                            explanation: format!("Common {} command", pattern.category),
                            source: SuggestionSource::Pattern {
                                similar_tasks: pattern.variations.clone(),
                            },
                            next_commands: None,
                            keyboard_shortcut: None,
                        });
                    }
                }
            }
        }
        
        suggestions
    }
}

pub struct WorkflowPredictor {
    workflows: Vec<Workflow>,
}

#[derive(Debug, Clone)]
struct Workflow {
    name: String,
    steps: Vec<String>,
    trigger_patterns: Vec<String>,
}

impl WorkflowPredictor {
    pub fn new() -> Self {
        Self {
            workflows: vec![
                Workflow {
                    name: "Git feature branch".to_string(),
                    steps: vec![
                        "git checkout -b feature/".to_string(),
                        "git add .".to_string(),
                        "git commit -m \"\"".to_string(),
                        "git push -u origin HEAD".to_string(),
                        "gh pr create".to_string(),
                    ],
                    trigger_patterns: vec!["git checkout -b".to_string()],
                },
                Workflow {
                    name: "NPM project setup".to_string(),
                    steps: vec![
                        "npm init -y".to_string(),
                        "npm install".to_string(),
                        "npm install --save-dev typescript @types/node".to_string(),
                        "npx tsc --init".to_string(),
                    ],
                    trigger_patterns: vec!["npm init".to_string()],
                },
                Workflow {
                    name: "Docker build and run".to_string(),
                    steps: vec![
                        "docker build -t app .".to_string(),
                        "docker run -p 3000:3000 app".to_string(),
                        "docker logs -f".to_string(),
                    ],
                    trigger_patterns: vec!["docker build".to_string()],
                },
            ],
        }
    }
    
    pub async fn predict_next(&self, command: &str) -> Option<Vec<String>> {
        // Find matching workflow
        for workflow in &self.workflows {
            // Check if command matches a trigger
            if workflow.trigger_patterns.iter().any(|t| command.starts_with(t)) {
                return Some(workflow.steps[1..].to_vec());
            }
            
            // Check if command is part of a workflow
            if let Some(step_index) = workflow.steps.iter().position(|s| s.starts_with(command)) {
                if step_index + 1 < workflow.steps.len() {
                    return Some(workflow.steps[step_index + 1..].to_vec());
                }
            }
        }
        
        None
    }
}

pub struct SuggestionCache {
    cache: tokio::sync::RwLock<HashMap<String, (Vec<Suggestion>, Instant)>>,
    ttl: Duration,
}

impl SuggestionCache {
    pub fn new(ttl: Duration) -> Self {
        Self {
            cache: tokio::sync::RwLock::new(HashMap::new()),
            ttl,
        }
    }
    
    pub async fn get(&self, key: &str) -> Option<Vec<Suggestion>> {
        let cache = self.cache.read().await;
        if let Some((suggestions, timestamp)) = cache.get(key) {
            if timestamp.elapsed() < self.ttl {
                return Some(suggestions.clone());
            }
        }
        None
    }
    
    pub async fn set(&self, key: String, suggestions: Vec<Suggestion>) {
        let mut cache = self.cache.write().await;
        cache.insert(key, (suggestions, Instant::now()));
        
        // Clean up old entries
        cache.retain(|_, (_, timestamp)| timestamp.elapsed() < self.ttl * 2);
    }
}

impl CommandSuggestionEngine {
    pub fn new() -> Self {
        Self {
            history_analyzer: HistoryAnalyzer::new(1000),
            context_detector: ContextDetector::new(),
            pattern_matcher: PatternMatcher::new(),
            workflow_predictor: WorkflowPredictor::new(),
            suggestion_cache: Arc::new(SuggestionCache::new(Duration::from_secs(60))),
        }
    }
    
    pub async fn suggest(&self, partial_input: &str) -> Vec<Suggestion> {
        // Check cache first
        if let Some(cached) = self.suggestion_cache.get(partial_input).await {
            return cached;
        }
        
        // Get current context
        let context = ProjectContext {
            project_type: self.context_detector.project_analyzer.analyze().await.unwrap_or(ProjectType::Unknown),
            git_status: self.context_detector.git_analyzer.get_status().await.unwrap_or(GitStatus::Clean),
            recent_files: vec![],
            open_ports: vec![],
        };
        
        // Gather suggestions from all sources in parallel
        let (history_suggestions, context_suggestions, pattern_suggestions) = tokio::join!(
            self.history_analyzer.suggest(partial_input),
            self.context_detector.suggest(partial_input, &context),
            self.pattern_matcher.suggest(partial_input)
        );
        
        // Combine all suggestions
        let mut all_suggestions = vec![];
        all_suggestions.extend(history_suggestions);
        all_suggestions.extend(context_suggestions);
        all_suggestions.extend(pattern_suggestions);
        
        // Add workflow predictions
        for suggestion in &mut all_suggestions {
            if let Some(next_commands) = self.workflow_predictor.predict_next(&suggestion.command).await {
                suggestion.next_commands = Some(next_commands);
            }
        }
        
        // Sort by confidence and deduplicate
        all_suggestions.sort_by(|a, b| b.confidence.partial_cmp(&a.confidence).unwrap());
        all_suggestions.dedup_by(|a, b| a.command == b.command);
        all_suggestions.truncate(10);
        
        // Cache results
        self.suggestion_cache.set(partial_input.to_string(), all_suggestions.clone()).await;
        
        all_suggestions
    }
    
    pub fn record_command(&mut self, command: String, success: bool, execution_time: Duration) {
        self.history_analyzer.add_command(command, success, execution_time);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_history_suggestions() {
        let mut analyzer = HistoryAnalyzer::new(100);
        
        analyzer.add_command("cargo test".to_string(), true, Duration::from_secs(1));
        analyzer.add_command("cargo test".to_string(), true, Duration::from_secs(1));
        analyzer.add_command("cargo build".to_string(), true, Duration::from_secs(2));
        
        let suggestions = analyzer.suggest("cargo").await;
        assert!(!suggestions.is_empty());
        assert_eq!(suggestions[0].command, "cargo test");
    }
    
    #[tokio::test]
    async fn test_workflow_prediction() {
        let predictor = WorkflowPredictor::new();
        
        let next = predictor.predict_next("git checkout -b feature/test").await;
        assert!(next.is_some());
        assert!(next.unwrap()[0].starts_with("git add"));
    }
}