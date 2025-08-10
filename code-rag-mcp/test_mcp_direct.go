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
	config := rag.DefaultConfig()
	config.Silent = false
	
	// Create engine
	engine, err := rag.NewEngine(config)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Clear and re-index to ensure clean state
	fmt.Println("Clearing index...")
	engine.ClearIndex(ctx)
	
	fmt.Println("Indexing /Users/rafael/Code/agentX...")
	stats, err := engine.IndexRepository(ctx, "/Users/rafael/Code/agentX", false)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nIndexed: %d files, %d chunks in %v\n\n", 
		stats.FilesProcessed, stats.ChunksCreated, stats.Duration)
	
	// Test searches
	fmt.Println("Testing search results:")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	queries := []struct {
		query string
		desc  string
	}{
		{"HybridSearcher", "Go code search"},
		{"useState useEffect", "React hooks search"},
		{"babel webpack", "Build tools (should NOT return node_modules)"},
		{"Client Connect", "Exact match test"},
	}
	
	for _, q := range queries {
		fmt.Printf("\n🔍 %s: '%s'\n", q.desc, q.query)
		fmt.Println("-" + strings.Repeat("-", 40))
		
		results, err := engine.Search(ctx, q.query, "", 3)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		for i, result := range results {
			isNodeModules := strings.Contains(result.FilePath, "node_modules")
			marker := "✅"
			if isNodeModules {
				marker = "❌"
			}
			
			fmt.Printf("%s #%d [%.3f] %s\n", marker, i+1, result.Score, result.FilePath)
			if result.Name != "" {
				fmt.Printf("     Name: %s (Type: %s)\n", result.Name, result.Type)
			}
		}
		
		if len(results) == 0 {
			fmt.Println("   No results found")
		}
	}
	
	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✨ Test complete!")
	
	// Check for node_modules in entire index
	fmt.Println("\nChecking for node_modules contamination...")
	testQueries := []string{"lodash", "react-dom", "typescript", "@babel"}
	contaminated := false
	
	for _, q := range testQueries {
		results, _ := engine.Search(ctx, q, "", 10)
		for _, r := range results {
			if strings.Contains(r.FilePath, "node_modules") {
				contaminated = true
				fmt.Printf("⚠️  Found node_modules result for '%s': %s\n", q, r.FilePath)
				break
			}
		}
	}
	
	if !contaminated {
		fmt.Println("✅ No node_modules contamination detected!")
	} else {
		fmt.Println("❌ Index still contains node_modules entries")
	}
}