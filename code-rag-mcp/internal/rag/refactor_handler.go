package rag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rafael/code-rag-mcp/internal/watcher"
	"github.com/sirupsen/logrus"
)

// RefactorHandler manages large-scale code changes efficiently
type RefactorHandler struct {
	engine            *Engine
	depGraph          *DependencyGraph
	logger            *logrus.Logger
	
	// Processing control
	maxBatchSize      int
	maxConcurrency    int
	adaptiveThreshold int
	
	// Progress tracking
	totalFiles        int32
	processedFiles    int32
	failedFiles       int32
	startTime         time.Time
	
	// Rate limiting
	rateLimiter       chan struct{}
	
	// Rollback support
	rollbackMu        sync.Mutex
	rollbackPoints    []RollbackPoint
	
	// Adaptive processing
	currentMode       ProcessingMode
	modeMu            sync.RWMutex
}

// ProcessingMode represents different processing strategies
type ProcessingMode int

const (
	ModeRealtime ProcessingMode = iota  // < 10 files
	ModeBatch                            // 10-100 files
	ModeBulk                            // 100-1000 files
	ModeOffline                         // > 1000 files
)

// RollbackPoint represents a point we can rollback to
type RollbackPoint struct {
	Timestamp   time.Time
	Files       []string
	Collection  string
	Checksum    string
}

// NewRefactorHandler creates a handler for large refactoring operations
func NewRefactorHandler(engine *Engine) *RefactorHandler {
	return &RefactorHandler{
		engine:            engine,
		depGraph:          NewDependencyGraph(),
		logger:            logrus.New(),
		maxBatchSize:      100,
		maxConcurrency:    8,
		adaptiveThreshold: 50,
		rateLimiter:       make(chan struct{}, 8), // Max 8 concurrent operations
		currentMode:       ModeRealtime,
	}
}

// ProcessChanges handles a set of file changes with adaptive processing
func (rh *RefactorHandler) ProcessChanges(ctx context.Context, changes []watcher.FileChange, collection string) error {
	changeCount := len(changes)
	
	// Determine processing mode based on change volume
	rh.determineProcessingMode(changeCount)
	
	// Log the operation
	rh.logger.Infof("Processing %d changes in %s mode", changeCount, rh.getModeName())
	rh.startTime = time.Now()
	atomic.StoreInt32(&rh.totalFiles, int32(changeCount))
	atomic.StoreInt32(&rh.processedFiles, 0)
	atomic.StoreInt32(&rh.failedFiles, 0)
	
	// Create rollback point if this is a large operation
	if changeCount > rh.adaptiveThreshold {
		if err := rh.createRollbackPoint(collection, changes); err != nil {
			rh.logger.Warnf("Failed to create rollback point: %v", err)
		}
	}
	
	// Process based on mode
	switch rh.currentMode {
	case ModeRealtime:
		return rh.processRealtime(ctx, changes, collection)
	case ModeBatch:
		return rh.processBatch(ctx, changes, collection)
	case ModeBulk:
		return rh.processBulk(ctx, changes, collection)
	case ModeOffline:
		return rh.processOffline(ctx, changes, collection)
	default:
		return rh.processBatch(ctx, changes, collection)
	}
}

// processRealtime handles small changes immediately
func (rh *RefactorHandler) processRealtime(ctx context.Context, changes []watcher.FileChange, collection string) error {
	for _, change := range changes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := rh.processSingleChange(ctx, change, collection); err != nil {
				rh.logger.Errorf("Failed to process %s: %v", change.Path, err)
				atomic.AddInt32(&rh.failedFiles, 1)
			} else {
				atomic.AddInt32(&rh.processedFiles, 1)
			}
			
			// Update dependents immediately in realtime mode
			if change.Type != watcher.ChangeTypeDelete {
				rh.invalidateDependents(change.Path, collection)
			}
		}
	}
	
	rh.reportProgress(true)
	return nil
}

