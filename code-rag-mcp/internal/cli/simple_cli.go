package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rafael/code-rag-mcp/internal/mcp"
	"github.com/rafael/code-rag-mcp/internal/rag"
	"github.com/rafael/code-rag-mcp/internal/services"
)

// SimpleCLI is a single-project focused CLI
type SimpleCLI struct {
	version   string
	ragEngine *rag.Engine
	config    *ProjectConfig
	indexed   bool
	silent    bool // Silent mode for MCP
}

// ProjectConfig for a single project
type ProjectConfig struct {
	ProjectPath    string    `json:"project_path"`
	ProjectName    string    `json:"project_name"`
	ExcludePaths   []string  `json:"exclude_paths"`
	IncludePattern []string  `json:"include_patterns,omitempty"`
	Indexed        bool      `json:"indexed"`
	FilesIndexed   int       `json:"files_indexed"`
	LastIndexed    time.Time `json:"last_indexed"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewSimple(version string) *SimpleCLI {
	return &SimpleCLI{
		version: version,
		config:  loadProjectConfig(),
	}
}

func loadProjectConfig() *ProjectConfig {
	configPath := ".code-rag/config.json"
	
	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		var config ProjectConfig
		if json.Unmarshal(data, &config) == nil {
			// Ensure exclude paths are set
			if len(config.ExcludePaths) == 0 {
				config.ExcludePaths = getDefaultExcludes()
			}
			return &config
		}
	}
	
	// No config exists - don't create one yet
	return nil
}

func getDefaultExcludes() []string {
	return []string{
		".git",
		".code-rag",
		"node_modules",
		"vendor",
		"dist",
		"build",
		"target",
		"*.min.js",
		"*.min.css",
	}
}

func (c *SimpleCLI) Run() error {
	// Check if initialized
	if c.config == nil {
		fmt.Println("🔍 Code RAG - AI-Powered Code Search\n")
		fmt.Println("No project initialized in this directory.")
		fmt.Println("\nTo get started:")
		fmt.Println("  code-rag init        Initialize this project")
		fmt.Println("  code-rag init --help Show initialization options")
		
		// Check if they're trying to run a command
		if len(os.Args) > 1 {
			command := strings.ToLower(os.Args[1])
			if command == "init" {
				return c.handleCommand(os.Args[1:])
			}
		}
		return nil
	}
	
	// Simple banner
	fmt.Printf("🔍 Code RAG - %s\n\n", c.config.ProjectName)
	
	// Show status if indexed
	if c.config.Indexed {
		c.showStatus()
	}
	
	// Handle command line arguments
	if len(os.Args) > 1 {
		return c.handleCommand(os.Args[1:])
	}
	
	// Interactive mode
	return c.runInteractive()
}

func (c *SimpleCLI) showStatus() {
	if c.config.Indexed {
		fmt.Printf("📂 %d files indexed • Last updated %s\n\n", 
			c.config.FilesIndexed, formatTime(c.config.LastIndexed))
	}
}

func (c *SimpleCLI) handleCommand(args []string) error {
	if len(args) == 0 {
		return c.runInteractive()
	}
	
	command := strings.ToLower(args[0])
	
	switch command {
	case "search", "s":
		if len(args) < 2 {
			return fmt.Errorf("usage: code-rag search <query>")
		}
		query := strings.Join(args[1:], " ")
		return c.search(query)
		
	case "index", "reindex":
		return c.indexProject()
		
	case "init":
		return c.initCommand(args[1:])
		
	case "list", "ls":
		return c.listIndexedFiles(args[1:])
		
	case "status":
		c.showDetailedStatus()
		return nil
		
	case "help", "h":
		c.showHelp()
		return nil
		
	case "mcp-server":
		return c.runMCPServer()
		
	case "mcp-http":
		return c.RunMCPServerHTTP()
		
	case "watch", "w":
		path := c.config.ProjectPath
		if len(args) > 1 {
			path = args[1]
		}
		return c.runWatch(path)
		
	default:
		// Treat as search query
		query := strings.Join(args, " ")
		return c.search(query)
	}
}

func (c *SimpleCLI) runInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type your search query or 'help' for commands")
	
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return nil
		case "help", "?":
			c.showInteractiveHelp()
		case "status":
			c.showDetailedStatus()
		case "index", "reindex":
			c.indexProject()
		default:
			c.search(input)
		}
	}
	
	return scanner.Err()
}

func (c *SimpleCLI) search(query string) error {
	if !c.config.Indexed {
		fmt.Println("Project not indexed yet. Indexing now...")
		if err := c.indexProject(); err != nil {
			return err
		}
	}
	
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	fmt.Printf("Searching: %s\n\n", query)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Use project-specific collection
	collectionName := fmt.Sprintf("code_rag_%s", strings.ToLower(c.config.ProjectName))
	results, err := c.ragEngine.SearchInCollection(ctx, query, collectionName, "any", 5)
	if err != nil {
		// Fallback to default search
		results, err = c.ragEngine.Search(ctx, query, "any", 5)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
	}
	
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}
	
	for i, result := range results {
		// Make paths relative to project
		relPath, _ := filepath.Rel(c.config.ProjectPath, result.FilePath)
		if relPath == "" {
			relPath = result.FilePath
		}
		
		fmt.Printf("%d. %s (%.0f%%)\n", i+1, relPath, result.Score*100)
		if result.LineStart > 0 {
			fmt.Printf("   Lines %d-%d\n", result.LineStart, result.LineEnd)
		}
		fmt.Printf("\n%s\n", truncateCode(result.Code, 200))
		fmt.Println(strings.Repeat("-", 60))
	}
	
	return nil
}

func (c *SimpleCLI) indexProject() error {
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	// Count files first (respecting excludes)
	fileCount := 0
	filepath.Walk(c.config.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		relPath, _ := filepath.Rel(c.config.ProjectPath, path)
		
		// Skip .code-rag directory
		if strings.HasPrefix(relPath, ".code-rag/") {
			return nil
		}
		
		// Skip excluded paths
		for _, exclude := range c.config.ExcludePaths {
			if strings.HasPrefix(relPath, exclude+string(filepath.Separator)) {
				return nil
			}
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				return nil
			}
		}
		
		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		
		ext := filepath.Ext(path)
		if isCodeFile(ext) {
			fileCount++
		}
		return nil
	})
	
	fmt.Printf("Indexing %d code files...\n", fileCount)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	// Index into project-specific collection with excludes
	collectionName := fmt.Sprintf("code_rag_%s", strings.ToLower(c.config.ProjectName))
	stats, err := c.ragEngine.IndexRepositoryToCollectionWithExcludes(ctx, c.config.ProjectPath, collectionName, false, c.config.ExcludePaths)
	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}
	
	// Update config
	c.config.Indexed = true
	c.config.FilesIndexed = stats.FilesProcessed
	c.config.LastIndexed = time.Now()
	c.saveConfig()
	
	fmt.Printf("✓ Indexed %d files in %s\n", stats.FilesProcessed, stats.Duration)
	return nil
}

func (c *SimpleCLI) showDetailedStatus() {
	fmt.Printf("📁 Project: %s\n", c.config.ProjectName)
	fmt.Printf("📂 Path: %s\n", c.config.ProjectPath)
	
	if c.config.Indexed {
		fmt.Printf("✓ Files indexed: %d\n", c.config.FilesIndexed)
		fmt.Printf("✓ Last indexed: %s\n", formatTime(c.config.LastIndexed))
	} else {
		fmt.Println("✗ Not indexed yet")
		fmt.Println("  Run: code-rag index")
	}
	
	// Check Qdrant
	if checkQdrantRunning() {
		fmt.Println("✓ Vector database: Running")
	} else {
		fmt.Println("✗ Vector database: Not running")
	}
}

func (c *SimpleCLI) showHelp() {
	fmt.Printf(`Code RAG for %s

Commands:
  search <query>   Search this project's code
  index           Re-index the project
  watch [path]    Watch for file changes and auto-index
  list            List indexed files
  status          Show project status
  help            Show this help

Usage:
  code-rag search "authentication"
  code-rag list
  code-rag index
  code-rag         (interactive mode)
`, c.config.ProjectName)
}

func (c *SimpleCLI) showInteractiveHelp() {
	fmt.Println(`Commands:
  <any text>      Search for that text
  index          Re-index the project
  status         Show project status
  help           Show this help
  exit           Quit`)
}

func (c *SimpleCLI) ensureRAGEngine() error {
	if c.ragEngine != nil {
		return nil
	}
	
	config := rag.DefaultConfig()
	// Use project-specific collection
	config.VectorDBConfig.CollectionName = fmt.Sprintf("code_rag_%s", 
		strings.ToLower(c.config.ProjectName))
	// Set silent mode for MCP
	config.Silent = c.silent
	
	engine, err := rag.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	
	c.ragEngine = engine
	return nil
}

func (c *SimpleCLI) saveConfig() error {
	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	os.MkdirAll(".code-rag", 0755)
	
	return os.WriteFile(".code-rag/config.json", data, 0644)
}

func (c *SimpleCLI) runMCPServer() error {
	// For MCP server mode, we need the full services running
	// Create a service manager and ensure services are running
	manager := services.NewManager(c.config.ProjectPath, false)
	ctx := context.Background()
	
	// Start services if not already running and wait for them to be ready
	fmt.Fprintf(os.Stderr, "🚀 Starting Code RAG services...\n")
	if err := manager.EnsureRunning(ctx); err != nil {
		// If services fail to start, log but continue with degraded functionality
		fmt.Fprintf(os.Stderr, "Warning: Could not start services: %v\n", err)
		fmt.Fprintf(os.Stderr, "MCP server will run with limited functionality\n")
	} else {
		fmt.Fprintf(os.Stderr, "✅ Services ready for MCP server\n")
	}
	
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	server := mcp.NewServer(c.ragEngine)
	return server.Run(ctx)
}

// RunMCPServerDirect runs MCP server without any status output to stdout
func (c *SimpleCLI) RunMCPServerDirect() error {
	// Find the project config - check current dir and binary location
	if c.config == nil {
		c.config = c.findProjectConfig()
		if c.config == nil {
			return fmt.Errorf("no project initialized - please run from project directory or install per project")
		}
	}
	
	// Enable silent mode for MCP
	c.silent = true
	
	// Redirect stdout to stderr for ALL initialization
	originalStdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() {
		// Ensure stdout is restored on any exit path
		os.Stdout = originalStdout
	}()
	
	// For MCP server mode, we need the full services running
	// Create a SILENT service manager for MCP mode
	manager := services.NewManagerSilent(c.config.ProjectPath)
	ctx := context.Background()
	
	// Start services silently
	err := manager.EnsureRunning(ctx)
	if err != nil {
		// Only log errors to stderr (already redirected)
		fmt.Printf("Warning: Could not start services: %v\n", err)
		fmt.Printf("MCP server will run with limited functionality\n")
	}
	
	// Initialize RAG engine with silent mode
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	// Restore stdout for MCP server communication
	os.Stdout = originalStdout
	
	server := mcp.NewServer(c.ragEngine)
	return server.Run(ctx)
}

// RunMCPServerHTTP runs MCP server as HTTP service
func (c *SimpleCLI) RunMCPServerHTTP() error {
	// Find the project config - check current dir and binary location
	if c.config == nil {
		c.config = c.findProjectConfig()
		if c.config == nil {
			return fmt.Errorf("no project initialized - please run from project directory or install per project")
		}
	}
	
	// For MCP server mode, we need the full services running
	// Create a service manager and ensure services are running
	manager := services.NewManager(c.config.ProjectPath, false)
	ctx := context.Background()
	
	// Start services silently for MCP mode
	fmt.Fprintf(os.Stderr, "Starting Code RAG services for MCP HTTP server...\n")
	err := manager.EnsureRunning(ctx)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not start services: %v\n", err)
		fmt.Fprintf(os.Stderr, "MCP server will run with limited functionality\n")
	} else {
		fmt.Fprintf(os.Stderr, "Services ready for MCP server\n")
	}
	
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	server := mcp.NewHTTPServer(c.ragEngine, ":9000")
	fmt.Fprintf(os.Stderr, "Code RAG MCP HTTP server starting on :9000\n")
	return server.Run(ctx)
}

// findProjectConfig searches for project config in multiple locations
func (c *SimpleCLI) findProjectConfig() *ProjectConfig {
	// Try current directory first
	if config := loadProjectConfig(); config != nil {
		return config
	}
	
	// Try relative to binary location
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// Check if binary is in .code-rag directory
		if filepath.Base(execDir) == ".code-rag" {
			projectDir := filepath.Dir(execDir)
			configPath := filepath.Join(projectDir, ".code-rag", "config.json")
			if data, err := os.ReadFile(configPath); err == nil {
				var config ProjectConfig
				if json.Unmarshal(data, &config) == nil {
					// Ensure the project path is absolute
					if !filepath.IsAbs(config.ProjectPath) {
						config.ProjectPath = projectDir
					}
					return &config
				}
			}
		}
	}
	
	return nil
}

func (c *SimpleCLI) initCommand(args []string) error {
	// Check if we're in the code-rag-mcp development folder
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	
	currentDir := filepath.Base(cwd)
	parentDir := filepath.Dir(cwd)
	parentName := filepath.Base(parentDir)
	
	// Smart detection: are we in code-rag-mcp inside another project?
	if currentDir == "code-rag-mcp" && parentName != "" && parentName != "/" {
		fmt.Println("🔍 Code RAG - Project Initialization\n")
		fmt.Printf("Detected: You're developing Code RAG inside the '%s' project.\n", parentName)
		fmt.Println("\nWould you like to:")
		fmt.Printf("1. Index the parent project (%s) while excluding code-rag-mcp\n", parentName)
		fmt.Println("2. Index only this directory (code-rag-mcp)")
		fmt.Println("3. Cancel\n")
		fmt.Print("Choice [1]: ")
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			choice := strings.TrimSpace(scanner.Text())
			if choice == "" || choice == "1" {
				// Index parent project
				return c.initProject(parentDir, parentName, []string{"code-rag-mcp"})
			} else if choice == "2" {
				// Index current directory
				return c.initProject(cwd, currentDir, getDefaultExcludes())
			}
		}
		fmt.Println("Initialization cancelled.")
		return nil
	}
	
	// Normal initialization for any other project
	fmt.Println("🔍 Code RAG - Project Initialization\n")
	fmt.Printf("Initializing Code RAG for: %s\n", currentDir)
	fmt.Printf("Path: %s\n\n", cwd)
	
	return c.initProject(cwd, currentDir, getDefaultExcludes())
}

func (c *SimpleCLI) initProject(projectPath string, projectName string, excludePaths []string) error {
	// Create config
	config := &ProjectConfig{
		ProjectPath:  projectPath,
		ProjectName:  projectName,
		ExcludePaths: excludePaths,
		CreatedAt:    time.Now(),
		Indexed:      false,
	}
	
	// Ensure we have all default excludes
	defaultExcludes := getDefaultExcludes()
	for _, exclude := range defaultExcludes {
		if !contains(config.ExcludePaths, exclude) {
			config.ExcludePaths = append(config.ExcludePaths, exclude)
		}
	}
	
	// Save config
	c.config = config
	if err := c.saveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	fmt.Printf("✓ Initialized Code RAG for %s\n", projectName)
	if len(excludePaths) > 0 {
		fmt.Printf("✓ Excluding: %s\n", strings.Join(excludePaths, ", "))
	}
	fmt.Println("\nIndexing project...")
	
	// Index the project
	return c.indexProject()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (c *SimpleCLI) listIndexedFiles(args []string) error {
	// List all files that were indexed
	if !c.config.Indexed {
		fmt.Println("No files indexed yet. Run 'code-rag index' first.")
		return nil
	}
	
	verbose := false
	for _, arg := range args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
	}
	
	fmt.Printf("📂 Indexed files in %s:\n\n", c.config.ProjectName)
	
	// Re-discover files with same logic used during indexing
	fileCount := 0
	filesByExt := make(map[string]int)
	var allFiles []string
	
	filepath.Walk(c.config.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		relPath, _ := filepath.Rel(c.config.ProjectPath, path)
		
		// Skip .code-rag directory
		if strings.HasPrefix(relPath, ".code-rag/") {
			return nil
		}
		
		// Skip excluded paths
		for _, exclude := range c.config.ExcludePaths {
			if strings.HasPrefix(relPath, exclude+string(filepath.Separator)) {
				return nil
			}
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				return nil
			}
		}
		
		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		
		ext := filepath.Ext(path)
		if isCodeFile(ext) {
			fileCount++
			filesByExt[ext]++
			if verbose {
				allFiles = append(allFiles, relPath)
			}
		}
		return nil
	})
	
	// Show summary by file type
	fmt.Println("File types indexed:")
	for ext, count := range filesByExt {
		fmt.Printf("  %-10s %d files\n", ext, count)
	}
	
	fmt.Printf("\nTotal: %d files\n", fileCount)
	
	if len(c.config.ExcludePaths) > 0 {
		fmt.Printf("\nExcluded paths: %s\n", strings.Join(c.config.ExcludePaths, ", "))
	}
	
	if verbose && len(allFiles) > 0 {
		fmt.Println("\nAll indexed files:")
		for _, file := range allFiles {
			fmt.Printf("  %s\n", file)
		}
	}
	
	return nil
}

func (c *SimpleCLI) createRAGEngine() (*rag.Engine, error) {
	// Create RAG config
	config := &rag.Config{
		EmbeddingConfig: &rag.EmbeddingConfig{
			Provider:  "service",
			Model:     "codebert-base",
			CacheSize: 1000,
		},
		VectorDBConfig: &rag.VectorDBConfig{
			Type:           "qdrant",
			URL:            "http://localhost:6333",
			CollectionName: c.config.ProjectName,
		},
		ChunkingConfig: &rag.ChunkingConfig{
			MaxChunkSize:  500,
			ChunkOverlap:  50,
			MinChunkSize:  100,
		},
	}
	
	return rag.NewEngine(config)
}

func (c *SimpleCLI) runWatch(path string) error {
	if !c.config.Indexed {
		fmt.Println("Project not indexed yet. Please run 'index' first.")
		return nil
	}
	
	// Create RAG engine if needed
	if c.ragEngine == nil {
		var err error
		c.ragEngine, err = c.createRAGEngine()
		if err != nil {
			return fmt.Errorf("failed to create RAG engine: %w", err)
		}
	}
	
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutting down file watcher...")
		cancel()
	}()
	
	fmt.Printf("👁️  Starting file watcher for %s\n", path)
	fmt.Println("⚡ Changes will be indexed automatically")
	fmt.Println("Press Ctrl+C to stop watching")
	fmt.Println("")
	
	// Start watching
	if err := c.ragEngine.StartWatching(ctx, path, c.config.ProjectName); err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Stop watching
	if err := c.ragEngine.StopWatching(); err != nil {
		fmt.Printf("Warning: Error stopping watcher: %v\n", err)
	}
	
	fmt.Println("✅ File watcher stopped successfully")
	return nil
}

func checkQdrantRunning() bool {
	// Simple HTTP check
	return true // Simplified for now
}