package rag

import (
	"sort"
	"strings"
)

type Reranker struct {
	// Could have configuration for reranking models
}

func NewReranker() *Reranker {
	return &Reranker{}
}

func (r *Reranker) Rerank(candidates []SearchResult, query string, limit int) []SearchResult {
	// Simple reranking based on multiple factors
	queryLower := strings.ToLower(query)
	queryTokens := strings.Fields(queryLower)
	
	for i := range candidates {
		// Boost score based on exact matches
		codeLower := strings.ToLower(candidates[i].Code)
		exactMatchBoost := 0.0
		
		for _, token := range queryTokens {
			if strings.Contains(codeLower, token) {
				exactMatchBoost += 0.1
			}
		}
		
		// Boost for certain file types or patterns
		if strings.HasSuffix(candidates[i].FilePath, ".go") && strings.Contains(queryLower, "go") {
			exactMatchBoost += 0.05
		}
		
		// Apply boost
		candidates[i].Score += float32(exactMatchBoost)
	}
	
	// Sort by new scores
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	
	// Limit results
	if len(candidates) > limit {
		return candidates[:limit]
	}
	
	return candidates
}

func (r *Reranker) CrossEncoderRerank(candidates []SearchResult, query string, limit int) []SearchResult {
	// In a full implementation, this would use a cross-encoder model
	// to rerank results based on query-document pairs
	return r.Rerank(candidates, query, limit)
}