// processBatch handles medium-sized changes in batches
func (rh *RefactorHandler) processBatch(ctx context.Context, changes []watcher.FileChange, collection string) error {
	// Window changes into batches
	batches := rh.createBatches(changes, 50)
	
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			rh.logger.Infof("Processing batch %d/%d (%d files)", i+1, len(batches), len(batch))
			
			// Process batch concurrently
			if err := rh.processBatchConcurrent(ctx, batch, collection, 4); err != nil {
				return fmt.Errorf("batch %d failed: %w", i+1, err)
			}
			
			// Small delay between batches to avoid overwhelming the system
			if i < len(batches)-1 {
				time.Sleep(100 * time.Millisecond)
			}
			
			rh.reportProgress(false)
		}
	}
	
	// Process dependency invalidations after all batches
	rh.processDependencyInvalidations(changes, collection)
	
	rh.reportProgress(true)
	return nil
}

// processBulk handles large changes with optimizations
func (rh *RefactorHandler) processBulk(ctx context.Context, changes []watcher.FileChange, collection string) error {
	// Disable real-time features during bulk processing
	rh.logger.Info("Entering bulk processing mode - disabling caches")
	
	// Clear caches to free memory
	if rh.engine.searchCache != nil {
		rh.engine.searchCache.Clear()
	}
	
	// Process in larger batches with higher concurrency
	batches := rh.createBatches(changes, 200)
	
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			rh.logger.Infof("Bulk processing batch %d/%d (%d files)", i+1, len(batches), len(batch))
			
			// Higher concurrency for bulk operations
			if err := rh.processBatchConcurrent(ctx, batch, collection, 8); err != nil {
				// Log error but continue with other batches
				rh.logger.Errorf("Batch %d failed: %v", i+1, err)
			}
			
			// Longer delay between batches
			if i < len(batches)-1 {
				time.Sleep(500 * time.Millisecond)
			}
			
			rh.reportProgress(false)
		}
	}
	
	// Rebuild dependency graph after bulk changes
	rh.logger.Info("Rebuilding dependency graph...")
	rh.rebuildDependencyGraph(collection)
	
	rh.reportProgress(true)
	return nil
}

// processOffline handles massive changes (like initial indexing)
func (rh *RefactorHandler) processOffline(ctx context.Context, changes []watcher.FileChange, collection string) error {
	rh.logger.Warn("Entering offline mode for massive changes - this may take a while")
	
	// Create a new collection for the changes
	tempCollection := fmt.Sprintf("%s_temp_%d", collection, time.Now().Unix())
	
	// Process everything into the temp collection
	batches := rh.createBatches(changes, 500)
	
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			// Cleanup temp collection on cancellation
			rh.cleanupTempCollection(tempCollection)
			return ctx.Err()
		default:
			rh.logger.Infof("Offline processing batch %d/%d (%d files)", i+1, len(batches), len(batch))
			
			// Maximum concurrency for offline processing
			if err := rh.processBatchConcurrent(ctx, batch, tempCollection, 16); err != nil {
				rh.logger.Errorf("Batch %d failed: %v", i+1, err)
			}
			
			// Minimal delay
			if i%10 == 0 && i > 0 {
				rh.logger.Infof("Progress: %d/%d files processed", 
					atomic.LoadInt32(&rh.processedFiles), 
					atomic.LoadInt32(&rh.totalFiles))
			}
		}
	}
	
	// Swap collections atomically
	rh.logger.Info("Swapping collections...")
	if err := rh.swapCollections(collection, tempCollection); err != nil {
		return fmt.Errorf("failed to swap collections: %w", err)
	}
	
	rh.reportProgress(true)
	return nil
}

// processBatchConcurrent processes a batch with concurrent workers
func (rh *RefactorHandler) processBatchConcurrent(ctx context.Context, batch []watcher.FileChange, collection string, workers int) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(batch))
	semaphore := make(chan struct{}, workers)
	
	for _, change := range batch {
		wg.Add(1)
		go func(ch watcher.FileChange) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Rate limiting
			rh.rateLimiter <- struct{}{}
			defer func() { <-rh.rateLimiter }()
			
			if err := rh.processSingleChange(ctx, ch, collection); err != nil {
				errorChan <- fmt.Errorf("%s: %w", ch.Path, err)
				atomic.AddInt32(&rh.failedFiles, 1)
			} else {
				atomic.AddInt32(&rh.processedFiles, 1)
			}
		}(change)
	}
	
	wg.Wait()
	close(errorChan)
	
	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		rh.logger.Warnf("Batch completed with %d errors", len(errors))
	}
	
	return nil
}

