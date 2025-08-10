package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// ChangeType represents the type of file change
type ChangeType int

const (
	ChangeTypeCreate ChangeType = iota
	ChangeTypeModify
	ChangeTypeDelete
	ChangeTypeRename
)

// FileChange represents a detected file change
type FileChange struct {
	Path        string
	Type        ChangeType
	OldPath     string // For renames
	ContentHash string
	Timestamp   time.Time
}

// ChangeHandler processes file changes
type ChangeHandler func(changes []FileChange) error

// Watcher monitors file system changes
type Watcher struct {
	mu              sync.RWMutex
	watcher         *fsnotify.Watcher
	rootPath        string
	handler         ChangeHandler
	changeQueue     []FileChange
	queueMu         sync.Mutex
	debounceTime    time.Duration
	batchSize       int
	fileHashes      map[string]string
	excludePatterns []string
	logger          *logrus.Logger
	stopCh          chan struct{}
	doneCh          chan struct{}
	renameTracker   *RenameTracker
}

// Config holds watcher configuration
type Config struct {
	RootPath        string
	Handler         ChangeHandler
	DebounceTime    time.Duration
	BatchSize       int
	ExcludePatterns []string
	Logger          *logrus.Logger
}

// NewWatcher creates a new file watcher
func NewWatcher(config Config) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Set defaults
	if config.DebounceTime == 0 {
		config.DebounceTime = 500 * time.Millisecond
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	w := &Watcher{
		watcher:         fsWatcher,
		rootPath:        config.RootPath,
		handler:         config.Handler,
		debounceTime:    config.DebounceTime,
		batchSize:       config.BatchSize,
		fileHashes:      make(map[string]string),
		excludePatterns: config.ExcludePatterns,
		logger:          config.Logger,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
		renameTracker:   NewRenameTracker(),
	}

	// Add default exclude patterns
	w.excludePatterns = append(w.excludePatterns,
		"*.pyc", "*.pyo", "__pycache__",
		"node_modules", ".git", ".svn",
		"*.swp", "*.swo", "*~",
		".DS_Store", "Thumbs.db",
		"vendor", "target", "dist", "build",
	)

	return w, nil
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Add root directory and subdirectories
	if err := w.addRecursive(w.rootPath); err != nil {
		return fmt.Errorf("failed to add directories: %w", err)
	}

	// Start event processing goroutines
	go w.processEvents(ctx)
	go w.processBatch(ctx)

	w.logger.Infof("Started watching %s", w.rootPath)
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	close(w.stopCh)
	<-w.doneCh
	return w.watcher.Close()
}

// addRecursive adds a directory and all subdirectories to the watcher
func (w *Watcher) addRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip if should be ignored
		if w.shouldIgnore(walkPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only watch directories
		if info.IsDir() {
			if err := w.watcher.Add(walkPath); err != nil {
				w.logger.Warnf("Failed to watch %s: %v", walkPath, err)
			} else {
				w.logger.Debugf("Watching directory: %s", walkPath)
			}
		} else if isCodeFile(walkPath) {
			// Calculate initial hash for code files
			hash, err := w.calculateFileHash(walkPath)
			if err == nil {
				w.mu.Lock()
				w.fileHashes[walkPath] = hash
				w.mu.Unlock()
			}
			// Track file for rename detection
			w.renameTracker.TrackFile(walkPath)
		}

		return nil
	})
}

