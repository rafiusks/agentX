package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rafael/code-rag-mcp/internal/mcp"
	"github.com/rafael/code-rag-mcp/internal/rag"
)

type CLI struct {
	version   string
	ragEngine *rag.Engine
	config    *Config
	isFirstRun bool
}

type Config struct {
	ProjectPath    string
	MCPConfigured  bool
	QdrantRunning  bool
	ProjectsIndexed []string
}

func New(version string) *CLI {
	return &CLI{
		version: version,
		config:  loadOrCreateConfig(),
	}
}

func (c *CLI) Run() error {
	// Show banner
	c.showBanner()
	
	// Check if this is first run
	c.isFirstRun = c.checkFirstRun()
	
	if c.isFirstRun {
		// Run onboarding
		if err := c.runOnboarding(); err != nil {
			return fmt.Errorf("onboarding failed: %w", err)
		}
	} else {
		// Show status
		c.showStatus()
	}
	
	// Check for command line arguments
	if len(os.Args) > 1 {
		return c.handleCommand(os.Args[1:])
	}
	
	// Enter interactive mode
	return c.runInteractive()
}

func (c *CLI) showBanner() {
	fmt.Println("🔍 Code RAG - AI-powered code search")
	fmt.Println()
}

func (c *CLI) checkFirstRun() bool {
	configPath := filepath.Join(os.Getenv("HOME"), ".code-rag", "config.json")
	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

func (c *CLI) runOnboarding() error {
	fmt.Println("Welcome! Let's set up Code RAG in 30 seconds.\n")
	
	// Step 1: Check environment
	fmt.Print("Checking environment...")
	if err := c.checkEnvironment(); err != nil {
		fmt.Println(" ❌")
		return err
	}
	fmt.Println(" ✓")
	
	// Step 2: Start services
	fmt.Print("Starting vector database...")
	if err := c.startServices(); err != nil {
		fmt.Println(" ❌")
		return err
	}
	fmt.Println(" ✓")
	
	// Step 3: Index current directory
	currentDir, _ := os.Getwd()
	fmt.Printf("Indexing current project (%s)...", filepath.Base(currentDir))
	if err := c.indexProject(currentDir); err != nil {
		fmt.Println(" ❌")
		// Non-fatal, continue
	} else {
		fmt.Println(" ✓")
		c.config.ProjectsIndexed = append(c.config.ProjectsIndexed, currentDir)
	}
	
	// Step 4: Configure Claude if available
	fmt.Print("Configuring AI clients...")
	if err := c.autoConfigureClients(); err != nil {
		fmt.Println(" ⚠️ (manual setup needed)")
	} else {
		fmt.Println(" ✓")
		c.config.MCPConfigured = true
	}
	
	// Save configuration
	c.saveConfig()
	
	fmt.Println("\n✨ Setup complete!")
	fmt.Println("\nTry these commands:")
	fmt.Println("  code-rag search \"authentication\"")
	fmt.Println("  code-rag explain main.go:45")
	fmt.Println("  Or just type 'code-rag' for interactive mode\n")
	
	return nil
}

func (c *CLI) showStatus() {
	projects := c.getProjectStatus()
	projectCount := len(projects)
	
	if projectCount == 0 {
		fmt.Println("No projects indexed yet. Let's index your current directory:")
		fmt.Println("  code-rag index .")
		fmt.Println()
		return
	}
	
	// Calculate totals
	totalFiles, totalSize, lastUpdate := c.getIndexStats()
	
	if projectCount == 1 {
		proj := projects[0]
		fmt.Printf("📂 %s • %d files • %s • Updated %s\n\n", 
			proj.Name, proj.FilesIndexed, formatSize(proj.Size), formatTime(proj.LastIndexed))
	} else {
		fmt.Printf("📚 %d projects • %d files • %s • Updated %s\n\n",
			projectCount, totalFiles, formatSize(totalSize), formatTime(lastUpdate))
	}
}

func (c *CLI) handleCommand(args []string) error {
	if len(args) == 0 {
		return c.runInteractive()
	}
	
	command := strings.ToLower(args[0])
	
	switch command {
	case "mcp-server":
		// Run as MCP server (for Claude Code integration)
		return c.runMCPServer()
	case "search", "s":
		if len(args) < 2 {
			return fmt.Errorf("usage: code-rag search <query>")
		}
		query := strings.Join(args[1:], " ")
		return c.search(query)
		
	case "index", "i":
		path := "."
		if len(args) > 1 {
			path = args[1]
		}
		return c.indexProject(path)
		
	case "explain", "e":
		if len(args) < 2 {
			return fmt.Errorf("usage: code-rag explain <file:line>")
		}
		return c.explain(args[1])
		
	case "status":
		c.showDetailedStatus()
		return nil
		
	case "list", "ls":
		c.showDetailedProjectStatus()
		return nil
		
	case "help", "h":
		c.showHelp()
		return nil
		
	case "version", "v":
		fmt.Printf("Code RAG version %s\n", c.version)
		return nil
		
	default:
		// Treat unknown commands as search queries
		query := strings.Join(args, " ")
		return c.search(query)
	}
}

func (c *CLI) runInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		// Handle special commands
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return nil
		case "help", "?":
			c.showInteractiveHelp()
			continue
		case "status":
			c.showDetailedStatus()
			continue
		}
		
		// Parse natural language commands
		if err := c.handleNaturalCommand(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	
	return scanner.Err()
}