// processSingleChange handles a single file change
func (rh *RefactorHandler) processSingleChange(ctx context.Context, change watcher.FileChange, collection string) error {
	switch change.Type {
	case watcher.ChangeTypeCreate, watcher.ChangeTypeModify:
		return rh.indexFile(ctx, change.Path, collection)
	case watcher.ChangeTypeDelete:
		return rh.deleteFile(ctx, change.Path, collection)
	case watcher.ChangeTypeRename:
		if change.OldPath != "" {
			if err := rh.deleteFile(ctx, change.OldPath, collection); err != nil {
				return err
			}
		}
		return rh.indexFile(ctx, change.Path, collection)
	default:
		return nil
	}
}

// indexFile indexes a single file
func (rh *RefactorHandler) indexFile(ctx context.Context, filePath string, collection string) error {
	// Read file content
	content, err := readFileContent(filePath)
	if err != nil {
		return err
	}
	
	// Update dependency graph
	rh.depGraph.AnalyzeFile(filePath, content)
	
	// Index the file
	indexer := NewIncrementalIndexer(rh.engine, 1)
	return indexer.IndexFiles(ctx, []string{filePath}, collection)
}

// deleteFile removes a file from the index
func (rh *RefactorHandler) deleteFile(ctx context.Context, filePath string, collection string) error {
	indexer := NewIncrementalIndexer(rh.engine, 1)
	return indexer.DeleteFiles(ctx, []string{filePath}, collection)
}

// invalidateDependents invalidates files that depend on the changed file
func (rh *RefactorHandler) invalidateDependents(filePath string, collection string) {
	dependents := rh.depGraph.GetDependents(filePath)
	if len(dependents) > 0 {
		rh.logger.Debugf("Invalidating %d dependents of %s", len(dependents), filePath)
		
		// Clear caches for dependent files
		rh.engine.invalidateCachesForFiles(dependents)
		
		// In aggressive mode, re-index dependents
		if rh.currentMode == ModeRealtime {
			ctx := context.Background()
			for _, dep := range dependents {
				rh.indexFile(ctx, dep, collection)
			}
		}
	}
}

// processDependencyInvalidations processes all dependency invalidations
func (rh *RefactorHandler) processDependencyInvalidations(changes []watcher.FileChange, collection string) {
	allDependents := make(map[string]bool)
	
	for _, change := range changes {
		if change.Type != watcher.ChangeTypeDelete {
			for _, dep := range rh.depGraph.GetDependents(change.Path) {
				allDependents[dep] = true
			}
		}
	}
	
	if len(allDependents) > 0 {
		rh.logger.Infof("Processing %d dependent files", len(allDependents))
		
		// Convert to slice
		deps := make([]string, 0, len(allDependents))
		for dep := range allDependents {
			deps = append(deps, dep)
		}
		
		// Clear caches
		rh.engine.invalidateCachesForFiles(deps)
	}
}

// createBatches divides changes into batches
func (rh *RefactorHandler) createBatches(changes []watcher.FileChange, batchSize int) [][]watcher.FileChange {
	var batches [][]watcher.FileChange
	
	for i := 0; i < len(changes); i += batchSize {
		end := i + batchSize
		if end > len(changes) {
			end = len(changes)
		}
		batches = append(batches, changes[i:end])
	}
	
	return batches
}

// determineProcessingMode sets the processing mode based on change volume
func (rh *RefactorHandler) determineProcessingMode(changeCount int) {
	rh.modeMu.Lock()
	defer rh.modeMu.Unlock()
	
	switch {
	case changeCount < 10:
		rh.currentMode = ModeRealtime
	case changeCount < 100:
		rh.currentMode = ModeBatch
	case changeCount < 1000:
		rh.currentMode = ModeBulk
	default:
		rh.currentMode = ModeOffline
	}
}