// processEvents handles file system events
func (w *Watcher) processEvents(ctx context.Context) {
	defer close(w.doneCh)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Errorf("Watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Skip if should be ignored
	if w.shouldIgnore(event.Name) {
		return
	}

	// Only process code files
	if !isCodeFile(event.Name) {
		// But watch new directories
		if event.Op&fsnotify.Create == fsnotify.Create {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				w.addRecursive(event.Name)
			}
		}
		return
	}

	var change *FileChange

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		// Check if this is part of a rename
		if renameEvent := w.renameTracker.HandleCreate(event.Name); renameEvent != nil {
			change = &FileChange{
				Path:      renameEvent.NewPath,
				OldPath:   renameEvent.OldPath,
				Type:      ChangeTypeRename,
				Timestamp: time.Now(),
			}
			w.logger.Debugf("File renamed: %s -> %s", renameEvent.OldPath, renameEvent.NewPath)
		} else {
			change = &FileChange{
				Path:      event.Name,
				Type:      ChangeTypeCreate,
				Timestamp: time.Now(),
			}
			w.logger.Debugf("File created: %s", event.Name)
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		// Check if content actually changed
		newHash, err := w.calculateFileHash(event.Name)
		if err != nil {
			w.logger.Warnf("Failed to calculate hash for %s: %v", event.Name, err)
			return
		}

		w.mu.RLock()
		oldHash, exists := w.fileHashes[event.Name]
		w.mu.RUnlock()

		if exists && oldHash == newHash {
			// Content didn't change, skip
			return
		}

		// Update hash
		w.mu.Lock()
		w.fileHashes[event.Name] = newHash
		w.mu.Unlock()

		change = &FileChange{
			Path:        event.Name,
			Type:        ChangeTypeModify,
			ContentHash: newHash,
			Timestamp:   time.Now(),
		}
		w.logger.Debugf("File modified: %s", event.Name)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// Track the deletion for potential rename detection
		if deleteInfo := w.renameTracker.HandleDelete(event.Name); deleteInfo != nil {
			// Wait a bit to see if this is part of a rename
			time.AfterFunc(100*time.Millisecond, func() {
				// If not matched as rename, process as delete
				w.queueMu.Lock()
				defer w.queueMu.Unlock()
				
				// Check if already processed as rename
				for _, ch := range w.changeQueue {
					if ch.Type == ChangeTypeRename && ch.OldPath == event.Name {
						return
					}
				}
				
				change := FileChange{
					Path:      event.Name,
					Type:      ChangeTypeDelete,
					Timestamp: time.Now(),
				}
				w.changeQueue = append(w.changeQueue, change)
			})
		} else {
			// Regular delete
			change = &FileChange{
				Path:      event.Name,
				Type:      ChangeTypeDelete,
				Timestamp: time.Now(),
			}
		}
		
		// Remove from hash map
		w.mu.Lock()
		delete(w.fileHashes, event.Name)
		w.mu.Unlock()
		
		w.logger.Debugf("File deleted: %s", event.Name)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// FSNotify rename events are unreliable, handle via our tracker
		w.logger.Debugf("File rename event: %s", event.Name)
	}

	if change != nil {
		w.queueChange(*change)
	}
}

// queueChange adds a change to the queue
func (w *Watcher) queueChange(change FileChange) {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()

	// Check if this file already has a pending change
	for i, existing := range w.changeQueue {
		if existing.Path == change.Path {
			// Replace with newer change
			w.changeQueue[i] = change
			return
		}
	}

	w.changeQueue = append(w.changeQueue, change)
}

// processBatch processes queued changes in batches
func (w *Watcher) processBatch(ctx context.Context) {
	ticker := time.NewTicker(w.debounceTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			// Process remaining changes
			w.flushChanges()
			return
		case <-ticker.C:
			w.flushChanges()
		}
	}
}

// flushChanges processes all queued changes
func (w *Watcher) flushChanges() {
	w.queueMu.Lock()
	if len(w.changeQueue) == 0 {
		w.queueMu.Unlock()
		return
	}

	// Take all changes up to batch size
	batchSize := len(w.changeQueue)
	if batchSize > w.batchSize {
		batchSize = w.batchSize
	}

	changes := make([]FileChange, batchSize)
	copy(changes, w.changeQueue[:batchSize])
	w.changeQueue = w.changeQueue[batchSize:]
	w.queueMu.Unlock()

	// Process the batch
	if err := w.handler(changes); err != nil {
		w.logger.Errorf("Failed to handle changes: %v", err)
	} else {
		w.logger.Infof("Processed %d file changes", len(changes))
	}
}

// shouldIgnore checks if a path should be ignored
func (w *Watcher) shouldIgnore(path string) bool {
	// Check exclude patterns
	base := filepath.Base(path)
	for _, pattern := range w.excludePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		// Also check if path contains the pattern (for directories like node_modules)
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Also ignore common version control directories
	if strings.Contains(path, "/.git/") || strings.Contains(path, "/.svn/") {
		return true
	}

	return false
}

// calculateFileHash computes SHA-256 hash of file contents
func (w *Watcher) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// isCodeFile checks if a file is a code file based on extension
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".rb", ".php", ".swift", ".kt", ".rs", ".scala",
		".sh", ".bash", ".zsh", ".yml", ".yaml", ".json", ".xml", ".html",
		".css", ".scss", ".sql", ".proto", ".graphql", ".vue", ".svelte",
	}

	for _, codeExt := range codeExts {
		if ext == codeExt {
			return true
		}
	}

	return false
}

// GetFileHashes returns current file hashes (for persistence)
func (w *Watcher) GetFileHashes() map[string]string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	hashes := make(map[string]string)
	for k, v := range w.fileHashes {
		hashes[k] = v
	}
	return hashes
}

// SetFileHashes loads file hashes (from persistence)
func (w *Watcher) SetFileHashes(hashes map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.fileHashes = hashes
}