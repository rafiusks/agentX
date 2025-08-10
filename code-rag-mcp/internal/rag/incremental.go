package rag

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/rafael/code-rag-mcp/internal/git"
	"github.com/rafael/code-rag-mcp/internal/watcher"
	"github.com/sirupsen/logrus"
)

// IncrementalIndexer handles incremental updates to the index
type IncrementalIndexer struct {
	engine       *Engine
	workerPool   chan struct{} // Limits concurrent indexing
	logger       *logrus.Logger
	mu           sync.Mutex
	processing   map[string]bool // Track files being processed
	lastIndexed  map[string]time.Time
}

// NewIncrementalIndexer creates a new incremental indexer
func NewIncrementalIndexer(engine *Engine, maxWorkers int) *IncrementalIndexer {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	return &IncrementalIndexer{
		engine:      engine,
		workerPool:  make(chan struct{}, maxWorkers),
		logger:      logrus.New(),
		processing:  make(map[string]bool),
		lastIndexed: make(map[string]time.Time),
	}
}

// StartWatching starts file watching for a repository
func (e *Engine) StartWatching(ctx context.Context, repoPath string, collection string) error {
	// Initialize git tracker if in a git repo
	gitTracker, err := git.NewGitTracker(repoPath)
	if err != nil {
		e.stats.MostSearchedTerms = append(e.stats.MostSearchedTerms, 
			fmt.Sprintf("Warning: Not a git repository: %v", err))
	} else {
		e.gitTracker = gitTracker
	}

	// Initialize hash store
	hashStorePath := filepath.Join(".code-rag", "hashes.db")
	hashStore, err := watcher.NewHashStore(hashStorePath)
	if err != nil {
		return fmt.Errorf("failed to create hash store: %w", err)
	}
	e.hashStore = hashStore

	// Load existing hashes
	existingHashes, _ := hashStore.GetAllHashes()

	// Create file watcher
	watcherConfig := watcher.Config{
		RootPath:     repoPath,
		Handler:      e.createChangeHandler(collection),
		DebounceTime: 500 * time.Millisecond,
		BatchSize:    50,
		Logger:       logrus.StandardLogger(),
	}

	fileWatcher, err := watcher.NewWatcher(watcherConfig)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Set existing hashes
	if len(existingHashes) > 0 {
		fileWatcher.SetFileHashes(existingHashes)
	}

	e.fileWatcher = fileWatcher

	// Start watching
	if err := fileWatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	fmt.Printf("Started watching %s for changes (collection: %s)\n", repoPath, collection)
	return nil
}

// createChangeHandler creates a handler for file changes
func (e *Engine) createChangeHandler(collection string) watcher.ChangeHandler {
	indexer := NewIncrementalIndexer(e, 4)
	
	return func(changes []watcher.FileChange) error {
		ctx := context.Background()
		
		// Group changes by type
		var toIndex, toDelete []string
		
		for _, change := range changes {
			switch change.Type {
			case watcher.ChangeTypeCreate, watcher.ChangeTypeModify:
				toIndex = append(toIndex, change.Path)
			case watcher.ChangeTypeDelete:
				toDelete = append(toDelete, change.Path)
			case watcher.ChangeTypeRename:
				// Delete old path, index new path
				if change.OldPath != "" {
					toDelete = append(toDelete, change.OldPath)
				}
				toIndex = append(toIndex, change.Path)
			}
		}

		// Process deletions first
		if len(toDelete) > 0 {
			if err := indexer.DeleteFiles(ctx, toDelete, collection); err != nil {
				logrus.Errorf("Failed to delete files from index: %v", err)
			}
		}

		// Process additions/modifications
		if len(toIndex) > 0 {
			if err := indexer.IndexFiles(ctx, toIndex, collection); err != nil {
				logrus.Errorf("Failed to index files: %v", err)
			}
		}

		// Update hash store
		if e.hashStore != nil {
			for _, change := range changes {
				if change.Type == watcher.ChangeTypeDelete {
					e.hashStore.DeleteHash(change.Path)
				} else if change.ContentHash != "" {
					e.hashStore.SetHash(change.Path, change.ContentHash, change.Timestamp, 0)
				}
			}
		}

		// Invalidate caches
		e.invalidateCachesForFiles(append(toIndex, toDelete...))

		return nil
	}
}

