package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rafael/code-rag-mcp/internal/git"
	"github.com/rafael/code-rag-mcp/internal/watcher"
)

type Engine struct {
	embedder        *Embedder
	chunker         *Chunker
	improvedChunker *ImprovedChunker
	indexer         *Indexer
	retriever       *Retriever
	reranker        *Reranker
	hybridSearcher  *HybridSearcher
	bleveSearcher   *BleveSearcher
	searchCache     *SearchCache
	embeddingCache  *EmbeddingCache
	chunkCache      *FileChunkCache
	stats           *Statistics
	useImproved     bool
	silent          bool // Silent mode - no stdout output
	// Change monitoring
	fileWatcher     *watcher.Watcher
	gitTracker      *git.GitTracker
	hashStore       *watcher.HashStore
	changeMu        sync.Mutex
}

type SearchResult struct {
	FilePath   string
	Code       string
	Language   string
	Score      float32
	LineStart  int
	LineEnd    int
	Repository string
	Type       string // "function", "class", "method", "block", etc.
	Name       string // Name of the function/class/method
}

type IndexStats struct {
	FilesProcessed int
	ChunksCreated  int
	Duration       time.Duration
}

type Dependencies struct {
	Imports   []string
	Exports   []string
	Functions []string
	Classes   []string
}

type Repository struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	IndexedAt    time.Time `json:"indexed_at"`
	FileCount    int       `json:"file_count"`
	ChunkCount   int       `json:"chunk_count"`
	LastModified time.Time `json:"last_modified"`
}

type Statistics struct {
	TotalSearches      int       `json:"total_searches"`
	AverageLatency     float64   `json:"average_latency_ms"`
	TotalIndexedFiles  int       `json:"total_indexed_files"`
	TotalChunks        int       `json:"total_chunks"`
	LastIndexedAt      time.Time `json:"last_indexed_at"`
	MostSearchedTerms  []string  `json:"most_searched_terms"`
}

func NewEngine(config *Config) (*Engine, error) {
	embedder, err := NewEmbedder(config.EmbeddingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}
	
	chunker := NewChunker(config.ChunkingConfig)
	improvedChunker := NewImprovedChunker(config.ChunkingConfig)
	
	indexer, err := NewIndexer(config.VectorDBConfig, embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}
	
	retriever := NewRetriever(indexer, embedder)
	reranker := NewReranker()
	
	// Create Bleve searcher for keyword search with persistent storage
	// Use a project-local directory for the index
	bleveIndexPath := filepath.Join(".code-rag", "bleve-index")
	bleveSearcher, err := NewBleveSearcher(bleveIndexPath)
	if err != nil {
		// Log but don't fail - can work without keyword search
		if !config.Silent {
			fmt.Printf("Warning: Failed to create Bleve searcher: %v\n", err)
		}
	}
	
	// Create caches for better performance
	searchCache := NewSearchCache(1000, 15*time.Minute)
	embeddingCache := NewEmbeddingCache(10000)
	chunkCache := NewFileChunkCache(5000)
	
	engine := &Engine{
		embedder:        embedder,
		chunker:         chunker,
		improvedChunker: improvedChunker,
		indexer:         indexer,
		retriever:       retriever,
		reranker:        reranker,
		bleveSearcher:   bleveSearcher,
		searchCache:     searchCache,
		embeddingCache:  embeddingCache,
		chunkCache:      chunkCache,
		stats:           &Statistics{},
		useImproved:     true, // Use improved chunking by default
		silent:          config.Silent,
	}
	
	// Create hybrid searcher
	engine.hybridSearcher = NewHybridSearcher(engine)
	
	return engine, nil
}

