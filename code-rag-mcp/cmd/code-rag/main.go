package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rafael/code-rag-mcp/internal/cli"
)

const version = "3.4.0"

func main() {
	// Parse command line flags
	var (
		standalone = flag.Bool("standalone", false, "Run without auto-managing services")
		simple     = flag.Bool("simple", false, "Use simple CLI without service management")
	)
	flag.Parse()
	
	// Force MCP server mode if requested - bypass all checks
	if len(os.Args) > 1 {
		command := strings.ToLower(os.Args[1])
		if command == "mcp-server" {
			app := cli.NewSimple("3.4.0")
			if err := app.RunMCPServerDirect(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		} else if command == "mcp-http" {
			app := cli.NewSimple("3.4.0")
			if err := app.RunMCPServerHTTP(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
	
	// Check if running in project-local mode
	if isProjectLocal() {
		// Check for MCP server command early - should run without services
		if len(os.Args) > 1 && strings.ToLower(os.Args[1]) == "mcp-server" {
			// Run MCP server directly without any stdout pollution
			fmt.Fprintf(os.Stderr, "[DEBUG] Running MCP server directly\n")
			app := cli.NewSimple(version)
			if err := app.RunMCPServerDirect(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		
		// Choose CLI mode based on flags
		if *simple || *standalone {
			// Use simple CLI without service management
			app := cli.NewSimple(version)
			if err := app.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Use integrated CLI with automatic service management
			app := cli.NewIntegrated(version, true)
			if err := app.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		// Show install instructions
		showInstallInstructions()
	}
}

func isProjectLocal() bool {
	// Check if we're running from a .code-rag installation
	if _, err := os.Stat(".code-rag/config.json"); err == nil {
		return true
	}
	
	// Check if we're the installed binary
	execPath, _ := os.Executable()
	if strings.Contains(execPath, ".code-rag") {
		return true
	}
	
	return false
}

func showInstallInstructions() {
	fmt.Println(`🔍 Code RAG - AI-Powered Code Search

This tool should be installed per project.

To install in your project:

  cd your-project
  curl -L https://get.code-rag.dev | sh

Or manually:

  mkdir -p .code-rag
  cp code-rag .code-rag/
  ./code-rag

For more info: https://github.com/rafael/code-rag-mcp`)
}