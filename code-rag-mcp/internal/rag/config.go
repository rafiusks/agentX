package rag

import "time"

type Config struct {
	EmbeddingConfig *EmbeddingConfig
	ChunkingConfig  *ChunkingConfig
	VectorDBConfig  *VectorDBConfig
	Silent          bool // Silent mode - no stdout output for MCP
}

type EmbeddingConfig struct {
	Provider    string // "codet5", "codebert", "openai", "local"
	Model       string
	APIKey      string
	APIEndpoint string
	CacheSize   int
	BatchSize   int
}

type ChunkingConfig struct {
	Strategy      string // "ast_aware", "sliding_window", "semantic"
	MaxChunkSize  int
	ChunkOverlap  int
	MinChunkSize  int
	Languages     []string
}

type VectorDBConfig struct {
	Type           string // "qdrant", "chroma", "weaviate"
	URL            string
	CollectionName string
	Dimension      int
	Distance       string // "cosine", "euclidean", "dot"
	Timeout        time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		EmbeddingConfig: &EmbeddingConfig{
			Provider:  "codet5",
			Model:     "Salesforce/codet5p-110m-embedding",
			CacheSize: 10000,
			BatchSize: 32,
		},
		ChunkingConfig: &ChunkingConfig{
			Strategy:     "ast_aware",
			MaxChunkSize: 512,
			ChunkOverlap: 128,
			MinChunkSize: 50,
			Languages:    []string{"go", "javascript", "typescript", "python", "rust"},
		},
		VectorDBConfig: &VectorDBConfig{
			Type:           "qdrant",
			URL:            "http://localhost:6333",
			CollectionName: "code_embeddings",
			Dimension:      768,
			Distance:       "cosine",
			Timeout:        30 * time.Second,
		},
	}
}