func (e *Engine) SearchInCollection(ctx context.Context, query string, collection string, language string, limit int) ([]SearchResult, error) {
	// Check cache first
	if e.searchCache != nil {
		if cached, found := e.searchCache.Get(query, collection, limit); found {
			// Filter out node_modules even from cached results
			return e.filterNodeModules(cached), nil
		}
	}
	
	// Detect query intent to optimize search
	analyzer := NewQueryAnalyzer()
	intent := analyzer.AnalyzeQuery(query)
	
	// Search in specific collection
	startTime := time.Now()
	defer func() {
		e.stats.TotalSearches++
		e.stats.AverageLatency = (e.stats.AverageLatency*float64(e.stats.TotalSearches-1) + 
			float64(time.Since(startTime).Milliseconds())) / float64(e.stats.TotalSearches)
	}()
	
	// Use hybrid search if available
	if e.hybridSearcher != nil {
		// Pass intent to hybrid searcher for optimized search
		results, err := e.hybridSearcher.HybridSearchWithIntent(ctx, query, collection, limit*2, intent)
		if err != nil {
			return nil, err
		}
		// Filter out node_modules and limit results
		filtered := e.filterNodeModules(results)
		if len(filtered) > limit {
			filtered = filtered[:limit]
		}
		if e.searchCache != nil {
			// Cache successful results
			e.searchCache.Set(query, collection, limit, filtered)
		}
		return filtered, nil
	}
	
	// Fall back to semantic-only search
	queryEmbedding, err := e.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	// Retrieve from specific collection
	candidates, err := e.retriever.RetrieveFromCollection(ctx, queryEmbedding, collection, language, limit*3)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve candidates: %w", err)
	}
	
	// Rerank results
	results := e.reranker.Rerank(candidates, query, limit)
	
	return results, nil
}

func (e *Engine) Search(ctx context.Context, query string, language string, limit int) ([]SearchResult, error) {
	startTime := time.Now()
	defer func() {
		e.stats.TotalSearches++
		e.stats.AverageLatency = (e.stats.AverageLatency*float64(e.stats.TotalSearches-1) + 
			float64(time.Since(startTime).Milliseconds())) / float64(e.stats.TotalSearches)
	}()
	
	// Get query embedding
	queryEmbedding, err := e.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	// Retrieve candidates
	candidates, err := e.retriever.Retrieve(ctx, queryEmbedding, language, limit*3)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve candidates: %w", err)
	}
	
	// Rerank results
	results := e.reranker.Rerank(candidates, query, limit*2)
	
	// Filter out node_modules and limit
	filtered := e.filterNodeModules(results)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	
	return filtered, nil
}

func (e *Engine) ExplainCode(ctx context.Context, code string, filePath string) (string, error) {
	// First, find similar code for context
	similar, err := e.FindSimilar(ctx, code, 0.8)
	if err != nil {
		// Continue without similar code
		similar = []SearchResult{}
	}
	
	// Build context from similar code
	context := buildContext(similar)
	
	// Generate explanation (this would typically call an LLM)
	explanation := e.generateExplanation(code, context, filePath)
	
	return explanation, nil
}

func (e *Engine) FindSimilar(ctx context.Context, code string, threshold float64) ([]SearchResult, error) {
	// Get code embedding
	codeEmbedding, err := e.embedder.EmbedCode(ctx, code, "")
	if err != nil {
		return nil, fmt.Errorf("failed to embed code: %w", err)
	}
	
	// Find similar vectors
	results, err := e.indexer.FindSimilar(ctx, codeEmbedding, threshold, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar code: %w", err)
	}
	
	return results, nil
}

func (e *Engine) IndexRepositoryToCollection(ctx context.Context, path string, collection string, incremental bool) (*IndexStats, error) {
	return e.IndexRepositoryToCollectionWithOptions(ctx, path, collection, incremental, false, nil)
}

