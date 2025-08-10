package rag

import (
	"context"
	"sort"
)

type Retriever struct {
	indexer  *Indexer
	embedder *Embedder
}

func NewRetriever(indexer *Indexer, embedder *Embedder) *Retriever {
	return &Retriever{
		indexer:  indexer,
		embedder: embedder,
	}
}

func (r *Retriever) RetrieveFromCollection(ctx context.Context, queryVector []float32, collection string, language string, limit int) ([]SearchResult, error) {
	// Perform vector search in specific collection
	results, err := r.indexer.SearchInCollection(ctx, queryVector, collection, language, limit)
	if err != nil {
		return nil, err
	}
	
	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results, nil
}

func (r *Retriever) Retrieve(ctx context.Context, queryVector []float32, language string, limit int) ([]SearchResult, error) {
	// Perform vector search
	results, err := r.indexer.Search(ctx, queryVector, language, limit)
	if err != nil {
		return nil, err
	}
	
	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results, nil
}

func (r *Retriever) HybridRetrieve(ctx context.Context, query string, language string, limit int) ([]SearchResult, error) {
	// Get semantic results
	queryVector, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	
	semanticResults, err := r.indexer.Search(ctx, queryVector, language, limit)
	if err != nil {
		return nil, err
	}
	
	// In a full implementation, we would also do keyword search here
	// and merge the results
	
	return semanticResults, nil
}