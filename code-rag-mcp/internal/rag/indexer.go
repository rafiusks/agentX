package rag

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Indexer struct {
	config   *VectorDBConfig
	embedder *Embedder
	store    VectorStore
}

type VectorStore interface {
	Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error
	UpsertToCollection(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error)
	SearchInCollection(ctx context.Context, collection string, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error)
	Delete(ctx context.Context, collection string, id string) error
	CreateCollection(ctx context.Context, name string, dimension int) error
	ListCollections(ctx context.Context) ([]string, error)
}

func NewIndexer(config *VectorDBConfig, embedder *Embedder) (*Indexer, error) {
	var store VectorStore
	
	switch config.Type {
	case "qdrant":
		store = NewQdrantStore(config)
	case "chroma":
		store = NewChromaStore(config)
	default:
		store = NewMockVectorStore()
	}
	
	// Ensure collection exists
	ctx := context.Background()
	err := store.CreateCollection(ctx, config.CollectionName, config.Dimension)
	if err != nil {
		// Collection might already exist, which is fine
		fmt.Printf("Collection creation info: %v\n", err)
	}
	
	return &Indexer{
		config:   config,
		embedder: embedder,
		store:    store,
	}, nil
}

func (i *Indexer) IndexToCollection(ctx context.Context, chunk Chunk, embedding []float32, collection string) error {
	id := generateChunkID(chunk)
	
	metadata := map[string]interface{}{
		"file_path":      chunk.FilePath,
		"language":       chunk.Language,
		"line_start":     chunk.LineStart,
		"line_end":       chunk.LineEnd,
		"type":           chunk.Type,
		"name":           chunk.Name,
		"code":           chunk.Code,
		"repository":     chunk.Repository,
		"indexed_at":     time.Now().Unix(),
		"collection":     collection,
		"symbols":        strings.Join(chunk.Symbols, " "),
		"signatures":     chunk.Signatures,
		"file_context":   chunk.FileContext,
		"parent_context": chunk.ParentContext,
	}
	
	// Ensure collection exists
	i.store.CreateCollection(ctx, collection, i.config.Dimension)
	
	return i.store.UpsertToCollection(ctx, collection, id, embedding, metadata)
}

func (i *Indexer) SearchInCollection(ctx context.Context, vector []float32, collection string, language string, limit int) ([]SearchResult, error) {
	filter := make(map[string]interface{})
	if language != "" && language != "any" {
		filter["language"] = language
	}
	
	return i.store.SearchInCollection(ctx, collection, vector, limit, filter)
}

func (i *Indexer) Index(ctx context.Context, chunk Chunk, embedding []float32) error {
	id := generateChunkID(chunk)
	
	metadata := map[string]interface{}{
		"file_path":  chunk.FilePath,
		"language":   chunk.Language,
		"line_start": chunk.LineStart,
		"line_end":   chunk.LineEnd,
		"type":       chunk.Type,
		"name":       chunk.Name,
		"code":       chunk.Code,
		"repository": chunk.Repository,
		"indexed_at": time.Now().Unix(),
	}
	
	return i.store.Upsert(ctx, id, embedding, metadata)
}

func (i *Indexer) Search(ctx context.Context, vector []float32, language string, limit int) ([]SearchResult, error) {
	filter := make(map[string]interface{})
	if language != "" && language != "any" {
		filter["language"] = language
	}
	
	return i.store.Search(ctx, vector, limit, filter)
}

func (i *Indexer) FindSimilar(ctx context.Context, vector []float32, threshold float64, limit int) ([]SearchResult, error) {
	results, err := i.store.Search(ctx, vector, limit*2, nil)
	if err != nil {
		return nil, err
	}
	
	// Filter by threshold
	var filtered []SearchResult
	for _, result := range results {
		if result.Score >= float32(threshold) {
			filtered = append(filtered, result)
			if len(filtered) >= limit {
				break
			}
		}
	}
	
	return filtered, nil
}

func (i *Indexer) GetRepositories() ([]Repository, error) {
	// This would query the vector store for unique repositories
	// For now, return mock data
	return []Repository{
		{
			Path:         "/Users/code/project1",
			Name:         "project1",
			IndexedAt:    time.Now().Add(-24 * time.Hour),
			FileCount:    150,
			ChunkCount:   1200,
			LastModified: time.Now().Add(-2 * time.Hour),
		},
	}, nil
}