func (e *Engine) IndexRepositoryToCollectionWithOptions(ctx context.Context, path string, collection string, incremental bool, forceClean bool, excludePaths []string) (*IndexStats, error) {
	// Clear index if forced
	if forceClean {
		if !e.silent {
			fmt.Fprintf(os.Stderr, "Clearing existing index for collection %s...\n", collection)
		}
		err := e.ClearIndex(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to clear index: %w", err)
		}
		// Force non-incremental after clearing
		incremental = false
	}
	
	return e.IndexRepositoryToCollectionWithExcludes(ctx, path, collection, incremental, excludePaths)
}

func (e *Engine) IndexRepositoryToCollectionWithExcludes(ctx context.Context, path string, collection string, incremental bool, excludePaths []string) (*IndexStats, error) {
	// Index to specific collection
	startTime := time.Now()
	stats := &IndexStats{}
	
	// Discover files to index
	files, err := discoverFilesWithExcludes(path, excludePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}
	
	// Debug: log actual files being indexed
	if !e.silent {
		fmt.Fprintf(os.Stderr, "DEBUG: Found %d files to index\n", len(files))
		nodeModulesCount := 0
		for _, f := range files {
			if strings.Contains(f, "node_modules") {
				nodeModulesCount++
			}
		}
		if nodeModulesCount > 0 {
			fmt.Fprintf(os.Stderr, "ERROR: %d node_modules files are in the list!\n", nodeModulesCount)
		}
		// Show first few files
		if len(files) > 0 && len(files) < 20 {
			for _, f := range files {
				fmt.Fprintf(os.Stderr, "  - %s\n", f)
			}
		}
	}
	
	// Filter for incremental indexing if needed
	if incremental {
		files = e.filterModifiedFiles(files)
	}
	
	stats.FilesProcessed = len(files)
	
	// Process each file into the specific collection
	for _, file := range files {
		// Use improved chunker if enabled
		var chunks []Chunk
		if e.useImproved && e.improvedChunker != nil {
			chunks, err = e.improvedChunker.ChunkFile(file)
		} else {
			chunks, err = e.chunker.ChunkFile(file)
		}
		if err != nil {
			continue
		}
		
		for _, chunk := range chunks {
			embedding, err := e.embedder.EmbedCode(ctx, chunk.Code, chunk.Language)
			if err != nil {
				continue
			}
			
			// Index to vector store for semantic search
			err = e.indexer.IndexToCollection(ctx, chunk, embedding, collection)
			if err != nil {
				continue
			}
			
			// Also index to Bleve for keyword search
			if e.bleveSearcher != nil {
				e.bleveSearcher.IndexChunk(chunk)
			}
			
			stats.ChunksCreated++
		}
	}
	
	stats.Duration = time.Since(startTime)
	e.stats.TotalIndexedFiles += stats.FilesProcessed
	e.stats.TotalChunks += stats.ChunksCreated
	e.stats.LastIndexedAt = time.Now()
	
	return stats, nil
}

func (e *Engine) ClearIndex(ctx context.Context) error {
	// Clear all caches
	if e.searchCache != nil {
		e.searchCache.Clear()
	}
	if e.embeddingCache != nil {
		e.embeddingCache.Clear()
	}
	if e.chunkCache != nil {
		e.chunkCache.Clear()
	}
	
	// Clear Bleve index
	if e.bleveSearcher != nil {
		err := e.bleveSearcher.ClearIndex()
		if err != nil {
			return fmt.Errorf("failed to clear Bleve index: %w", err)
		}
	}
	
	// Reset statistics
	e.stats = &Statistics{}
	
	return nil
}

func (e *Engine) IndexRepository(ctx context.Context, path string, incremental bool) (*IndexStats, error) {
	return e.IndexRepositoryWithOptions(ctx, path, incremental, false)
}

