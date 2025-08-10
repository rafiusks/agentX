package rag

import (
	"context"
	"fmt"
)

type Embedder struct {
	config   *EmbeddingConfig
	cache    *EmbeddingCache
	provider EmbeddingProvider
}

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}


func NewEmbedder(config *EmbeddingConfig) (*Embedder, error) {
	var provider EmbeddingProvider
	
	switch config.Provider {
	case "codet5":
		// Use the embedding service which runs CodeBERT/CodeT5
		provider = NewServiceEmbeddingProvider(config)
	case "service":
		provider = NewServiceEmbeddingProvider(config)
	case "openai":
		provider = NewOpenAIProvider(config)
	case "local":
		provider = NewLocalProvider(config)
	default:
		// Default to service provider
		provider = NewServiceEmbeddingProvider(config)
	}
	
	// Create embedding cache with configured size
	cache := NewEmbeddingCache(config.CacheSize)
	
	return &Embedder{
		config:   config,
		cache:    cache,
		provider: provider,
	}, nil
}

func (e *Embedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	// Check cache first
	if cached, found := e.cache.Get(query); found {
		return cached, nil
	}
	
	// Generate embedding
	embedding, err := e.provider.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	// Cache the result
	e.cache.Set(query, embedding)
	
	return embedding, nil
}

func (e *Embedder) EmbedCode(ctx context.Context, code string, language string) ([]float32, error) {
	// Prepare code for embedding (add language context)
	text := fmt.Sprintf("[%s]\n%s", language, code)
	
	// Check cache
	if cached, found := e.cache.Get(text); found {
		return cached, nil
	}
	
	// Generate embedding
	embedding, err := e.provider.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to embed code: %w", err)
	}
	
	// Cache the result
	e.cache.Set(text, embedding)
	
	return embedding, nil
}

func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.provider.EmbedBatch(ctx, texts)
}

// Cache methods are now in cache.go

// Mock provider for testing
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return a mock 768-dimensional embedding
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 768.0
	}
	return embedding, nil
}

func (m *MockProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i], _ = m.Embed(ctx, texts[i])
	}
	return embeddings, nil
}

// Placeholder providers - these would be fully implemented
type CodeT5Provider struct {
	config *EmbeddingConfig
}

func NewCodeT5Provider(config *EmbeddingConfig) EmbeddingProvider {
	// Use the real implementation
	return NewRealCodeT5Provider(config)
}

type OpenAIProvider struct {
	config *EmbeddingConfig
}

func NewOpenAIProvider(config *EmbeddingConfig) *OpenAIProvider {
	return &OpenAIProvider{config: config}
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Would call OpenAI API
	return NewMockProvider().Embed(ctx, text)
}

func (p *OpenAIProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return NewMockProvider().EmbedBatch(ctx, texts)
}

type LocalProvider struct {
	config *EmbeddingConfig
}

func NewLocalProvider(config *EmbeddingConfig) *LocalProvider {
	return &LocalProvider{config: config}
}

func (p *LocalProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Would use local model
	return NewMockProvider().Embed(ctx, text)
}

func (p *LocalProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return NewMockProvider().EmbedBatch(ctx, texts)
}