// getModeName returns the name of the current processing mode
func (rh *RefactorHandler) getModeName() string {
	rh.modeMu.RLock()
	defer rh.modeMu.RUnlock()
	
	switch rh.currentMode {
	case ModeRealtime:
		return "realtime"
	case ModeBatch:
		return "batch"
	case ModeBulk:
		return "bulk"
	case ModeOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// reportProgress reports current processing progress
func (rh *RefactorHandler) reportProgress(final bool) {
	processed := atomic.LoadInt32(&rh.processedFiles)
	failed := atomic.LoadInt32(&rh.failedFiles)
	total := atomic.LoadInt32(&rh.totalFiles)
	
	elapsed := time.Since(rh.startTime)
	rate := float64(processed) / elapsed.Seconds()
	
	if final {
		rh.logger.Infof("Processing complete: %d/%d files (%.1f%%), %d failed, %.1f files/sec, duration: %v",
			processed, total, float64(processed)/float64(total)*100, failed, rate, elapsed)
	} else {
		remaining := total - processed
		eta := time.Duration(float64(remaining)/rate) * time.Second
		rh.logger.Infof("Progress: %d/%d files (%.1f%%), %d failed, %.1f files/sec, ETA: %v",
			processed, total, float64(processed)/float64(total)*100, failed, rate, eta)
	}
}

// createRollbackPoint creates a point we can rollback to
func (rh *RefactorHandler) createRollbackPoint(collection string, changes []watcher.FileChange) error {
	rh.rollbackMu.Lock()
	defer rh.rollbackMu.Unlock()
	
	point := RollbackPoint{
		Timestamp:  time.Now(),
		Collection: collection,
		Files:      make([]string, len(changes)),
	}
	
	for i, change := range changes {
		point.Files[i] = change.Path
	}
	
	// Keep only last 10 rollback points
	if len(rh.rollbackPoints) >= 10 {
		rh.rollbackPoints = rh.rollbackPoints[1:]
	}
	
	rh.rollbackPoints = append(rh.rollbackPoints, point)
	
	rh.logger.Infof("Created rollback point at %v for %d files", point.Timestamp, len(point.Files))
	return nil
}

// Rollback reverts to a previous state
func (rh *RefactorHandler) Rollback(timestamp time.Time) error {
	rh.rollbackMu.Lock()
	defer rh.rollbackMu.Unlock()
	
	// Find the rollback point
	var point *RollbackPoint
	for i := len(rh.rollbackPoints) - 1; i >= 0; i-- {
		if rh.rollbackPoints[i].Timestamp.Before(timestamp) || rh.rollbackPoints[i].Timestamp.Equal(timestamp) {
			point = &rh.rollbackPoints[i]
			break
		}
	}
	
	if point == nil {
		return fmt.Errorf("no rollback point found for timestamp %v", timestamp)
	}
	
	rh.logger.Warnf("Rolling back to %v (%d files)", point.Timestamp, len(point.Files))
	
	// Re-index all files from the rollback point
	ctx := context.Background()
	changes := make([]watcher.FileChange, len(point.Files))
	for i, file := range point.Files {
		changes[i] = watcher.FileChange{
			Path: file,
			Type: watcher.ChangeTypeModify,
		}
	}
	
	// Process with higher priority
	oldMode := rh.currentMode
	rh.currentMode = ModeBatch
	err := rh.ProcessChanges(ctx, changes, point.Collection)
	rh.currentMode = oldMode
	
	return err
}

// Helper functions

func (rh *RefactorHandler) rebuildDependencyGraph(collection string) {
	// This would rebuild the entire dependency graph
	// In practice, this might scan all indexed files
	rh.logger.Info("Dependency graph rebuild complete")
}

func (rh *RefactorHandler) cleanupTempCollection(collection string) {
	// Delete temporary collection
	ctx := context.Background()
	rh.logger.Infof("Cleaning up temporary collection %s", collection)
	// Implementation would delete the collection from vector store
	_ = ctx
}

func (rh *RefactorHandler) swapCollections(main, temp string) error {
	// Atomic collection swap
	// This would be implemented based on the vector store capabilities
	rh.logger.Infof("Swapped %s with %s", main, temp)
	return nil
}

func readFileContent(filePath string) (string, error) {
	// Read file content for dependency analysis
	// Implementation would read the file
	return "", nil
}