func (e *Engine) IndexRepositoryWithOptions(ctx context.Context, path string, incremental bool, forceClean bool) (*IndexStats, error) {
	// Clear index if forced
	if forceClean {
		if !e.silent {
			fmt.Fprintf(os.Stderr, "Clearing existing index...\n")
		}
		err := e.ClearIndex(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to clear index: %w", err)
		}
		// Force non-incremental after clearing
		incremental = false
	}
	
	startTime := time.Now()
	stats := &IndexStats{}
	
	// Discover files to index
	files, err := discoverFiles(path)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}
	
	// Filter for incremental indexing if needed
	if incremental {
		files = e.filterModifiedFiles(files)
	}
	
	stats.FilesProcessed = len(files)
	
	// Process each file
	for _, file := range files {
		// Use improved chunker if enabled
		var chunks []Chunk
		if e.useImproved && e.improvedChunker != nil {
			chunks, err = e.improvedChunker.ChunkFile(file)
		} else {
			chunks, err = e.chunker.ChunkFile(file)
		}
		if err != nil {
			// Log error but continue
			continue
		}
		
		for _, chunk := range chunks {
			embedding, err := e.embedder.EmbedCode(ctx, chunk.Code, chunk.Language)
			if err != nil {
				continue
			}
			
			err = e.indexer.Index(ctx, chunk, embedding)
			if err != nil {
				continue
			}
			
			stats.ChunksCreated++
		}
	}
	
	stats.Duration = time.Since(startTime)
	e.stats.TotalIndexedFiles += stats.FilesProcessed
	e.stats.TotalChunks += stats.ChunksCreated
	e.stats.LastIndexedAt = time.Now()
	
	return stats, nil
}

func (e *Engine) GetDependencies(ctx context.Context, filePath string, includeTransitive bool) (*Dependencies, error) {
	// Parse file to extract dependencies
	deps, err := e.chunker.ExtractDependencies(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract dependencies: %w", err)
	}
	
	if includeTransitive {
		// Recursively get dependencies of dependencies
		transitiveDeps := e.getTransitiveDependencies(deps)
		deps = mergeDependencies(deps, transitiveDeps)
	}
	
	return deps, nil
}

func (e *Engine) SuggestImprovements(ctx context.Context, code string, focus string) (string, error) {
	// Analyze code for potential improvements
	analysis := e.analyzeCode(code, focus)
	
	// Find best practices from similar code
	similar, err := e.FindSimilar(ctx, code, 0.7)
	if err != nil {
		similar = []SearchResult{}
	}
	
	// Generate suggestions based on analysis and similar code
	suggestions := e.generateSuggestions(analysis, similar, focus)
	
	return suggestions, nil
}

func (e *Engine) GetIndexedRepositories() ([]Repository, error) {
	return e.indexer.GetRepositories()
}

func (e *Engine) GetStatistics() *Statistics {
	return e.stats
}

// Helper functions
func buildContext(similar []SearchResult) string {
	context := "Related code examples:\n\n"
	for i, result := range similar {
		if i >= 3 {
			break
		}
		context += fmt.Sprintf("Example from %s:\n```\n%s\n```\n\n", result.FilePath, result.Code)
	}
	return context
}

func (e *Engine) generateExplanation(code string, context string, filePath string) string {
	// This is a placeholder - in a real implementation, this would call an LLM
	explanation := "## Code Explanation\n\n"
	explanation += "This code appears to be a function or code block that performs specific operations.\n\n"
	
	if filePath != "" {
		explanation += fmt.Sprintf("**File:** %s\n\n", filePath)
	}
	
	if context != "" {
		explanation += "### Related Context\n\n"
		explanation += context
	}
	
	explanation += "### Analysis\n\n"
	explanation += "The code structure suggests it follows standard patterns for its language.\n"
	explanation += "Consider reviewing similar implementations in your codebase for consistency.\n"
	
	return explanation
}

func discoverFiles(path string) ([]string, error) {
	return discoverFilesWithExcludes(path, nil)
}

