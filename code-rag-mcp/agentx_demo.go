package main

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/rafael/code-rag-mcp/internal/rag"
)

func main() {
	fmt.Println("🚀 Code RAG Quick Test - AgentX Project")
	fmt.Println("=" + strings.Repeat("=", 50))
	
	// Initialize with default config
	config := rag.DefaultConfig()
	config.Silent = true
	
	engine, err := rag.NewEngine(config)
	if err != nil {
		panic(err)
	}
	
	ctx := context.Background()
	
	// Test searches on AgentX codebase
	testCases := []struct {
		query    string
		expected string
	}{
		{"ChatInterface", "React component"},
		{"ProviderManager", "Provider management"},
		{"websocket handler", "WebSocket handling"},
		{"authentication jwt", "JWT authentication"},
		{"useState", "React hooks"},
	}
	
	for _, tc := range testCases {
		fmt.Printf("\n🔍 Searching: '%s' (%s)\n", tc.query, tc.expected)
		fmt.Println(strings.Repeat("-", 40))
		
		results, err := engine.Search(ctx, tc.query, "", 2)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		
		if len(results) == 0 {
			fmt.Println("❌ No results found")
			continue
		}
		
		for i, result := range results {
			// Check for node_modules
			if strings.Contains(result.FilePath, "node_modules") {
				fmt.Printf("❌ #%d [CONTAMINATED] %s\n", i+1, result.FilePath)
			} else {
				// Clean path for display
				path := result.FilePath
				if strings.HasPrefix(path, "/Users/rafael/Code/agentX/") {
					path = path[28:] // Remove prefix for cleaner display
				}
				
				fmt.Printf("✅ #%d [%.2f] %s", i+1, result.Score, path)
				if result.Name != "" {
					fmt.Printf(" → %s", result.Name)
				}
				fmt.Println()
			}
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✨ Test Complete!")
}