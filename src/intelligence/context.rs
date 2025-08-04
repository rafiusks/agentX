use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, VecDeque};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use uuid::Uuid;

/// Smart context management system for understanding developer environment
pub struct ContextManagementSystem {
    // Detection components
    directory_analyzer: DirectoryAnalyzer,
    git_analyzer: GitAnalyzer,
    environment_detector: EnvironmentDetector,
    dependency_scanner: DependencyScanner,
    
    // Storage
    context_store: Arc<ContextStore>,
    session_memory: Arc<SessionMemory>,
    
    // Privacy
    privacy_filter: PrivacyFilter,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeveloperContext {
    // Project information
    pub project_root: PathBuf,
    pub project_name: String,
    pub project_type: ProjectType,
    pub languages: Vec<Language>,
    pub frameworks: Vec<Framework>,
    
    // Git state
    pub git_info: Option<GitInfo>,
    
    // Environment
    pub environment_vars: HashMap<String, String>,
    pub installed_tools: Vec<Tool>,
    pub running_services: Vec<Service>,
    
    // Multi-repo awareness
    pub workspace_root: Option<PathBuf>,
    pub related_repos: Vec<Repository>,
    pub cross_repo_dependencies: Vec<Dependency>,
    
    // Session info
    pub session_id: Uuid,
    pub session_start: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GitInfo {
    pub current_branch: String,
    pub remote_url: Option<String>,
    pub uncommitted_changes: Vec<FileChange>,
    pub recent_commits: Vec<Commit>,
    pub stash_count: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileChange {
    pub path: String,
    pub change_type: ChangeType,
    pub lines_added: usize,
    pub lines_removed: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChangeType {
    Added,
    Modified,
    Deleted,
    Renamed { from: String },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Commit {
    pub hash: String,
    pub author: String,
    pub message: String,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Default)]
pub enum ProjectType {
    #[default]
    Rust,
    JavaScript,
    TypeScript,
    Python,
    Go,
    Java,
    CSharp,
    Ruby,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub enum Language {
    Rust,
    JavaScript,
    TypeScript,
    Python,
    Go,
    Java,
    CSharp,
    Ruby,
    Shell,
    YAML,
    JSON,
    Markdown,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub enum Framework {
    // Web frameworks
    React,
    Vue,
    Angular,
    NextJS,
    Express,
    Django,
    Flask,
    Rails,
    
    // Backend frameworks
    Actix,
    Rocket,
    Axum,
    SpringBoot,
    
    // Testing frameworks
    Jest,
    Mocha,
    PyTest,
    RSpec,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tool {
    pub name: String,
    pub version: String,
    pub path: PathBuf,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Service {
    pub name: String,
    pub port: Option<u16>,
    pub pid: u32,
    pub status: ServiceStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceStatus {
    Running,
    Stopped,
    Failed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Repository {
    pub name: String,
    pub path: PathBuf,
    pub remote_url: Option<String>,
    pub main_language: Language,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Dependency {
    pub from_repo: String,
    pub to_repo: String,
    pub dependency_type: DependencyType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DependencyType {
    Library,
    Service,
    SharedCode,
}

pub struct DirectoryAnalyzer;

impl DirectoryAnalyzer {
    pub async fn analyze(&self, path: &Path) -> Result<ProjectAnalysis> {
        let mut analysis = ProjectAnalysis::default();
        
        // Detect project type
        analysis.project_type = self.detect_project_type(path).await?;
        
        // Scan for languages
        analysis.languages = self.detect_languages(path).await?;
        
        // Detect frameworks
        analysis.frameworks = self.detect_frameworks(path, &analysis.project_type).await?;
        
        // Find project root
        analysis.project_root = self.find_project_root(path).await?;
        
        Ok(analysis)
    }
    
    async fn detect_project_type(&self, path: &Path) -> Result<ProjectType> {
        if path.join("Cargo.toml").exists() {
            Ok(ProjectType::Rust)
        } else if path.join("package.json").exists() {
            let content = tokio::fs::read_to_string(path.join("package.json")).await?;
            if content.contains("\"typescript\"") {
                Ok(ProjectType::TypeScript)
            } else {
                Ok(ProjectType::JavaScript)
            }
        } else if path.join("requirements.txt").exists() 
            || path.join("setup.py").exists() 
            || path.join("pyproject.toml").exists() {
            Ok(ProjectType::Python)
        } else if path.join("go.mod").exists() {
            Ok(ProjectType::Go)
        } else if path.join("pom.xml").exists() || path.join("build.gradle").exists() {
            Ok(ProjectType::Java)
        } else if path.join("*.csproj").exists() || path.join("*.sln").exists() {
            Ok(ProjectType::CSharp)
        } else if path.join("Gemfile").exists() {
            Ok(ProjectType::Ruby)
        } else {
            Ok(ProjectType::Unknown)
        }
    }
    
    async fn detect_languages(&self, path: &Path) -> Result<Vec<Language>> {
        let mut languages = Vec::new();
        
        // Walk directory and detect file extensions
        let entries = tokio::fs::read_dir(path).await?;
        let mut entries = tokio_stream::wrappers::ReadDirStream::new(entries);
        
        use tokio_stream::StreamExt;
        while let Some(entry) = entries.next().await {
            if let Ok(entry) = entry {
                if let Some(ext) = entry.path().extension() {
                    match ext.to_str() {
                        Some("rs") => languages.push(Language::Rust),
                        Some("js") | Some("jsx") => languages.push(Language::JavaScript),
                        Some("ts") | Some("tsx") => languages.push(Language::TypeScript),
                        Some("py") => languages.push(Language::Python),
                        Some("go") => languages.push(Language::Go),
                        Some("java") => languages.push(Language::Java),
                        Some("cs") => languages.push(Language::CSharp),
                        Some("rb") => languages.push(Language::Ruby),
                        Some("sh") | Some("bash") => languages.push(Language::Shell),
                        Some("yml") | Some("yaml") => languages.push(Language::YAML),
                        Some("json") => languages.push(Language::JSON),
                        Some("md") => languages.push(Language::Markdown),
                        _ => {}
                    }
                }
            }
        }
        
        // Remove duplicates
        languages.sort();
        languages.dedup();
        
        Ok(languages)
    }
    
    async fn detect_frameworks(&self, path: &Path, project_type: &ProjectType) -> Result<Vec<Framework>> {
        let mut frameworks = Vec::new();
        
        match project_type {
            ProjectType::JavaScript | ProjectType::TypeScript => {
                if let Ok(content) = tokio::fs::read_to_string(path.join("package.json")).await {
                    if content.contains("\"react\"") {
                        frameworks.push(Framework::React);
                    }
                    if content.contains("\"vue\"") {
                        frameworks.push(Framework::Vue);
                    }
                    if content.contains("\"@angular/core\"") {
                        frameworks.push(Framework::Angular);
                    }
                    if content.contains("\"next\"") {
                        frameworks.push(Framework::NextJS);
                    }
                    if content.contains("\"express\"") {
                        frameworks.push(Framework::Express);
                    }
                    if content.contains("\"jest\"") {
                        frameworks.push(Framework::Jest);
                    }
                    if content.contains("\"mocha\"") {
                        frameworks.push(Framework::Mocha);
                    }
                }
            }
            ProjectType::Python => {
                if let Ok(content) = tokio::fs::read_to_string(path.join("requirements.txt")).await {
                    if content.contains("django") {
                        frameworks.push(Framework::Django);
                    }
                    if content.contains("flask") {
                        frameworks.push(Framework::Flask);
                    }
                    if content.contains("pytest") {
                        frameworks.push(Framework::PyTest);
                    }
                }
            }
            ProjectType::Rust => {
                if let Ok(content) = tokio::fs::read_to_string(path.join("Cargo.toml")).await {
                    if content.contains("actix-web") {
                        frameworks.push(Framework::Actix);
                    }
                    if content.contains("rocket") {
                        frameworks.push(Framework::Rocket);
                    }
                    if content.contains("axum") {
                        frameworks.push(Framework::Axum);
                    }
                }
            }
            _ => {}
        }
        
        Ok(frameworks)
    }
    
    async fn find_project_root(&self, path: &Path) -> Result<PathBuf> {
        let mut current = path.to_path_buf();
        
        loop {
            // Check for VCS markers
            if current.join(".git").exists() {
                return Ok(current);
            }
            
            // Check for project files
            if current.join("Cargo.toml").exists()
                || current.join("package.json").exists()
                || current.join("go.mod").exists()
                || current.join("pom.xml").exists() {
                return Ok(current);
            }
            
            // Move up
            if !current.pop() {
                break;
            }
        }
        
        // Default to original path
        Ok(path.to_path_buf())
    }
}

#[derive(Debug, Default)]
struct ProjectAnalysis {
    project_root: PathBuf,
    project_type: ProjectType,
    languages: Vec<Language>,
    frameworks: Vec<Framework>,
}

pub struct GitAnalyzer;

impl GitAnalyzer {
    pub async fn analyze(&self, path: &Path) -> Result<Option<GitInfo>> {
        // Check if path is in a git repository
        if !self.is_git_repo(path).await? {
            return Ok(None);
        }
        
        // Get current branch
        let current_branch = self.get_current_branch(path).await?;
        
        // Get remote URL
        let remote_url = self.get_remote_url(path).await.ok();
        
        // Get uncommitted changes
        let uncommitted_changes = self.get_uncommitted_changes(path).await?;
        
        // Get recent commits
        let recent_commits = self.get_recent_commits(path, 10).await?;
        
        // Get stash count
        let stash_count = self.get_stash_count(path).await?;
        
        Ok(Some(GitInfo {
            current_branch,
            remote_url,
            uncommitted_changes,
            recent_commits,
            stash_count,
        }))
    }
    
    async fn is_git_repo(&self, path: &Path) -> Result<bool> {
        Ok(path.join(".git").exists())
    }
    
    async fn get_current_branch(&self, _path: &Path) -> Result<String> {
        // This would run `git branch --show-current`
        Ok("main".to_string())
    }
    
    async fn get_remote_url(&self, _path: &Path) -> Result<String> {
        // This would run `git remote get-url origin`
        Ok("https://github.com/user/repo.git".to_string())
    }
    
    async fn get_uncommitted_changes(&self, _path: &Path) -> Result<Vec<FileChange>> {
        // This would run `git status --porcelain`
        Ok(vec![])
    }
    
    async fn get_recent_commits(&self, _path: &Path, _limit: usize) -> Result<Vec<Commit>> {
        // This would run `git log --format=...`
        Ok(vec![])
    }
    
    async fn get_stash_count(&self, _path: &Path) -> Result<usize> {
        // This would run `git stash list | wc -l`
        Ok(0)
    }
}

pub struct EnvironmentDetector;

impl EnvironmentDetector {
    pub async fn detect(&self) -> Result<EnvironmentInfo> {
        let environment_vars = self.get_relevant_env_vars();
        let installed_tools = self.detect_installed_tools().await?;
        let running_services = self.detect_running_services().await?;
        
        Ok(EnvironmentInfo {
            environment_vars,
            installed_tools,
            running_services,
        })
    }
    
    fn get_relevant_env_vars(&self) -> HashMap<String, String> {
        let mut vars = HashMap::new();
        
        // Get relevant environment variables (filtered for privacy)
        for (key, value) in std::env::vars() {
            if self.is_safe_env_var(&key) {
                vars.insert(key, value);
            }
        }
        
        vars
    }
    
    fn is_safe_env_var(&self, key: &str) -> bool {
        // Only include safe environment variables
        key.starts_with("NODE_")
            || key.starts_with("RUST_")
            || key.starts_with("PYTHON_")
            || key == "PATH"
            || key == "SHELL"
            || key == "EDITOR"
            || key == "LANG"
            || key == "LC_ALL"
    }
    
    async fn detect_installed_tools(&self) -> Result<Vec<Tool>> {
        let mut tools = Vec::new();
        
        // Check for common development tools
        let tool_checks = vec![
            ("cargo", "cargo --version"),
            ("node", "node --version"),
            ("npm", "npm --version"),
            ("python", "python --version"),
            ("git", "git --version"),
            ("docker", "docker --version"),
        ];
        
        for (name, version_cmd) in tool_checks {
            if let Ok(output) = tokio::process::Command::new("sh")
                .arg("-c")
                .arg(version_cmd)
                .output()
                .await
            {
                if output.status.success() {
                    let version = String::from_utf8_lossy(&output.stdout).trim().to_string();
                    if let Ok(path) = which::which(name) {
                        tools.push(Tool {
                            name: name.to_string(),
                            version,
                            path,
                        });
                    }
                }
            }
        }
        
        Ok(tools)
    }
    
    async fn detect_running_services(&self) -> Result<Vec<Service>> {
        // This would check for common development services
        // For now, return empty list
        Ok(vec![])
    }
}

#[derive(Debug)]
struct EnvironmentInfo {
    environment_vars: HashMap<String, String>,
    installed_tools: Vec<Tool>,
    running_services: Vec<Service>,
}

pub struct DependencyScanner;

impl DependencyScanner {
    pub async fn scan(&self, _workspace_root: &Path) -> Result<Vec<Dependency>> {
        let dependencies = Vec::new();
        
        // Scan for cross-repository dependencies
        // This is a simplified version - real implementation would be more sophisticated
        
        Ok(dependencies)
    }
}

pub struct ContextStore {
    contexts: tokio::sync::RwLock<HashMap<PathBuf, DeveloperContext>>,
}

impl ContextStore {
    pub fn new() -> Self {
        Self {
            contexts: tokio::sync::RwLock::new(HashMap::new()),
        }
    }
    
    pub async fn get(&self, path: &Path) -> Option<DeveloperContext> {
        let contexts = self.contexts.read().await;
        contexts.get(path).cloned()
    }
    
    pub async fn store(&self, path: PathBuf, context: DeveloperContext) {
        let mut contexts = self.contexts.write().await;
        contexts.insert(path, context);
    }
}

pub struct SessionMemory {
    events: tokio::sync::RwLock<VecDeque<ContextEvent>>,
    max_events: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextEvent {
    pub id: Uuid,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub event_type: EventType,
    pub data: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EventType {
    CommandExecuted,
    FileOpened,
    DirectoryChanged,
    ErrorOccurred,
    AgentInteraction,
}

impl SessionMemory {
    pub fn new(max_events: usize) -> Self {
        Self {
            events: tokio::sync::RwLock::new(VecDeque::with_capacity(max_events)),
            max_events,
        }
    }
    
    pub async fn record(&self, event: ContextEvent) {
        let mut events = self.events.write().await;
        events.push_back(event);
        
        // Maintain max size
        while events.len() > self.max_events {
            events.pop_front();
        }
    }
    
    pub async fn get_recent(&self, count: usize) -> Vec<ContextEvent> {
        let events = self.events.read().await;
        events.iter().rev().take(count).cloned().collect()
    }
}

pub struct PrivacyFilter;

impl PrivacyFilter {
    pub fn filter_context(&self, context: &mut DeveloperContext) {
        // Remove sensitive environment variables
        context.environment_vars.retain(|key, _| {
            !self.is_sensitive_key(key)
        });
        
        // Anonymize values that might contain secrets
        for (_, value) in &mut context.environment_vars {
            if self.might_contain_secret(value) {
                *value = "[REDACTED]".to_string();
            }
        }
    }
    
    fn is_sensitive_key(&self, key: &str) -> bool {
        let sensitive_patterns = [
            "PASSWORD", "SECRET", "KEY", "TOKEN", 
            "CREDENTIAL", "AUTH", "PRIVATE"
        ];
        
        sensitive_patterns.iter().any(|pattern| key.contains(pattern))
    }
    
    fn might_contain_secret(&self, value: &str) -> bool {
        // Simple heuristic - real implementation would be more sophisticated
        value.len() > 20 && value.chars().any(|c| !c.is_alphanumeric())
    }
}

impl ContextManagementSystem {
    pub fn new() -> Self {
        Self {
            directory_analyzer: DirectoryAnalyzer,
            git_analyzer: GitAnalyzer,
            environment_detector: EnvironmentDetector,
            dependency_scanner: DependencyScanner,
            context_store: Arc::new(ContextStore::new()),
            session_memory: Arc::new(SessionMemory::new(1000)),
            privacy_filter: PrivacyFilter,
        }
    }
    
    pub async fn build_context(&self, path: &Path) -> Result<DeveloperContext> {
        // Check cache first
        if let Some(cached) = self.context_store.get(path).await {
            return Ok(cached);
        }
        
        // Build new context
        let project_analysis = self.directory_analyzer.analyze(path).await?;
        let git_info = self.git_analyzer.analyze(&project_analysis.project_root).await?;
        let env_info = self.environment_detector.detect().await?;
        
        // Find workspace root
        let workspace_root = self.find_workspace_root(&project_analysis.project_root).await?;
        
        // Scan for related repos if in workspace
        let (related_repos, cross_repo_dependencies) = if let Some(ref ws_root) = workspace_root {
            let repos = self.find_related_repos(ws_root).await?;
            let deps = self.dependency_scanner.scan(ws_root).await?;
            (repos, deps)
        } else {
            (vec![], vec![])
        };
        
        // Build context
        let mut context = DeveloperContext {
            project_root: project_analysis.project_root.clone(),
            project_name: project_analysis.project_root
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("unknown")
                .to_string(),
            project_type: project_analysis.project_type,
            languages: project_analysis.languages,
            frameworks: project_analysis.frameworks,
            git_info,
            environment_vars: env_info.environment_vars,
            installed_tools: env_info.installed_tools,
            running_services: env_info.running_services,
            workspace_root,
            related_repos,
            cross_repo_dependencies,
            session_id: Uuid::new_v4(),
            session_start: chrono::Utc::now(),
        };
        
        // Apply privacy filter
        self.privacy_filter.filter_context(&mut context);
        
        // Cache the context
        self.context_store.store(path.to_path_buf(), context.clone()).await;
        
        Ok(context)
    }
    
    async fn find_workspace_root(&self, project_root: &Path) -> Result<Option<PathBuf>> {
        let mut current = project_root.to_path_buf();
        
        // Look for workspace markers
        loop {
            if current.join(".workspace").exists()
                || current.join("workspace.json").exists()
                || (current.join("Cargo.toml").exists() && 
                    tokio::fs::read_to_string(current.join("Cargo.toml"))
                        .await
                        .map(|c| c.contains("[workspace]"))
                        .unwrap_or(false)) {
                return Ok(Some(current));
            }
            
            if !current.pop() || current.parent().is_none() {
                break;
            }
        }
        
        Ok(None)
    }
    
    async fn find_related_repos(&self, workspace_root: &Path) -> Result<Vec<Repository>> {
        let mut repos = Vec::new();
        
        // Scan subdirectories for repositories
        let mut entries = tokio::fs::read_dir(workspace_root).await?;
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();
            if path.is_dir() && path.join(".git").exists() {
                if let Ok(project_analysis) = self.directory_analyzer.analyze(&path).await {
                    repos.push(Repository {
                        name: path.file_name()
                            .and_then(|n| n.to_str())
                            .unwrap_or("unknown")
                            .to_string(),
                        path: path.clone(),
                        remote_url: self.git_analyzer.analyze(&path)
                            .await
                            .ok()
                            .flatten()
                            .and_then(|g| g.remote_url),
                        main_language: project_analysis.languages.first().cloned()
                            .unwrap_or(Language::Rust),
                    });
                }
            }
        }
        
        Ok(repos)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_project_type_detection() {
        let analyzer = DirectoryAnalyzer;
        let temp_dir = tempfile::tempdir().unwrap();
        
        // Create a Rust project marker
        tokio::fs::write(temp_dir.path().join("Cargo.toml"), "[package]").await.unwrap();
        
        let project_type = analyzer.detect_project_type(temp_dir.path()).await.unwrap();
        assert_eq!(project_type, ProjectType::Rust);
    }
    
    #[tokio::test]
    async fn test_privacy_filter() {
        let filter = PrivacyFilter;
        let mut context = DeveloperContext {
            environment_vars: HashMap::from([
                ("PATH".to_string(), "/usr/bin".to_string()),
                ("API_KEY".to_string(), "secret123".to_string()),
                ("DATABASE_PASSWORD".to_string(), "password123".to_string()),
            ]),
            // ... other fields with default values
            project_root: PathBuf::new(),
            project_name: String::new(),
            project_type: ProjectType::Unknown,
            languages: vec![],
            frameworks: vec![],
            git_info: None,
            installed_tools: vec![],
            running_services: vec![],
            workspace_root: None,
            related_repos: vec![],
            cross_repo_dependencies: vec![],
            session_id: Uuid::new_v4(),
            session_start: chrono::Utc::now(),
        };
        
        filter.filter_context(&mut context);
        
        assert!(context.environment_vars.contains_key("PATH"));
        assert!(!context.environment_vars.contains_key("API_KEY"));
        assert!(!context.environment_vars.contains_key("DATABASE_PASSWORD"));
    }
}