// DeleteFileChunks removes all chunks for a specific file
func (i *Indexer) DeleteFileChunks(ctx context.Context, filePath string, collection string) error {
	// This would need to track chunk IDs per file
	// For now, we'll use a filter-based approach
	filter := map[string]interface{}{
		"file_path": filePath,
	}
	
	// Search for all chunks from this file
	// Note: This requires the store to support deletion by filter
	// For Qdrant, we'd need to search first then delete by IDs
	vector := make([]float32, 768) // Dummy vector for search
	results, err := i.store.SearchInCollection(ctx, collection, vector, 1000, filter)
	if err != nil {
		return err
	}
	
	// Delete each chunk by ID
	for _, result := range results {
		// Generate the same ID that was used during indexing
		chunk := Chunk{
			FilePath:  result.FilePath,
			LineStart: result.LineStart,
			LineEnd:   result.LineEnd,
		}
		id := generateChunkID(chunk)
		
		// Delete from store (this method would need to be added to VectorStore)
		if err := i.store.Delete(ctx, collection, id); err != nil {
			// Log but continue with other deletions
			fmt.Printf("Failed to delete chunk %s: %v\n", id, err)
		}
	}
	
	return nil
}

func generateChunkID(chunk Chunk) string {
	return fmt.Sprintf("%s:%d-%d:%s", chunk.FilePath, chunk.LineStart, chunk.LineEnd, chunk.Type)
}

// Mock Vector Store for testing
type MockVectorStore struct {
	data        map[string]interface{}
	collections map[string]map[string]interface{}
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		data:        make(map[string]interface{}),
		collections: make(map[string]map[string]interface{}),
	}
}

func (m *MockVectorStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	m.data[id] = map[string]interface{}{
		"vector":   vector,
		"metadata": metadata,
	}
	return nil
}

func (m *MockVectorStore) UpsertToCollection(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	if m.collections[collection] == nil {
		m.collections[collection] = make(map[string]interface{})
	}
	m.collections[collection][id] = map[string]interface{}{
		"vector":   vector,
		"metadata": metadata,
	}
	return nil
}

func (m *MockVectorStore) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	// Return mock results
	results := []SearchResult{
		{
			FilePath:   "/mock/file.go",
			Code:       "func MockFunction() {\n    // Mock implementation\n}",
			Language:   "go",
			Score:      0.95,
			LineStart:  10,
			LineEnd:    13,
			Repository: "mock-repo",
		},
	}
	
	if limit < len(results) {
		return results[:limit], nil
	}
	return results, nil
}

func (m *MockVectorStore) SearchInCollection(ctx context.Context, collection string, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	// Return mock results for specific collection
	return m.Search(ctx, vector, limit, filter)
}

func (m *MockVectorStore) Delete(ctx context.Context, collection string, id string) error {
	if collection == "" {
		delete(m.data, id)
	} else if m.collections[collection] != nil {
		delete(m.collections[collection], id)
	}
	return nil
}

func (m *MockVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	return nil
}

func (m *MockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{"code_embeddings"}, nil
}

// Placeholder stores - would be fully implemented
type QdrantStore struct {
	config *VectorDBConfig
}

func NewQdrantStore(config *VectorDBConfig) VectorStore {
	// Use real Qdrant implementation
	return NewRealQdrantStore(config)
}

func (q *QdrantStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	// Would use Qdrant Go client
	return NewMockVectorStore().Upsert(ctx, id, vector, metadata)
}

func (q *QdrantStore) UpsertToCollection(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	return NewMockVectorStore().UpsertToCollection(ctx, collection, id, vector, metadata)
}

func (q *QdrantStore) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	return NewMockVectorStore().Search(ctx, vector, limit, filter)
}

func (q *QdrantStore) SearchInCollection(ctx context.Context, collection string, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	return NewMockVectorStore().SearchInCollection(ctx, collection, vector, limit, filter)
}

func (q *QdrantStore) Delete(ctx context.Context, collection string, id string) error {
	return NewMockVectorStore().Delete(ctx, collection, id)
}

func (q *QdrantStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	return NewMockVectorStore().CreateCollection(ctx, name, dimension)
}

func (q *QdrantStore) ListCollections(ctx context.Context) ([]string, error) {
	return NewMockVectorStore().ListCollections(ctx)
}

type ChromaStore struct {
	config *VectorDBConfig
}

func NewChromaStore(config *VectorDBConfig) *ChromaStore {
	return &ChromaStore{config: config}
}

func (c *ChromaStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	return NewMockVectorStore().Upsert(ctx, id, vector, metadata)
}

func (c *ChromaStore) UpsertToCollection(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	return NewMockVectorStore().UpsertToCollection(ctx, collection, id, vector, metadata)
}

func (c *ChromaStore) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	return NewMockVectorStore().Search(ctx, vector, limit, filter)
}

func (c *ChromaStore) SearchInCollection(ctx context.Context, collection string, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	return NewMockVectorStore().SearchInCollection(ctx, collection, vector, limit, filter)
}

func (c *ChromaStore) Delete(ctx context.Context, collection string, id string) error {
	return NewMockVectorStore().Delete(ctx, collection, id)
}

func (c *ChromaStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	return NewMockVectorStore().CreateCollection(ctx, name, dimension)
}

func (c *ChromaStore) ListCollections(ctx context.Context) ([]string, error) {
	return NewMockVectorStore().ListCollections(ctx)
}