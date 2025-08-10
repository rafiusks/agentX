package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// HybridSearcher combines semantic and keyword search
type HybridSearcher struct {
	engine         *Engine
	indexer        *Indexer
	embedder       *Embedder
	analyzer       *QueryAnalyzer
	bleveSearcher  *BleveSearcher
	crossEncoder   *CrossEncoderClient
}

// NewHybridSearcher creates a new hybrid searcher
func NewHybridSearcher(engine *Engine) *HybridSearcher {
	// Create cross-encoder client for high-precision reranking
	crossEncoder := NewCrossEncoderClient("http://localhost:8002")
	
	// Use the engine's Bleve searcher if available
	return &HybridSearcher{
		engine:        engine,
		indexer:       engine.indexer,
		embedder:      engine.embedder,
		analyzer:      NewQueryAnalyzer(),
		bleveSearcher: engine.bleveSearcher, // Use the engine's persistent Bleve searcher
		crossEncoder:  crossEncoder,
	}
}

// HybridSearch performs both semantic and keyword search, then merges results
func (hs *HybridSearcher) HybridSearch(ctx context.Context, query string, collection string, limit int) ([]SearchResult, error) {
	// 0. Analyze query intent
	intent := hs.analyzer.AnalyzeQuery(query)
	
	// 1. Perform semantic search with expanded query
	semanticResults, err := hs.semanticSearch(ctx, query, collection, limit*2)
	if err != nil {
		// Fall back to keyword search only
		return hs.keywordSearch(query, collection, limit)
	}
	
	// 2. Perform keyword search
	keywordResults, _ := hs.keywordSearch(query, collection, limit*2)
	
	// 3. Merge and rerank results
	merged := hs.mergeResults(semanticResults, keywordResults, query)
	
	// 4. Apply intent-based boosting
	merged = hs.analyzer.ApplyIntentToSearch(intent, merged)
	
	// 5. Apply final reranking
	reranked := hs.rerankResults(merged, query)
	
	// 6. Apply cross-encoder reranking for top results (if available)
	if hs.crossEncoder != nil && len(reranked) > 0 {
		// Rerank top 20 results for efficiency
		numToRerank := min(20, len(reranked))
		topResults := reranked[:numToRerank]
		
		// Try cross-encoder reranking
		crossReranked, err := hs.crossEncoder.Rerank(ctx, query, topResults, numToRerank)
		if err == nil {
			// Replace top results with cross-encoder reranked results
			reranked = append(crossReranked, reranked[numToRerank:]...)
		}
	}
	
	// 7. Limit results
	if len(reranked) > limit {
		return reranked[:limit], nil
	}
	
	return reranked, nil
}

// semanticSearch performs embedding-based search
func (hs *HybridSearcher) semanticSearch(ctx context.Context, query string, collection string, limit int) ([]SearchResult, error) {
	// Expand query for better semantic matching
	expandedQuery := hs.expandQuery(query)
	
	// Get embedding for expanded query
	embedding, err := hs.embedder.EmbedQuery(ctx, expandedQuery)
	if err != nil {
		return nil, err
	}
	
	// Search in vector store
	results, err := hs.indexer.SearchInCollection(ctx, embedding, collection, "", limit)
	if err != nil {
		return nil, err
	}
	
	// Normalize scores to 0-1 range
	for i := range results {
		results[i].Score = normalizeScore(results[i].Score)
	}
	
	return results, nil
}

// keywordSearch performs BM25-style keyword search using Bleve
func (hs *HybridSearcher) keywordSearch(query string, collection string, limit int) ([]SearchResult, error) {
	// If Bleve searcher is not available, return empty results
	if hs.bleveSearcher == nil {
		return []SearchResult{}, nil
	}
	
	// Perform keyword search using Bleve
	results, err := hs.bleveSearcher.Search(query, collection, limit)
	if err != nil {
		// Log error but don't fail the entire search
		fmt.Printf("Keyword search error: %v\n", err)
		return []SearchResult{}, nil
	}
	
	return results, nil
}