func discoverFilesWithExcludes(path string, excludePaths []string) ([]string, error) {
	var files []string
	var skippedCount int
	var skippedDirs []string
	
	// Default excludes for common directories that should always be ignored
	defaultExcludes := []string{
		"node_modules",
		"vendor", 
		"dist",
		"build",
		".git",
		".next",
		"out",
		"target",
		".pytest_cache",
		"__pycache__",
		".tox",
		".coverage",
		"htmlcov",
	}
	
	// Merge default excludes with provided excludes
	allExcludes := append(defaultExcludes, excludePaths...)
	
	// Load gitignore patterns with improved parser
	gitignorePath := filepath.Join(path, ".gitignore")
	gitignore, _ := NewImprovedGitIgnore(gitignorePath) // Ignore error if no .gitignore
	
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}
		
		// Get relative path for exclusion checking
		relPath, _ := filepath.Rel(path, filePath)
		
		// Skip directories first
		if info.IsDir() {
			name := filepath.Base(filePath)
			
			// CRITICAL: Check for node_modules specifically
			if name == "node_modules" || strings.Contains(relPath, "node_modules") {
				skippedDirs = append(skippedDirs, relPath)
				return filepath.SkipDir
			}
			
			// Check default excludes first (before gitignore for efficiency)
			for _, exclude := range allExcludes {
				if name == exclude {
					skippedDirs = append(skippedDirs, relPath)
					return filepath.SkipDir
				}
			}
			
			// Check gitignore for directories
			if gitignore != nil && gitignore.IsIgnored(filePath, true) {
				skippedDirs = append(skippedDirs, relPath)
				return filepath.SkipDir
			}
			
			// Skip hidden directories (except .github which often has workflows)
			if strings.HasPrefix(name, ".") && name != ".github" {
				if name != "." { // Don't add root to skipped
					skippedDirs = append(skippedDirs, relPath)
				}
				return filepath.SkipDir
			}
			
			return nil
		}
		
		// Check gitignore for files
		if gitignore != nil && gitignore.IsIgnored(filePath, false) {
			skippedCount++
			return nil
		}
		
		// Check if file path matches any exclude pattern
		for _, exclude := range allExcludes {
			if matched, _ := filepath.Match(exclude, filepath.Base(filePath)); matched {
				skippedCount++
				return nil
			}
			if strings.HasPrefix(relPath, exclude+string(filepath.Separator)) {
				skippedCount++
				return nil
			}
		}
		
		// Skip non-code files
		ext := filepath.Ext(filePath)
		if isCodeFile(ext) {
			files = append(files, filePath)
		} else {
			skippedCount++
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Log what we skipped (for debugging) - disabled for MCP compatibility
	// These were debug logs that pollute stdout in MCP mode
	
	return files, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isCodeFile(ext string) bool {
	codeExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".rb", ".php", ".swift", ".kt", ".rs", ".scala",
		".sh", ".bash", ".zsh", ".yml", ".yaml", ".json", ".xml", ".html", 
		".css", ".scss", ".sql",
	}
	
	for _, codeExt := range codeExts {
		if strings.EqualFold(ext, codeExt) {
			return true
		}
	}
	
	return false
}

func (e *Engine) filterModifiedFiles(files []string) []string {
	// Try git-based detection first
	if e.gitTracker != nil {
		// Get last indexed commit from hash store
		lastCommit := e.getLastIndexedCommit()
		
		// Get changed files since last index
		changedFiles, err := e.gitTracker.GetChangedFiles(lastCommit)
		if err == nil {
			// Convert to map for fast lookup
			changedMap := make(map[string]bool)
			for _, cf := range changedFiles {
				changedMap[cf.Path] = true
			}
			
			// Filter files to only changed ones
			var filtered []string
			for _, file := range files {
				if changedMap[file] {
					filtered = append(filtered, file)
				}
			}
			
			if !e.silent {
				fmt.Printf("Incremental indexing: %d of %d files changed\n", len(filtered), len(files))
			}
			return filtered
		}
	}
	
	// Fall back to hash-based detection
	if e.hashStore != nil {
		currentHashes := make(map[string]string)
		
		// Calculate current hashes
		for _, file := range files {
			if hash, err := e.calculateFileHash(file); err == nil {
				currentHashes[file] = hash
			}
		}
		
		// Get changed files from hash store
		created, modified, _ := e.hashStore.GetChangedFiles(currentHashes)
		
		var filtered []string
		filtered = append(filtered, created...)
		filtered = append(filtered, modified...)
		
		if !e.silent {
			fmt.Printf("Hash-based incremental: %d of %d files changed\n", len(filtered), len(files))
		}
		return filtered
	}
	
	// No incremental detection available, return all files
	return files
}

