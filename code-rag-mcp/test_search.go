package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	
	"github.com/rafael/code-rag-mcp/internal/rag"
)

func main() {
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
	
	// Test search queries
	queries := []string{
		"Client",
		"HybridSearcher",
		"websocket handler",
		"authentication",
		"Connect",
	}
	
	fmt.Println("Testing search precision with improved exact matching:")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	for _, query := range queries {
		fmt.Printf("\n🔍 Query: '%s'\n", query)
		fmt.Println("-" + strings.Repeat("-", 40))
		
		results, err := engine.SearchInCollection(ctx, query, "agentx", "", 5)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		for i, result := range results {
			fmt.Printf("#%d Score: %.4f | %s:%d\n", i+1, result.Score, result.FilePath, result.LineStart)
			if result.Name != "" {
				fmt.Printf("   Name: %s (Type: %s)\n", result.Name, result.Type)
			}
			// Show first line of code
			lines := strings.Split(result.Code, "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				if len(firstLine) > 60 {
					firstLine = firstLine[:60] + "..."
				}
				fmt.Printf("   Code: %s\n", firstLine)
			}
		}
		
		if len(results) == 0 {
			fmt.Println("   No results found")
		}
	}
	
	// Test that node_modules are filtered
	fmt.Println("\n\n🧪 Testing node_modules filtering:")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	testQueries := []string{
		"react",
		"express",
		"lodash",
	}
	
	for _, query := range testQueries {
		fmt.Printf("\nQuery: '%s'\n", query)
		results, err := engine.SearchInCollection(ctx, query, "agentx", "", 10)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		nodeModulesCount := 0
		for _, result := range results {
			if strings.Contains(result.FilePath, "node_modules") {
				nodeModulesCount++
			}
		}
		
		fmt.Printf("Total results: %d\n", len(results))
		fmt.Printf("Results from node_modules: %d\n", nodeModulesCount)
		
		if nodeModulesCount > 0 {
			fmt.Println("⚠️  WARNING: node_modules results still appearing!")
		} else {
			fmt.Println("✅ No node_modules results (good!)")
		}
	}
}