// HybridSearchWithIntent performs hybrid search with intent-based optimization
func (hs *HybridSearcher) HybridSearchWithIntent(ctx context.Context, query string, collection string, limit int, intent QueryIntent) ([]SearchResult, error) {
	// Default weights
	semanticWeight := float32(0.5)
	keywordWeight := float32(0.5)
	
	switch intent.SearchType {
	case "definition":
		// For definitions, keyword search is more important
		keywordWeight = 0.7
		semanticWeight = 0.3
	case "usage", "example":
		// For usage examples, semantic search is better
		semanticWeight = 0.7
		keywordWeight = 0.3
	case "implementation":
		// Balanced for implementations
		semanticWeight = 0.5
		keywordWeight = 0.5
	}
	
	// Expand query based on intent
	expandedQuery := hs.expandQuery(query)
	if intent.SearchType == "definition" {
		// For definitions, also search for the exact term
		expandedQuery = query + " " + expandedQuery
	}
	
	// Perform semantic search
	semanticResults, err := hs.semanticSearch(ctx, expandedQuery, collection, limit*2)
	if err != nil {
		// Log but continue with keyword search
		semanticResults = []SearchResult{}
	}
	
	// Perform keyword search with intent-based boosting
	keywordResults, err := hs.keywordSearchWithIntent(expandedQuery, collection, limit*2, intent)
	if err != nil {
		// Log but continue
		keywordResults = []SearchResult{}
	}
	
	// Combine and re-rank results
	combined := hs.combineResults(semanticResults, keywordResults, semanticWeight, keywordWeight)
	
	// Apply intent-based filtering
	if intent.SearchType == "definition" {
		// Prioritize results with matching names
		combined = hs.prioritizeByName(combined, query)
	}
	
	// Limit results
	if len(combined) > limit {
		combined = combined[:limit]
	}
	
	return combined, nil
}

// keywordSearchWithIntent performs keyword search with intent-based boosting
func (hs *HybridSearcher) keywordSearchWithIntent(query string, collection string, limit int, intent QueryIntent) ([]SearchResult, error) {
	if hs.engine.bleveSearcher == nil {
		return []SearchResult{}, nil
	}
	
	// Modify search based on intent
	searchQuery := query
	if intent.SearchType == "definition" {
		// For definitions, search for exact matches
		searchQuery = "name:" + query + " OR " + query
	}
	
	results, err := hs.engine.bleveSearcher.Search(searchQuery, collection, limit)
	if err != nil {
		return []SearchResult{}, err
	}
	
	return results, nil
}

// combineResults merges semantic and keyword search results with weighted scoring
func (hs *HybridSearcher) combineResults(semanticResults, keywordResults []SearchResult, semanticWeight, keywordWeight float32) []SearchResult {
	// Create map to track combined scores
	scoreMap := make(map[string]*SearchResult)
	
	// Add semantic results
	for _, result := range semanticResults {
		key := result.FilePath + ":" + fmt.Sprintf("%d", result.LineStart)
		if existing, ok := scoreMap[key]; ok {
			existing.Score = existing.Score + result.Score*semanticWeight
		} else {
			r := result
			r.Score = result.Score * semanticWeight
			scoreMap[key] = &r
		}
	}
	
	// Add keyword results
	for _, result := range keywordResults {
		key := result.FilePath + ":" + fmt.Sprintf("%d", result.LineStart)
		if existing, ok := scoreMap[key]; ok {
			existing.Score = existing.Score + result.Score*keywordWeight
		} else {
			r := result
			r.Score = result.Score * keywordWeight
			scoreMap[key] = &r
		}
	}
	
	// Convert map to slice
	var combined []SearchResult
	for _, result := range scoreMap {
		combined = append(combined, *result)
	}
	
	// Sort by score
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Score > combined[j].Score
	})
	
	return combined
}

// prioritizeByName moves results with matching names to the top
func (hs *HybridSearcher) prioritizeByName(results []SearchResult, query string) []SearchResult {
	queryLower := strings.ToLower(query)
	
	// Separate exact matches and others
	var exactMatches []SearchResult
	var otherResults []SearchResult
	
	for _, result := range results {
		if strings.ToLower(result.Name) == queryLower {
			exactMatches = append(exactMatches, result)
		} else {
			otherResults = append(otherResults, result)
		}
	}
	
	// Combine with exact matches first
	return append(exactMatches, otherResults...)
}