// calculateFileHash computes SHA-256 hash of file contents
func (e *Engine) calculateFileHash(path string) (string, error) {
	// Delegate to watcher if available
	if e.fileWatcher != nil {
		return "", fmt.Errorf("use watcher for hashing")
	}
	
	// Otherwise, calculate directly (for backward compatibility)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// This is simplified - in production, use crypto/sha256
	stat, _ := file.Stat()
	return fmt.Sprintf("%s_%d_%d", path, stat.Size(), stat.ModTime().Unix()), nil
}

// getLastIndexedCommit retrieves the last indexed git commit
func (e *Engine) getLastIndexedCommit() string {
	// This would be stored in the hash store or database
	// For now, return empty to index all changes
	return ""
}

// filterNodeModules removes any results from node_modules directories
func (e *Engine) filterNodeModules(results []SearchResult) []SearchResult {
	filtered := make([]SearchResult, 0, len(results))
	for _, result := range results {
		// Check if the file path contains node_modules
		if !strings.Contains(result.FilePath, "node_modules") &&
		   !strings.Contains(result.FilePath, "vendor") &&
		   !strings.Contains(result.FilePath, ".next") &&
		   !strings.Contains(result.FilePath, "dist/") &&
		   !strings.Contains(result.FilePath, "build/") {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func (e *Engine) getTransitiveDependencies(deps *Dependencies) *Dependencies {
	// This would implement recursive dependency resolution
	return &Dependencies{}
}

func mergeDependencies(deps1, deps2 *Dependencies) *Dependencies {
	// This would merge two dependency sets
	return deps1
}

func (e *Engine) analyzeCode(code string, focus string) map[string]interface{} {
	// This would implement code analysis logic
	return map[string]interface{}{
		"complexity": "medium",
		"issues":     []string{},
	}
}

func (e *Engine) generateSuggestions(analysis map[string]interface{}, similar []SearchResult, focus string) string {
	suggestions := "## Improvement Suggestions\n\n"
	
	switch focus {
	case "performance":
		suggestions += "### Performance Optimizations\n"
		suggestions += "- Consider caching frequently accessed data\n"
		suggestions += "- Review algorithm complexity\n"
		suggestions += "- Profile for bottlenecks\n\n"
		
	case "readability":
		suggestions += "### Readability Improvements\n"
		suggestions += "- Add descriptive variable names\n"
		suggestions += "- Break down complex functions\n"
		suggestions += "- Add documentation comments\n\n"
		
	case "security":
		suggestions += "### Security Considerations\n"
		suggestions += "- Validate all inputs\n"
		suggestions += "- Use parameterized queries\n"
		suggestions += "- Review authentication logic\n\n"
		
	case "best_practices":
		suggestions += "### Best Practices\n"
		suggestions += "- Follow language idioms\n"
		suggestions += "- Implement proper error handling\n"
		suggestions += "- Add unit tests\n\n"
		
	default:
		suggestions += "### General Improvements\n"
		suggestions += "- Review code organization\n"
		suggestions += "- Consider refactoring opportunities\n"
		suggestions += "- Ensure consistent style\n\n"
	}
	
	if len(similar) > 0 {
		suggestions += "### Examples from Your Codebase\n"
		suggestions += fmt.Sprintf("Found %d similar implementations that might serve as reference.\n", len(similar))
	}
	
	return suggestions
}