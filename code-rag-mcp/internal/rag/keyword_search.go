package rag

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// BleveSearcher implements keyword search using Bleve
type BleveSearcher struct {
	index    bleve.Index
	indexPath string
	mu       sync.RWMutex
}

// CodeDocument represents a document in the search index
type CodeDocument struct {
	ID           string   `json:"id"`
	Content      string   `json:"content"`
	FilePath     string   `json:"filepath"`
	Language     string   `json:"language"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Symbols      []string `json:"symbols"`
	LineStart    int      `json:"line_start"`
	LineEnd      int      `json:"line_end"`
	Repository   string   `json:"repository"`
}

// NewBleveSearcher creates a new Bleve-based keyword searcher
func NewBleveSearcher(indexPath string) (*BleveSearcher, error) {
	// Create custom analyzer for code
	err := createCodeAnalyzer()
	if err != nil {
		return nil, fmt.Errorf("failed to create code analyzer: %w", err)
	}
	
	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index with custom mapping
		mapping := createCodeMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}
	
	return &BleveSearcher{
		index:     index,
		indexPath: indexPath,
	}, nil
}

// createCodeAnalyzer creates a custom analyzer for code
func createCodeAnalyzer() error {
	// For now, we'll use the standard analyzer
	// Custom analyzers require more complex setup
	return nil
}

// createCodeMapping creates the index mapping for code documents
func createCodeMapping() *mapping.IndexMappingImpl {
	mapping := bleve.NewIndexMapping()
	
	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	
	// Content field - searchable text
	contentField := bleve.NewTextFieldMapping()
	contentField.Analyzer = "standard"
	contentField.Store = true
	contentField.IncludeInAll = true
	docMapping.AddFieldMappingsAt("content", contentField)
	
	// FilePath field - keyword field for exact matching
	pathField := bleve.NewKeywordFieldMapping()
	pathField.Store = true
	pathField.IncludeInAll = false
	docMapping.AddFieldMappingsAt("filepath", pathField)
	
	// Language field
	langField := bleve.NewKeywordFieldMapping()
	langField.Store = true
	docMapping.AddFieldMappingsAt("language", langField)
	
	// Type field (function, class, etc.)
	typeField := bleve.NewKeywordFieldMapping()
	typeField.Store = true
	docMapping.AddFieldMappingsAt("type", typeField)
	
	// Name field - boosted for exact matches
	nameField := bleve.NewTextFieldMapping()
	nameField.Analyzer = "standard"
	nameField.Store = true
	nameField.IncludeInAll = true
	docMapping.AddFieldMappingsAt("name", nameField)
	
	// Symbols field - extracted identifiers
	symbolsField := bleve.NewTextFieldMapping()
	symbolsField.Analyzer = "standard"
	symbolsField.Store = false
	symbolsField.IncludeInAll = true
	docMapping.AddFieldMappingsAt("symbols", symbolsField)
	
	// Numeric fields
	numericField := bleve.NewNumericFieldMapping()
	numericField.Store = true
	numericField.IncludeInAll = false
	docMapping.AddFieldMappingsAt("line_start", numericField)
	docMapping.AddFieldMappingsAt("line_end", numericField)
	
	// Set default analyzer
	mapping.DefaultAnalyzer = "standard"
	mapping.DefaultMapping = docMapping
	
	return mapping
}

// IndexChunk indexes a code chunk
func (bs *BleveSearcher) IndexChunk(chunk Chunk) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	// Extract symbols from code
	symbols := extractSymbols(chunk.Code, chunk.Language)
	
	doc := CodeDocument{
		ID:         generateChunkID(chunk),
		Content:    chunk.Code,
		FilePath:   chunk.FilePath,
		Language:   chunk.Language,
		Type:       chunk.Type,
		Name:       chunk.Name,
		Symbols:    symbols,
		LineStart:  chunk.LineStart,
		LineEnd:    chunk.LineEnd,
		Repository: chunk.Repository,
	}
	
	return bs.index.Index(doc.ID, doc)
}

// DeleteFile removes all chunks for a file from the index
func (bs *BleveSearcher) DeleteFile(filePath string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	// Search for all documents with this file path
	query := bleve.NewTermQuery(filePath)
	query.SetField("file_path")
	
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 1000 // Get all chunks for this file
	
	result, err := bs.index.Search(searchRequest)
	if err != nil {
		return err
	}
	
	// Delete each document
	for _, hit := range result.Hits {
		if err := bs.index.Delete(hit.ID); err != nil {
			// Log but continue
			fmt.Printf("Failed to delete document %s: %v\n", hit.ID, err)
		}
	}
	
	return nil
}

// Search performs keyword search
func (bs *BleveSearcher) Search(queryStr string, collection string, limit int) ([]SearchResult, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	// Build multi-field query
	boolQuery := bleve.NewBooleanQuery()
	
	// Exact phrase match (highest boost)
	phraseQuery := bleve.NewMatchPhraseQuery(queryStr)
	phraseQuery.SetField("content")
	phraseQuery.SetBoost(3.0)
	boolQuery.AddShould(phraseQuery)
	
	// Name field match (high boost for function/class names)
	nameQuery := bleve.NewMatchQuery(queryStr)
	nameQuery.SetField("name")
	nameQuery.SetBoost(2.5)
	boolQuery.AddShould(nameQuery)
	
	// Fuzzy match for typos
	fuzzyQuery := bleve.NewFuzzyQuery(queryStr)
	fuzzyQuery.SetField("content")
	fuzzyQuery.SetFuzziness(2)
	fuzzyQuery.SetBoost(1.5)
	boolQuery.AddShould(fuzzyQuery)
	
	// Regular match query
	matchQuery := bleve.NewMatchQuery(queryStr)
	matchQuery.SetField("content")
	matchQuery.SetBoost(1.0)
	boolQuery.AddShould(matchQuery)
	
	// Symbol match
	symbolQuery := bleve.NewMatchQuery(queryStr)
	symbolQuery.SetField("symbols")
	symbolQuery.SetBoost(1.2)
	boolQuery.AddShould(symbolQuery)
	
	// Create search request
	searchRequest := bleve.NewSearchRequest(boolQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}
	searchRequest.IncludeLocations = false
	
	// Execute search
	searchResult, err := bs.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Convert to SearchResult
	results := make([]SearchResult, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		result := SearchResult{
			Score: float32(hit.Score),
		}
		
		// Extract fields
		if filepath, ok := hit.Fields["filepath"].(string); ok {
			result.FilePath = filepath
		}
		if content, ok := hit.Fields["content"].(string); ok {
			result.Code = content
		}
		if lang, ok := hit.Fields["language"].(string); ok {
			result.Language = lang
		}
		if lineStart, ok := hit.Fields["line_start"].(float64); ok {
			result.LineStart = int(lineStart)
		}
		if lineEnd, ok := hit.Fields["line_end"].(float64); ok {
			result.LineEnd = int(lineEnd)
		}
		if typeStr, ok := hit.Fields["type"].(string); ok {
			result.Type = typeStr
		}
		if name, ok := hit.Fields["name"].(string); ok {
			result.Name = name
		}
		if repo, ok := hit.Fields["repository"].(string); ok {
			result.Repository = repo
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// SearchWithFilters performs keyword search with additional filters
func (bs *BleveSearcher) SearchWithFilters(queryStr string, filters map[string]string, limit int) ([]SearchResult, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	// Build compound query
	compoundQuery := bleve.NewBooleanQuery()
	
	// Add main search query
	mainQuery := bleve.NewMatchQuery(queryStr)
	compoundQuery.AddMust(mainQuery)
	
	// Add filters
	for field, value := range filters {
		filterQuery := bleve.NewTermQuery(value)
		filterQuery.SetField(field)
		compoundQuery.AddMust(filterQuery)
	}
	
	searchRequest := bleve.NewSearchRequest(compoundQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}
	
	searchResult, err := bs.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	
	// Convert results (same as above)
	results := make([]SearchResult, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		result := SearchResult{
			Score: float32(hit.Score),
		}
		
		// Extract fields (same as Search method)
		if filepath, ok := hit.Fields["filepath"].(string); ok {
			result.FilePath = filepath
		}
		if content, ok := hit.Fields["content"].(string); ok {
			result.Code = content
		}
		if lang, ok := hit.Fields["language"].(string); ok {
			result.Language = lang
		}
		if lineStart, ok := hit.Fields["line_start"].(float64); ok {
			result.LineStart = int(lineStart)
		}
		if lineEnd, ok := hit.Fields["line_end"].(float64); ok {
			result.LineEnd = int(lineEnd)
		}
		if typeStr, ok := hit.Fields["type"].(string); ok {
			result.Type = typeStr
		}
		if name, ok := hit.Fields["name"].(string); ok {
			result.Name = name
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// DeleteChunk removes a chunk from the index
func (bs *BleveSearcher) DeleteChunk(chunkID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	return bs.index.Delete(chunkID)
}

// Close closes the index
func (bs *BleveSearcher) Close() error {
	return bs.index.Close()
}

// extractSymbols extracts identifiers from code
func extractSymbols(code string, language string) []string {
	symbols := []string{}
	
	if code == "" {
		return symbols
	}
	
	// Language-specific patterns
	var patterns []*regexp.Regexp
	
	switch language {
	case "go":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`func\s+(\w+)`),
			regexp.MustCompile(`type\s+(\w+)`),
			regexp.MustCompile(`var\s+(\w+)`),
			regexp.MustCompile(`const\s+(\w+)`),
			regexp.MustCompile(`interface\s+(\w+)`),
		}
	case "javascript", "typescript":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`function\s+(\w+)`),
			regexp.MustCompile(`class\s+(\w+)`),
			regexp.MustCompile(`const\s+(\w+)`),
			regexp.MustCompile(`let\s+(\w+)`),
			regexp.MustCompile(`var\s+(\w+)`),
		}
	case "python":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`def\s+(\w+)`),
			regexp.MustCompile(`class\s+(\w+)`),
		}
	default:
		// Generic patterns
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`(\w+)\s*\(`), // Function calls
			regexp.MustCompile(`(\w+)\s*=`),  // Assignments
		}
	}
	
	// Extract symbols using patterns
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				symbols = append(symbols, match[1])
			}
		}
	}
	
	// Also extract camelCase and snake_case identifiers
	identifierPattern := regexp.MustCompile(`\b[a-zA-Z_]\w+\b`)
	identifiers := identifierPattern.FindAllString(code, -1)
	
	// Deduplicate
	seen := make(map[string]bool)
	for _, id := range append(symbols, identifiers...) {
		if !seen[id] && len(id) > 2 { // Skip very short identifiers
			seen[id] = true
			symbols = append(symbols, id)
		}
	}
	
	return symbols
}

// GetIndexStats returns statistics about the index
func (bs *BleveSearcher) GetIndexStats() (map[string]interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	docCount, err := bs.index.DocCount()
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"document_count": docCount,
		"index_path":     bs.indexPath,
	}, nil
}