// expandIdentifierPatterns expands camelCase, PascalCase, snake_case, and kebab-case patterns
func expandIdentifierPatterns(query string) string {
	result := query
	words := strings.Fields(query)
	
	for _, word := range words {
		// Handle camelCase and PascalCase
		camelParts := splitCamelCase(word)
		if len(camelParts) > 1 {
			// Add individual parts and snake_case version
			result += " " + strings.Join(camelParts, " ")
			result += " " + strings.ToLower(strings.Join(camelParts, "_"))
		}
		
		// Handle snake_case
		if strings.Contains(word, "_") {
			parts := strings.Split(word, "_")
			result += " " + strings.Join(parts, " ")
			// Add camelCase version
			camelVersion := ""
			for i, part := range parts {
				if i == 0 {
					camelVersion += strings.ToLower(part)
				} else {
					camelVersion += strings.Title(strings.ToLower(part))
				}
			}
			result += " " + camelVersion
		}
		
		// Handle kebab-case
		if strings.Contains(word, "-") {
			parts := strings.Split(word, "-")
			result += " " + strings.Join(parts, " ")
			result += " " + strings.Join(parts, "_")
		}
		
		// Handle dot notation (e.g., Client.Connect)
		if strings.Contains(word, ".") {
			parts := strings.Split(word, ".")
			result += " " + strings.Join(parts, " ")
		}
	}
	
	return result
}

// splitCamelCase splits a camelCase or PascalCase string into parts
func splitCamelCase(s string) []string {
	var parts []string
	var current []rune
	
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Check if next char is lowercase (new word boundary)
			if i+1 < len(s) {
				nextRune := rune(s[i+1])
				if unicode.IsLower(nextRune) || (len(current) > 0 && unicode.IsLower(current[len(current)-1])) {
					if len(current) > 0 {
						parts = append(parts, string(current))
						current = []rune{}
					}
				}
			} else if len(current) > 0 {
				parts = append(parts, string(current))
				current = []rune{}
			}
		}
		current = append(current, r)
	}
	
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	
	return parts
}