// IndexFiles incrementally indexes a set of files
func (ii *IncrementalIndexer) IndexFiles(ctx context.Context, files []string, collection string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(files))

	for _, file := range files {
		// Check if already processing
		ii.mu.Lock()
		if ii.processing[file] {
			ii.mu.Unlock()
			continue
		}
		ii.processing[file] = true
		ii.mu.Unlock()

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			defer func() {
				ii.mu.Lock()
				delete(ii.processing, filePath)
				ii.mu.Unlock()
			}()

			// Acquire worker slot
			ii.workerPool <- struct{}{}
			defer func() { <-ii.workerPool }()

			// Index the file
			if err := ii.indexSingleFile(ctx, filePath, collection); err != nil {
				errors <- fmt.Errorf("failed to index %s: %w", filePath, err)
			} else {
				ii.logger.Debugf("Indexed file: %s", filePath)
			}
		}(file)
	}

	wg.Wait()
	close(errors)

	// Collect errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("indexing errors: %v", allErrors)
	}

	ii.logger.Infof("Successfully indexed %d files", len(files))
	return nil
}

// indexSingleFile indexes a single file
func (ii *IncrementalIndexer) indexSingleFile(ctx context.Context, filePath string, collection string) error {
	// First, delete old chunks for this file
	if err := ii.deleteFileChunks(ctx, filePath, collection); err != nil {
		ii.logger.Warnf("Failed to delete old chunks for %s: %v", filePath, err)
	}

	// Chunk the file
	var chunks []Chunk
	var err error
	if ii.engine.useImproved && ii.engine.improvedChunker != nil {
		chunks, err = ii.engine.improvedChunker.ChunkFile(filePath)
	} else {
		chunks, err = ii.engine.chunker.ChunkFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("failed to chunk file: %w", err)
	}

	// Index each chunk
	for _, chunk := range chunks {
		// Generate embedding
		embedding, err := ii.engine.embedder.EmbedCode(ctx, chunk.Code, chunk.Language)
		if err != nil {
			ii.logger.Warnf("Failed to embed chunk from %s: %v", filePath, err)
			continue
		}

		// Index to vector store
		if err := ii.engine.indexer.IndexToCollection(ctx, chunk, embedding, collection); err != nil {
			ii.logger.Warnf("Failed to index chunk to vector store: %v", err)
			continue
		}

		// Index to Bleve
		if ii.engine.bleveSearcher != nil {
			if err := ii.engine.bleveSearcher.IndexChunk(chunk); err != nil {
				ii.logger.Warnf("Failed to index chunk to Bleve: %v", err)
			}
		}
	}

	// Update last indexed time
	ii.mu.Lock()
	ii.lastIndexed[filePath] = time.Now()
	ii.mu.Unlock()

	return nil
}

// DeleteFiles removes files from the index
func (ii *IncrementalIndexer) DeleteFiles(ctx context.Context, files []string, collection string) error {
	for _, file := range files {
		if err := ii.deleteFileChunks(ctx, file, collection); err != nil {
			ii.logger.Errorf("Failed to delete %s from index: %v", file, err)
		} else {
			ii.logger.Debugf("Deleted file from index: %s", file)
		}
	}
	return nil
}

// deleteFileChunks removes all chunks for a file from the index
func (ii *IncrementalIndexer) deleteFileChunks(ctx context.Context, filePath string, collection string) error {
	// Delete from vector store
	if err := ii.engine.indexer.DeleteFileChunks(ctx, filePath, collection); err != nil {
		return fmt.Errorf("failed to delete from vector store: %w", err)
	}

	// Delete from Bleve
	if ii.engine.bleveSearcher != nil {
		if err := ii.engine.bleveSearcher.DeleteFile(filePath); err != nil {
			return fmt.Errorf("failed to delete from Bleve: %w", err)
		}
	}

	return nil
}

// invalidateCachesForFiles invalidates all caches for the given files
func (e *Engine) invalidateCachesForFiles(files []string) {
	e.changeMu.Lock()
	defer e.changeMu.Unlock()

	// Clear search cache (could be more selective)
	if e.searchCache != nil {
		e.searchCache.Clear()
	}

	// Clear chunk cache for modified files
	if e.chunkCache != nil {
		for range files {
			// This would need a new method in FileChunkCache
			// For now, clear all
			e.chunkCache.Clear()
			break
		}
	}

	// Clear embedding cache for modified files
	// This is harder without tracking which embeddings came from which files
	// For now, we'll leave embeddings cached as they're content-based
}

// StopWatching stops file watching
func (e *Engine) StopWatching() error {
	if e.fileWatcher != nil {
		// Save current hashes
		if e.hashStore != nil {
			hashes := e.fileWatcher.GetFileHashes()
			for path, hash := range hashes {
				e.hashStore.SetHash(path, hash, time.Now(), 0)
			}
			e.hashStore.Close()
		}

		// Stop watcher
		if err := e.fileWatcher.Stop(); err != nil {
			return err
		}
		e.fileWatcher = nil
	}
	return nil
}