func (c *CLI) handleNaturalCommand(input string) error {
	lower := strings.ToLower(input)
	
	// Detect intent
	switch {
	case strings.HasPrefix(lower, "search "), 
	     strings.HasPrefix(lower, "find "),
	     strings.HasPrefix(lower, "look for "):
		query := strings.TrimPrefix(input, input[:strings.Index(input, " ")+1])
		return c.search(query)
		
	case strings.HasPrefix(lower, "index "),
	     strings.HasPrefix(lower, "add "):
		parts := strings.Fields(input)
		if len(parts) > 1 {
			return c.indexProject(parts[len(parts)-1])
		}
		return fmt.Errorf("please specify a path to index")
		
	case strings.HasPrefix(lower, "explain "):
		parts := strings.Fields(input)
		if len(parts) > 1 {
			return c.explain(parts[1])
		}
		return fmt.Errorf("please specify a file:line to explain")
		
	default:
		// Default to search
		return c.search(input)
	}
}

func (c *CLI) search(query string) error {
	// Initialize RAG engine if needed
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	fmt.Printf("Searching for: %s\n\n", query)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	results, err := c.ragEngine.Search(ctx, query, "any", 5)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}
	
	// Display results
	for i, result := range results {
		fmt.Printf("%d. %s (score: %.2f)\n", i+1, result.FilePath, result.Score)
		if result.LineStart > 0 {
			fmt.Printf("   Lines %d-%d\n", result.LineStart, result.LineEnd)
		}
		fmt.Printf("\n%s\n\n", truncateCode(result.Code, 200))
		fmt.Println(strings.Repeat("-", 60))
	}
	
	return nil
}

func (c *CLI) indexProject(path string) error {
	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	
	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}
	
	// Initialize RAG engine if needed
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	fmt.Printf("Indexing %s...\n", absPath)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	stats, err := c.ragEngine.IndexRepository(ctx, absPath, false)
	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}
	
	fmt.Printf("✓ Indexed %d files (%d chunks) in %s\n", 
		stats.FilesProcessed, stats.ChunksCreated, stats.Duration)
	
	// Update config
	c.config.ProjectsIndexed = append(c.config.ProjectsIndexed, absPath)
	c.saveConfig()
	
	return nil
}

func (c *CLI) explain(target string) error {
	// Parse file:line format
	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		return fmt.Errorf("format should be file:line (e.g., main.go:45)")
	}
	
	filePath := parts[0]
	
	// Read file content around the line
	content, err := readFileAroundLine(filePath, parts[1])
	if err != nil {
		return err
	}
	
	// Initialize RAG engine if needed
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	fmt.Printf("Explaining %s...\n\n", target)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	explanation, err := c.ragEngine.ExplainCode(ctx, content, filePath)
	if err != nil {
		return fmt.Errorf("explanation failed: %w", err)
	}
	
	fmt.Println(explanation)
	return nil
}

func (c *CLI) ensureRAGEngine() error {
	if c.ragEngine != nil {
		return nil
	}
	
	// Initialize with default config
	config := rag.DefaultConfig()
	engine, err := rag.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to initialize RAG engine: %w", err)
	}
	
	c.ragEngine = engine
	return nil
}

func (c *CLI) showHelp() {
	fmt.Println(`Usage: code-rag [command] [arguments]

Commands:
  search <query>    Search for code matching the query
  index <path>      Index a project directory
  explain <file:ln> Explain code at specific location
  list             Show all indexed projects
  status           Show detailed system status
  help             Show this help message
  version          Show version information

Examples:
  code-rag search "websocket handler"
  code-rag index ~/Code/my-project
  code-rag explain main.go:45
  code-rag list

Interactive mode:
  code-rag         Start interactive mode

First time setup:
  Just run 'code-rag' and it will set everything up automatically.`)
}

func (c *CLI) showInteractiveHelp() {
	fmt.Println(`Commands:
  search <query>    Find code matching your query
  index <path>      Add a project to the index
  explain <file:ln> Get explanation for specific code
  status           Show current status
  help             Show this help
  exit             Quit the program

You can also just type your search query directly!`)
}

func (c *CLI) showDetailedStatus() {
	// Show project details
	c.showDetailedProjectStatus()
	
	fmt.Println("\n🔧 System Status")
	fmt.Println(strings.Repeat("─", 60))
	
	// Vector DB status
	if c.checkQdrantRunning() {
		fmt.Println("✓ Vector database: Running")
	} else {
		fmt.Println("✗ Vector database: Not running")
		fmt.Println("  To start: docker run -d --name qdrant -p 6333:6333 qdrant/qdrant")
	}
	
	// MCP configuration
	if c.config.MCPConfigured {
		fmt.Println("✓ Claude Code: Configured")
	} else {
		fmt.Println("⚠ Claude Code: Not configured")
		fmt.Println("  To configure: code-rag configure")
	}
	
	// Show available commands
	fmt.Println("\n📝 Quick Commands")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("  code-rag search 'your query'  - Search all indexed code")
	fmt.Println("  code-rag index /path          - Index a new project")
	fmt.Println("  code-rag status               - Show this status")
	fmt.Println("  code-rag                      - Interactive mode")
}

// Helper functions
func truncateCode(code string, maxLen int) string {
	if len(code) <= maxLen {
		return code
	}
	return code[:maxLen] + "..."
}

func readFileAroundLine(filePath, lineStr string) (string, error) {
	// Simple implementation - in production, read specific lines
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (c *CLI) runMCPServer() error {
	// Initialize RAG engine
	if err := c.ensureRAGEngine(); err != nil {
		return err
	}
	
	// Create and run MCP server
	server := mcp.NewServer(c.ragEngine)
	ctx := context.Background()
	return server.Run(ctx)
}