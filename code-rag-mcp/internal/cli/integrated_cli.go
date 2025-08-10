package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rafael/code-rag-mcp/internal/services"
)

// IntegratedCLI manages both services and CLI interface
type IntegratedCLI struct {
	simpleCLI *SimpleCLI
	manager   *services.Manager
	autoStart bool
}

// NewIntegrated creates a new integrated CLI
func NewIntegrated(version string, autoStart bool) *IntegratedCLI {
	projectPath, _ := os.Getwd()
	
	return &IntegratedCLI{
		simpleCLI: NewSimple(version),
		manager:   services.NewManager(projectPath, false),
		autoStart: autoStart,
	}
}

// Run starts the integrated CLI with service management
func (i *IntegratedCLI) Run() error {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle signals in background
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Shutting down...")
		i.cleanup()
		os.Exit(0)
	}()
	
	// Show banner
	fmt.Println("🔍 Code RAG - Semantic Code Search with CodeBERT")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("")
	
	// Check if config exists
	if i.simpleCLI.config == nil {
		return i.runFirstTimeSetup(ctx)
	}
	
	// Auto-start services if requested
	if i.autoStart {
		if err := i.manager.EnsureRunning(ctx); err != nil {
			fmt.Printf("⚠️  Could not start services: %v\n", err)
			fmt.Println("Running in standalone mode (reduced functionality)")
		} else {
			fmt.Println("")
		}
	}
	
	// Show status
	if i.simpleCLI.config.Indexed {
		i.simpleCLI.showStatus()
	} else {
		fmt.Println("ℹ️  Project not indexed yet. Run 'index' to get started.")
	}
	
	// Check for command line arguments
	if len(os.Args) > 1 {
		return i.handleCommand(os.Args[1:])
	}
	
	// Run interactive mode
	return i.runInteractive(ctx)
}

// runFirstTimeSetup handles first-time initialization
func (i *IntegratedCLI) runFirstTimeSetup(ctx context.Context) error {
	fmt.Println("Welcome! Let's set up Code RAG for your project.")
	fmt.Println("")
	
	// Detect if we're in code-rag-mcp
	cwd, _ := os.Getwd()
	currentDir := filepath.Base(cwd)
	parentDir := filepath.Dir(cwd)
	parentName := filepath.Base(parentDir)
	
	var projectPath, projectName string
	var excludePaths []string
	
	if currentDir == "code-rag-mcp" && parentName != "" && parentName != "/" {
		fmt.Printf("📁 Detected: You're in Code RAG development folder inside '%s'\n", parentName)
		fmt.Println("\nWould you like to:")
		fmt.Printf("1. Index the parent project (%s) - Recommended\n", parentName)
		fmt.Println("2. Index this folder (code-rag-mcp)")
		fmt.Println("3. Exit")
		fmt.Print("\nChoice [1]: ")
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			choice := strings.TrimSpace(scanner.Text())
			if choice == "3" {
				return nil
			} else if choice == "2" {
				projectPath = cwd
				projectName = currentDir
				excludePaths = getDefaultExcludes()
			} else {
				projectPath = parentDir
				projectName = parentName
				excludePaths = append([]string{"code-rag-mcp"}, getDefaultExcludes()...)
			}
		}
	} else {
		projectPath = cwd
		projectName = currentDir
		excludePaths = getDefaultExcludes()
	}
	
	// Create config
	config := &ProjectConfig{
		ProjectPath:  projectPath,
		ProjectName:  projectName,
		ExcludePaths: excludePaths,
		CreatedAt:    time.Now(),
	}
	
	// Save config
	i.simpleCLI.config = config
	if err := i.simpleCLI.saveConfig(); err != nil {
		return err
	}
	
	fmt.Printf("\n✅ Initialized Code RAG for %s\n\n", projectName)
	
	// Ask about starting services
	fmt.Println("Would you like to start CodeBERT services for better search?")
	fmt.Println("(Requires Docker, uses ~2GB RAM)")
	fmt.Print("\nStart services? [Y/n]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "n" && response != "no" {
			if err := i.manager.Start(ctx); err != nil {
				fmt.Printf("⚠️  Could not start services: %v\n", err)
				fmt.Println("Continuing without CodeBERT (basic search only)")
			}
		}
	}
	
	// Index the project
	fmt.Println("\nIndexing your project...")
	return i.simpleCLI.indexProject()
}

// runInteractive runs the interactive CLI
func (i *IntegratedCLI) runInteractive(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nType to search, or use commands: index, status, services, help, exit")
	fmt.Println("")
	
	for {
		fmt.Print("🔍 > ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		// Handle commands
		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])
		
		switch command {
		case "exit", "quit", "q":
			i.cleanup()
			return nil
			
		case "help", "?":
			i.showHelp()
			
		case "index", "reindex":
			i.simpleCLI.indexProject()
			
		case "status":
			i.simpleCLI.showDetailedStatus()
			
		case "services":
			i.handleServicesCommand(ctx, parts[1:])
			
		case "list", "ls":
			args := []string{}
			if len(parts) > 1 {
				args = parts[1:]
			}
			i.simpleCLI.listIndexedFiles(args)
			
		default:
			// Treat as search query
			i.simpleCLI.search(input)
		}
		
		fmt.Println("")
	}
	
	return scanner.Err()
}

// handleCommand handles command-line arguments
func (i *IntegratedCLI) handleCommand(args []string) error {
	ctx := context.Background()
	command := strings.ToLower(args[0])
	
	switch command {
	case "mcp-server":
		// Run MCP server without starting services
		return i.simpleCLI.runMCPServer()
		
	case "start":
		return i.manager.Start(ctx)
		
	case "stop":
		return i.manager.Stop()
		
	case "status":
		i.manager.Status()
		return nil
		
	case "search", "s":
		// Ensure services are running for search
		if i.autoStart {
			i.manager.EnsureRunning(ctx)
		}
		if len(args) < 2 {
			return fmt.Errorf("usage: code-rag search <query>")
		}
		query := strings.Join(args[1:], " ")
		return i.simpleCLI.search(query)
		
	default:
		return i.simpleCLI.handleCommand(args)
	}
}

// handleServicesCommand manages service commands
func (i *IntegratedCLI) handleServicesCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		i.manager.Status()
		return
	}
	
	subCommand := strings.ToLower(args[0])
	switch subCommand {
	case "start":
		if err := i.manager.Start(ctx); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "stop":
		if err := i.manager.Stop(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "restart":
		i.manager.Stop()
		time.Sleep(2 * time.Second)
		if err := i.manager.Start(ctx); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "status":
		i.manager.Status()
	default:
		fmt.Println("Usage: services [start|stop|restart|status]")
	}
}

// showHelp displays help information
func (i *IntegratedCLI) showHelp() {
	fmt.Println(`
Commands:
  Search:
    <any text>         Search for code semantically
    list [-v]          List indexed files
    
  Management:
    index              Re-index the project
    status             Show project status
    services [cmd]     Manage Docker services (start/stop/status)
    
  System:
    help               Show this help
    exit               Exit (stops services if auto-started)
    
Examples:
  websocket handler      Find websocket-related code
  authentication logic   Find auth implementations
  list -v               Show all indexed files
  services start        Start CodeBERT services
`)
}

// cleanup performs cleanup before exit
func (i *IntegratedCLI) cleanup() {
	if i.autoStart {
		fmt.Println("Stopping services...")
		i.manager.Stop()
	}
	fmt.Println("Goodbye! 👋")
}