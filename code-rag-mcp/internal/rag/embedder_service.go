package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ServiceEmbeddingProvider uses the Python embedding service
type ServiceEmbeddingProvider struct {
	serviceURL string
	httpClient *http.Client
	model      string
}

// NewServiceEmbeddingProvider creates a provider that calls the Python service
func NewServiceEmbeddingProvider(config *EmbeddingConfig) *ServiceEmbeddingProvider {
	serviceURL := os.Getenv("EMBEDDING_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8001"
	}
	
	model := config.Model
	if model == "" {
		model = "microsoft/codebert-base"
	}
	
	return &ServiceEmbeddingProvider{
		serviceURL: serviceURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: model,
	}
}

type embeddingServiceRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model,omitempty"`
}

type embeddingServiceResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimension  int         `json:"dimension"`
}

func (p *ServiceEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

func (p *ServiceEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Check if service is available
	healthURL := fmt.Sprintf("%s/health", p.serviceURL)
	healthReq, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	healthResp, err := p.httpClient.Do(healthReq)
	if err != nil {
		// Service not available, fall back to local embeddings
		return p.fallbackEmbeddings(texts), nil
	}
	healthResp.Body.Close()
	
	// Prepare request
	reqBody := embeddingServiceRequest{
		Texts: texts,
		Model: p.model,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Call embedding service
	url := fmt.Sprintf("%s/embed", p.serviceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Fall back to local embeddings if service fails
		return p.fallbackEmbeddings(texts), nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Fall back to local embeddings
		fmt.Printf("Embedding service error: %s, falling back to local\n", body)
		return p.fallbackEmbeddings(texts), nil
	}
	
	// Parse response
	var result embeddingServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Embeddings, nil
}

// fallbackEmbeddings generates simple embeddings when service is unavailable
func (p *ServiceEmbeddingProvider) fallbackEmbeddings(texts []string) [][]float32 {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = generateSimpleEmbedding(text)
	}
	return embeddings
}

// generateSimpleEmbedding creates a basic embedding for fallback
func generateSimpleEmbedding(text string) []float32 {
	const dim = 768
	embedding := make([]float32, dim)
	
	// Simple hash-based embedding
	hash := 0
	for _, ch := range text {
		hash = (hash*31 + int(ch)) % 1000000007
	}
	
	for i := 0; i < dim; i++ {
		seed := hash * (i + 1)
		embedding[i] = float32((seed%2000)-1000) / 1000.0
		if i < len(text) {
			embedding[i] += float32(text[i]) / 255.0
		}
	}
	
	// Normalize
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	if sum > 0 {
		norm := 1.0 / float32(sqrt(float64(sum)))
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	
	return embedding
}

// sqrt is defined in embedder_codet5.go