package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rafael/code-rag-mcp/internal/rag"
	"github.com/sirupsen/logrus"
)

func runWatch(config *rag.Config, path string, collection string) error {
	// Create engine
	engine, err := rag.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Set up logging
	logrus.SetLevel(logrus.InfoLevel)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutting down file watcher...")
		cancel()
	}()

	// Start watching
	fmt.Printf("👁️  Starting file watcher for %s\n", absPath)
	fmt.Printf("📂 Collection: %s\n", collection)
	fmt.Println("⚡ Changes will be indexed automatically")
	fmt.Println("Press Ctrl+C to stop watching")
	fmt.Println("")

	// Start the watcher
	if err := engine.StartWatching(ctx, absPath, collection); err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop watching
	if err := engine.StopWatching(); err != nil {
		logrus.Warnf("Error stopping watcher: %v", err)
	}

	fmt.Println("✅ File watcher stopped successfully")
	return nil
}

// WatchCommand represents the watch subcommand
type WatchCommand struct {
	Path       string `arg:"positional" help:"Path to watch for changes"`
	Collection string `arg:"-c,--collection" default:"default" help:"Collection to index into"`
	Verbose    bool   `arg:"-v,--verbose" help:"Enable verbose logging"`
}

func (w *WatchCommand) Execute(config *rag.Config) error {
	if w.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return runWatch(config, w.Path, w.Collection)
}