// expandQuery expands the query with synonyms and related terms
func (hs *HybridSearcher) expandQuery(query string) string {
	// First, expand camelCase and snake_case patterns
	expanded := expandIdentifierPatterns(query)
	lower := strings.ToLower(expanded)
	
	// Comprehensive synonym groups for better semantic matching
	synonymGroups := [][]string{
		// Authentication & User Management
		{"auth", "authentication", "authorization", "login", "signin", "signup", "register", "registration", "logout", "signout", "session", "jwt", "token", "oauth", "credentials", "password"},
		{"user", "account", "profile", "member", "customer", "person", "identity", "principal"},
		
		// CRUD Operations
		{"create", "add", "new", "insert", "post", "save", "store", "register", "make", "build"},
		{"read", "get", "fetch", "retrieve", "find", "search", "query", "list", "show", "view"},
		{"update", "edit", "modify", "patch", "put", "change", "alter", "revise", "set"},
		{"delete", "remove", "destroy", "drop", "purge", "clear", "erase", "trash"},
		
		// Web Concepts
		{"websocket", "ws", "socket", "realtime", "real-time", "streaming", "stream", "connection", "push"},
		{"api", "endpoint", "rest", "graphql", "route", "path", "resource", "service"},
		{"handler", "controller", "processor", "listener", "callback", "responder"},
		{"middleware", "interceptor", "filter", "hook", "plugin", "guard", "decorator"},
		{"request", "req", "input", "payload", "body", "params", "query"},
		{"response", "res", "resp", "output", "result", "reply", "answer"},
		
		// Database
		{"db", "database", "sql", "postgres", "postgresql", "mysql", "mongo", "mongodb", "redis", "store", "repository", "repo"},
		{"model", "schema", "entity", "table", "collection", "document", "record"},
		{"migration", "migrate", "upgrade", "seed", "initialize"},
		
		// Error Handling
		{"error", "err", "exception", "panic", "failure", "fault", "catch", "throw", "reject"},
		{"validate", "validation", "verify", "check", "ensure", "assert", "confirm"},
		
		// Testing
		{"test", "spec", "testing", "unittest", "unit-test", "integration", "e2e", "mock", "stub", "spy", "fixture"},
		
		// Configuration
		{"config", "configuration", "settings", "env", "environment", "options", "preferences", "setup"},
		
		// Caching
		{"cache", "caching", "redis", "memcached", "memory", "buffer", "store"},
		
		// Messaging
		{"message", "msg", "event", "notification", "alert", "email", "sms", "push"},
		{"queue", "mq", "rabbitmq", "kafka", "pubsub", "publish", "subscribe", "broker"},
		
		// Common Programming Terms
		{"function", "func", "method", "procedure", "routine", "operation"},
		{"class", "struct", "type", "interface", "trait", "protocol"},
		{"import", "require", "include", "use", "dependency"},
		{"export", "expose", "public", "api"},
		{"init", "initialize", "setup", "bootstrap", "start", "begin", "constructor", "new"},
		{"close", "shutdown", "cleanup", "dispose", "teardown", "stop", "end", "destructor"},
		
		// Additional High-Value Patterns
		{"handler", "controller", "endpoint", "route", "view", "action", "processor", "listener"},
		{"async", "await", "promise", "callback", "then", "asynchronous", "concurrent", "parallel"},
		{"sync", "synchronous", "blocking", "sequential"},
		{"connect", "connection", "connected", "disconnect", "disconnected", "reconnect"},
		{"subscribe", "subscription", "unsubscribe", "publish", "publisher", "subscriber"},
		{"emit", "event", "trigger", "fire", "dispatch", "broadcast"},
		{"transform", "convert", "parse", "serialize", "deserialize", "encode", "decode"},
		{"validate", "validator", "validation", "verify", "check", "ensure", "assert"},
		{"filter", "map", "reduce", "aggregate", "collect", "group", "sort"},
		{"push", "pop", "enqueue", "dequeue", "peek", "shift", "unshift"},
		{"lock", "unlock", "mutex", "semaphore", "atomic", "synchronized"},
		{"begin", "commit", "rollback", "transaction", "savepoint"},
	}
	
	// Find all matching synonyms
	queryWords := strings.Fields(lower)
	addedSynonyms := make(map[string]bool)
	
	for _, word := range queryWords {
		for _, group := range synonymGroups {
			for _, term := range group {
				if term == word {
					// Add all synonyms in this group
					for _, synonym := range group {
						if synonym != word && !addedSynonyms[synonym] {
							addedSynonyms[synonym] = true
							expanded += " " + synonym
						}
					}
					break
				}
			}
		}
	}
	
	// Add language-specific context
	if strings.Contains(lower, "go") || strings.Contains(lower, "golang") {
		expanded += " golang go func"
	}
	if strings.Contains(lower, "js") || strings.Contains(lower, "javascript") {
		expanded += " javascript js node async await"
	}
	if strings.Contains(lower, "react") {
		expanded += " component jsx tsx hooks"
	}
	
	return expanded
}

// mergeResults combines semantic and keyword results using Reciprocal Rank Fusion
func (hs *HybridSearcher) mergeResults(semantic, keyword []SearchResult, query string) []SearchResult {
	// Use Reciprocal Rank Fusion (RRF) for more robust merging
	// RRF score = 1 / (k + rank), where k is a constant (typically 60)
	const k = 60.0
	
	// Create a map to track combined scores
	scoreMap := make(map[string]float64)
	resultMap := make(map[string]*SearchResult)
	
	// Add semantic results with RRF scoring
	for rank, result := range semantic {
		key := fmt.Sprintf("%s:%d-%d", result.FilePath, result.LineStart, result.LineEnd)
		rrfScore := 1.0 / (k + float64(rank+1))
		scoreMap[key] += rrfScore
		
		// Keep the full result for later
		if _, exists := resultMap[key]; !exists {
			resultCopy := result
			resultMap[key] = &resultCopy
		}
	}
	
	// Add keyword results with RRF scoring
	for rank, result := range keyword {
		key := fmt.Sprintf("%s:%d-%d", result.FilePath, result.LineStart, result.LineEnd)
		rrfScore := 1.0 / (k + float64(rank+1))
		scoreMap[key] += rrfScore
		
		// Keep the full result for later
		if _, exists := resultMap[key]; !exists {
			resultCopy := result
			resultMap[key] = &resultCopy
		}
	}
	
	// Create final results with RRF scores
	var merged []SearchResult
	for key, rrfScore := range scoreMap {
		if result, ok := resultMap[key]; ok {
			result.Score = float32(rrfScore)
			merged = append(merged, *result)
		}
	}
	
	// Sort by RRF score (descending)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	
	return merged
}

