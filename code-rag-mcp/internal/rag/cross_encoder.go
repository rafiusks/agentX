package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CrossEncoderClient interfaces with the cross-encoder reranking service
type CrossEncoderClient struct {
	serviceURL string
	httpClient *http.Client
}

// NewCrossEncoderClient creates a new cross-encoder client
func NewCrossEncoderClient(serviceURL string) *CrossEncoderClient {
	if serviceURL == "" {
		serviceURL = "http://localhost:8002"
	}
	
	return &CrossEncoderClient{
		serviceURL: serviceURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CodeCandidate represents a candidate for reranking
type CodeCandidate struct {
	Code     string  `json:"code"`
	FilePath string  `json:"file_path"`
	Score    float32 `json:"score"`
	Language string  `json:"language"`
	Type     string  `json:"type"`
	Name     string  `json:"name"`
}

// RerankRequest is the request structure for reranking
type RerankRequest struct {
	Query      string          `json:"query"`
	Candidates []CodeCandidate `json:"candidates"`
	TopK       int             `json:"top_k"`
	UseCache   bool            `json:"use_cache"`
}

// RerankResult represents a reranked result
type RerankResult struct {
	FilePath           string  `json:"file_path"`
	Code               string  `json:"code"`
	Language           string  `json:"language"`
	Type               string  `json:"type"`
	Name               string  `json:"name"`
	OriginalScore      float32 `json:"original_score"`
	CrossEncoderScore  float32 `json:"cross_encoder_score"`
	FinalScore         float32 `json:"final_score"`
}

// RerankResponse is the response from the reranking service
type RerankResponse struct {
	Results      []RerankResult `json:"results"`
	RerankTimeMs float64        `json:"rerank_time_ms"`
	CacheHit     bool           `json:"cache_hit"`
}

// Rerank performs cross-encoder reranking on search results
func (c *CrossEncoderClient) Rerank(ctx context.Context, query string, results []SearchResult, topK int) ([]SearchResult, error) {
	// Convert SearchResults to CodeCandidates
	candidates := make([]CodeCandidate, len(results))
	for i, result := range results {
		candidates[i] = CodeCandidate{
			Code:     result.Code,
			FilePath: result.FilePath,
			Score:    result.Score,
			Language: result.Language,
			Type:     result.Type,
			Name:     result.Name,
		}
	}
	
	// Create request
	request := RerankRequest{
		Query:      query,
		Candidates: candidates,
		TopK:       topK,
		UseCache:   true,
	}
	
	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.serviceURL+"/rerank", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If service is unavailable, return original results
		fmt.Printf("Cross-encoder service unavailable: %v\n", err)
		return results, nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Return original results if reranking fails
		fmt.Printf("Cross-encoder reranking failed with status %d\n", resp.StatusCode)
		return results, nil
	}
	
	// Parse response
	var rerankResp RerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Convert back to SearchResults
	rerankedResults := make([]SearchResult, len(rerankResp.Results))
	for i, result := range rerankResp.Results {
		rerankedResults[i] = SearchResult{
			FilePath:   result.FilePath,
			Code:       result.Code,
			Language:   result.Language,
			Score:      result.FinalScore,
			Type:       result.Type,
			Name:       result.Name,
			// Note: LineStart and LineEnd are lost in reranking
			// You might want to preserve them in the candidate structure
		}
	}
	
	if rerankResp.CacheHit {
		fmt.Printf("Cross-encoder cache hit, reranking took %.2fms\n", rerankResp.RerankTimeMs)
	} else {
		fmt.Printf("Cross-encoder reranked %d results in %.2fms\n", len(candidates), rerankResp.RerankTimeMs)
	}
	
	return rerankedResults, nil
}

// HealthCheck checks if the cross-encoder service is healthy
func (c *CrossEncoderClient) HealthCheck(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.serviceURL+"/health", nil)
	if err != nil {
		return false, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK, nil
}