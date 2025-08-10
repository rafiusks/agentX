package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rafael/code-rag-mcp/internal/rag"
	"github.com/sirupsen/logrus"
)

// FileWatcher manages the file watching service
type FileWatcher struct {
	mu          sync.RWMutex
	engine      *rag.Engine
	projectPath string
	collection  string
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	startTime   time.Time
	
	// Stats
	filesWatched    int
	changesDetected int
	lastChange      time.Time
	
	logger *logrus.Logger
}

// WatcherStatus represents the current status of the file watcher
type WatcherStatus struct {
	Running         bool      `json:"running"`
	ProjectPath     string    `json:"project_path"`
	Collection      string    `json:"collection"`
	FilesWatched    int       `json:"files_watched"`
	ChangesDetected int       `json:"changes_detected"`
	LastChange      time.Time `json:"last_change,omitempty"`
	Uptime          string    `json:"uptime,omitempty"`
}

// NewFileWatcher creates a new file watcher service
func NewFileWatcher(projectPath, collection string) *FileWatcher {
	return &FileWatcher{
		projectPath: projectPath,
		collection:  collection,
		logger:      logrus.New(),
	}
}

// Start starts the file watching service
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if fw.running {
		return fmt.Errorf("file watcher already running")
	}
	
	// Create RAG engine with config
	config, err := fw.loadRAGConfig()
	if err != nil {
		return fmt.Errorf("failed to load RAG config: %w", err)
	}
	
	engine, err := rag.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create RAG engine: %w", err)
	}
	
	fw.engine = engine
	
	// Create context for watcher
	fw.ctx, fw.cancel = context.WithCancel(ctx)
	
	// Start watching in background
	go func() {
		fw.logger.Infof("Starting file watcher for %s", fw.projectPath)
		
		if err := fw.engine.StartWatching(fw.ctx, fw.projectPath, fw.collection); err != nil {
			fw.logger.Errorf("File watcher error: %v", err)
		}
		
		// Mark as stopped when done
		fw.mu.Lock()
		fw.running = false
		fw.mu.Unlock()
	}()
	
	// Count files being watched
	fw.countWatchedFiles()
	
	fw.running = true
	fw.startTime = time.Now()
	
	fw.logger.Info("File watcher service started successfully")
	return nil
}

// Stop stops the file watching service
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if !fw.running {
		return nil
	}
	
	fw.logger.Info("Stopping file watcher service...")
	
	// Cancel context to stop watcher
	if fw.cancel != nil {
		fw.cancel()
	}
	
	// Stop the engine watcher
	if fw.engine != nil {
		if err := fw.engine.StopWatching(); err != nil {
			fw.logger.Warnf("Error stopping watcher: %v", err)
		}
	}
	
	fw.running = false
	fw.logger.Info("File watcher service stopped")
	return nil
}

// IsRunning returns whether the watcher is running
func (fw *FileWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.running
}

// GetStatus returns the current status of the file watcher
func (fw *FileWatcher) GetStatus() WatcherStatus {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	
	status := WatcherStatus{
		Running:         fw.running,
		ProjectPath:     fw.projectPath,
		Collection:      fw.collection,
		FilesWatched:    fw.filesWatched,
		ChangesDetected: fw.changesDetected,
		LastChange:      fw.lastChange,
	}
	
	if fw.running && !fw.startTime.IsZero() {
		status.Uptime = time.Since(fw.startTime).Round(time.Second).String()
	}
	
	return status
}

// RecordChange records that a change was detected
func (fw *FileWatcher) RecordChange() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	fw.changesDetected++
	fw.lastChange = time.Now()
}

// loadRAGConfig loads or creates the RAG configuration
func (fw *FileWatcher) loadRAGConfig() (*rag.Config, error) {
	// Check for existing config file
	configPath := filepath.Join(".code-rag", "rag_config.json")
	
	// Default configuration
	config := &rag.Config{
		EmbeddingConfig: &rag.EmbeddingConfig{
			Provider:  "service",
			Model:     "codebert-base",
			CacheSize: 1000,
		},
		VectorDBConfig: &rag.VectorDBConfig{
			Type:           "qdrant",
			URL:            "http://localhost:6333",
			CollectionName: fw.collection,
		},
		ChunkingConfig: &rag.ChunkingConfig{
			MaxChunkSize:  500,
			ChunkOverlap:  50,
			MinChunkSize:  100,
		},
	}
	
	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			fw.logger.Warnf("Failed to parse config file: %v", err)
		}
	}
	
	return config, nil
}

// countWatchedFiles counts the number of files being watched
func (fw *FileWatcher) countWatchedFiles() {
	count := 0
	
	err := filepath.Walk(fw.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(filepath.Base(path), ".") && path != fw.projectPath {
			return filepath.SkipDir
		}
		
		// Count code files
		if !info.IsDir() && isCodeFile(path) {
			count++
		}
		
		return nil
	})
	
	if err != nil {
		fw.logger.Warnf("Error counting files: %v", err)
	}
	
	fw.filesWatched = count
}

// isCodeFile checks if a file is a code file
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

// HealthCheck performs a health check on the file watcher
func (fw *FileWatcher) HealthCheck() error {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	
	if !fw.running {
		return fmt.Errorf("file watcher not running")
	}
	
	// Check if watcher has been idle too long (might be stuck)
	if !fw.lastChange.IsZero() && time.Since(fw.lastChange) > 24*time.Hour {
		// This is just informational, not an error
		fw.logger.Debug("File watcher idle for over 24 hours")
	}
	
	return nil
}