package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	
	"github.com/rafael/code-rag-mcp/internal/rag"
)

func main() {
	// Test the file discovery
	fmt.Println("Testing file discovery for AgentX:")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	projectPath := "/Users/rafael/Code/agentX"
	
	// Create config
	config := &rag.Config{
		EmbeddingConfig: &rag.EmbeddingConfig{
			Provider:  "minilm",
			Model:     "minilm",
		},
		ChunkingConfig: &rag.ChunkingConfig{
			MaxChunkSize: 500,
			ChunkOverlap: 100,
			MinChunkSize: 50,
		},
		VectorDBConfig: &rag.VectorDBConfig{
			Type:      "memory",
			URL:       "",
			Dimension: 384,
		},
		Silent: false,
	}
	
	// Create engine
	engine, err := rag.NewEngine(config)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Index with debug output
	fmt.Printf("\nIndexing %s...\n", projectPath)
	stats, err := engine.IndexRepositoryWithOptions(ctx, projectPath, false, true)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nIndexing complete:\n")
	fmt.Printf("- Files processed: %d\n", stats.FilesProcessed)
	fmt.Printf("- Chunks created: %d\n", stats.ChunksCreated)
	fmt.Printf("- Time taken: %s\n", stats.Duration)
	
	// Now test search to verify no node_modules
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Testing that node_modules are filtered:")
	
	// Search for something that would definitely be in node_modules
	queries := []string{"babel", "webpack", "lodash", "react-dom"}
	
	for _, query := range queries {
		fmt.Printf("\nSearching for '%s':\n", query)
		results, err := engine.Search(ctx, query, "", 5)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		nodeModulesCount := 0
		for _, result := range results {
			relPath, _ := filepath.Rel(projectPath, result.FilePath)
			if strings.Contains(result.FilePath, "node_modules") {
				nodeModulesCount++
				fmt.Printf("  ⚠️  node_modules result: %s\n", relPath)
			} else {
				fmt.Printf("  ✓ Clean result: %s\n", relPath)
			}
		}
		
		if nodeModulesCount > 0 {
			fmt.Printf("  WARNING: %d node_modules results found!\n", nodeModulesCount)
		} else {
			fmt.Printf("  ✅ No node_modules results\n")
		}
	}
}