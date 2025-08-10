package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RealQdrantStore implements VectorStore using actual Qdrant API
type RealQdrantStore struct {
	config     *VectorDBConfig
	httpClient *http.Client
	baseURL    string
}

func NewRealQdrantStore(config *VectorDBConfig) *RealQdrantStore {
	return &RealQdrantStore{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL: config.URL,
	}
}

func (q *RealQdrantStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/%s", q.baseURL, name)
	req, _ := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	resp, err := q.httpClient.Do(req)
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return nil // Collection already exists
	}
	
	// Create collection
	createURL := fmt.Sprintf("%s/collections/%s", q.baseURL, name)
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}
	
	jsonData, _ := json.Marshal(payload)
	req, err = http.NewRequestWithContext(ctx, "PUT", createURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err = q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", body)
	}
	
	// Wait for collection to be ready
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (q *RealQdrantStore) UpsertToCollection(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Ensure collection exists
	q.CreateCollection(ctx, collection, len(vector))
	
	url := fmt.Sprintf("%s/collections/%s/points", q.baseURL, collection)
	
	// Generate numeric ID from string ID
	numericID := hashStringToUint64(id)
	
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      numericID,
				"vector":  vector,
				"payload": metadata,
			},
		},
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert: %s", body)
	}
	
	return nil
}

func (q *RealQdrantStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	return q.UpsertToCollection(ctx, q.config.CollectionName, id, vector, metadata)
}

func (q *RealQdrantStore) SearchInCollection(ctx context.Context, collection string, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, collection)
	
	payload := map[string]interface{}{
		"vector": vector,
		"limit":  limit,
		"with_payload": true,
	}
	
	// Add filter if provided
	if len(filter) > 0 {
		must := []map[string]interface{}{}
		for key, value := range filter {
			must = append(must, map[string]interface{}{
				"key":   key,
				"match": map[string]interface{}{"value": value},
			})
		}
		payload["filter"] = map[string]interface{}{
			"must": must,
		}
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", body)
	}
	
	var searchResp struct {
		Result []struct {
			ID      uint64                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}
	
	results := make([]SearchResult, 0, len(searchResp.Result))
	for _, r := range searchResp.Result {
		result := SearchResult{
			Score: r.Score,
		}
		
		// Extract fields from payload
		if filePath, ok := r.Payload["file_path"].(string); ok {
			result.FilePath = filePath
		}
		if code, ok := r.Payload["code"].(string); ok {
			result.Code = code
		}
		if lang, ok := r.Payload["language"].(string); ok {
			result.Language = lang
		}
		if lineStart, ok := r.Payload["line_start"].(float64); ok {
			result.LineStart = int(lineStart)
		}
		if lineEnd, ok := r.Payload["line_end"].(float64); ok {
			result.LineEnd = int(lineEnd)
		}
		if repo, ok := r.Payload["repository"].(string); ok {
			result.Repository = repo
		}
		if typeStr, ok := r.Payload["type"].(string); ok {
			result.Type = typeStr
		}
		if name, ok := r.Payload["name"].(string); ok {
			result.Name = name
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

func (q *RealQdrantStore) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	return q.SearchInCollection(ctx, q.config.CollectionName, vector, limit, filter)
}

func (q *RealQdrantStore) Delete(ctx context.Context, collection string, id string) error {
	if collection == "" {
		collection = q.config.CollectionName
	}
	url := fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, collection)
	
	numericID := hashStringToUint64(id)
	payload := map[string]interface{}{
		"points": []uint64{numericID},
	}
	
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	
	return nil
}

func (q *RealQdrantStore) ListCollections(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/collections", q.baseURL)
	
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var collectionsResp struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&collectionsResp); err != nil {
		return nil, err
	}
	
	names := make([]string, len(collectionsResp.Result.Collections))
	for i, c := range collectionsResp.Result.Collections {
		names[i] = c.Name
	}
	
	return names, nil
}

// hashStringToUint64 converts string ID to uint64 for Qdrant
func hashStringToUint64(s string) uint64 {
	var h uint64 = 5381
	for _, c := range s {
		h = ((h << 5) + h) + uint64(c)
	}
	return h
}