// rerankResults applies final reranking based on code-specific signals
func (hs *HybridSearcher) rerankResults(results []SearchResult, query string) []SearchResult {
	// First, deduplicate results based on file path and line numbers
	seen := make(map[string]bool)
	deduped := make([]SearchResult, 0, len(results))
	
	for _, result := range results {
		key := fmt.Sprintf("%s:%d-%d", result.FilePath, result.LineStart, result.LineEnd)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, result)
		}
	}
	
	results = deduped
	queryTokens := tokenize(query)
	
	// File-type weighting for better relevance
	fileTypeWeights := map[string]float32{
		"handler":    1.5,  // API handlers likely most relevant
		"controller": 1.5,  // Controllers are important
		"service":    1.3,  // Business logic
		"model":      1.2,  // Data structures
		"schema":     1.2,  // Database schemas
		"route":      1.4,  // Routing definitions
		"api":        1.4,  // API definitions
		"repository": 1.1,  // Data access layer
		"util":       0.8,  // Utility functions less likely
		"helper":     0.8,  // Helper functions
		"test":       0.7,  // Tests usually not what's wanted
		"spec":       0.7,  // Test specs
		"mock":       0.6,  // Mock implementations
	}
	
	for i := range results {
		// Boost based on various factors
		boost := float32(1.0)
		
		// Apply file-type weighting
		pathLower := strings.ToLower(results[i].FilePath)
		for pattern, weight := range fileTypeWeights {
			if strings.Contains(pathLower, pattern) {
				boost *= weight
				break // Apply only the first matching weight
			}
		}
		
		// 1. Exact match boost
		codeLower := strings.ToLower(results[i].Code)
		for _, token := range queryTokens {
			if strings.Contains(codeLower, token) {
				boost *= 1.1
			}
		}
		
		// MASSIVE boost for exact name matches
		if len(queryTokens) > 0 && results[i].Name != "" {
			queryMainWord := queryTokens[0]
			if len(queryTokens) > 1 {
				// Try to find the most important word (longest or capitalized)
				for _, token := range queryTokens {
					if len(token) > len(queryMainWord) {
						queryMainWord = token
					}
				}
			}
			if strings.EqualFold(results[i].Name, queryMainWord) {
				boost *= 5.0  // Massive boost for exact name matches
			} else if strings.Contains(strings.ToLower(results[i].Name), queryMainWord) {
				boost *= 2.5  // Good boost for partial name matches
			}
		}
		
		// 2. File path relevance
		// pathLower already declared above
		if strings.Contains(query, "handler") && strings.Contains(pathLower, "handler") {
			boost *= 1.2
		}
		if strings.Contains(query, "service") && strings.Contains(pathLower, "service") {
			boost *= 1.2
		}
		if strings.Contains(query, "test") && strings.Contains(pathLower, "test") {
			boost *= 1.3
		}
		
		// 3. Code type boost
		if strings.Contains(query, "function") && results[i].Type == "function" {
			boost *= 1.2
		}
		if strings.Contains(query, "class") && results[i].Type == "class" {
			boost *= 1.2
		}
		
		// 4. Language match
		if strings.Contains(query, "go") && results[i].Language == "go" {
			boost *= 1.15
		}
		if strings.Contains(query, "react") && (results[i].Language == "jsx" || results[i].Language == "tsx") {
			boost *= 1.15
		}
		
		// 5. Penalize very short or very long chunks
		codeLength := len(results[i].Code)
		if codeLength < 50 {
			boost *= 0.8
		} else if codeLength > 2000 {
			boost *= 0.9
		}
		
		// Apply boost
		results[i].Score *= boost
	}
	
	// Sort by final score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

// tokenize splits query into tokens
func tokenize(text string) []string {
	var tokens []string
	var current []rune
	
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = []rune{}
		}
	}
	
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	
	return tokens
}

// normalizeScore normalizes scores to 0-1 range
func normalizeScore(score float32) float32 {
	// Assuming cosine similarity which is already -1 to 1
	// Convert to 0-1 range
	normalized := (score + 1) / 2
	
	// Apply sigmoid for better distribution
	return float32(1 / (1 + math.Exp(-float64(normalized*10-5))))
}