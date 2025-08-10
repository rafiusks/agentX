package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles the lifecycle of Docker services and file watcher
type Manager struct {
	projectPath string
	verbose     bool
	silent      bool // Silent mode for MCP - no stdout output
	services    []string
	watcher     *FileWatcher
	watcherStop chan struct{}
}

// NewManager creates a new service manager
func NewManager(projectPath string, verbose bool) *Manager {
	return &Manager{
		projectPath: projectPath,
		verbose:     verbose,
		silent:      false,
		services:    []string{"code-rag-qdrant", "code-rag-embedding", "code-rag-cross-encoder"},
		watcherStop: make(chan struct{}),
	}
}

// NewManagerSilent creates a new service manager in silent mode for MCP
func NewManagerSilent(projectPath string) *Manager {
	return &Manager{
		projectPath: projectPath,
		verbose:     false,
		silent:      true,
		services:    []string{"code-rag-qdrant", "code-rag-embedding", "code-rag-cross-encoder"},
		watcherStop: make(chan struct{}),
	}
}

// Start starts all required services
func (m *Manager) Start(ctx context.Context) error {
	if !m.silent {
		fmt.Println("🚀 Starting Code RAG services...")
	}
	
	// Check if Docker is running
	if err := m.checkDocker(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}
	
	// Check if services are already running
	running, err := m.checkRunning()
	if err != nil {
		return err
	}
	
	if running {
		if !m.silent {
			fmt.Println("✅ Services already running")
		}
		return nil
	}
	
	// Start services using docker-compose
	if !m.silent {
		fmt.Println("  Starting Qdrant vector database...")
		fmt.Println("  Starting CodeBERT embedding service...")
	}
	
	cmd := exec.CommandContext(ctx, "docker-compose", "-f", 
		filepath.Join(m.projectPath, "docker-compose.yml"), "up", "-d")
	cmd.Dir = m.projectPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		if m.verbose {
			fmt.Printf("Docker output: %s\n", output)
		}
		return fmt.Errorf("failed to start services: %w", err)
	}
	
	// Wait for services to be ready
	if !m.silent {
		fmt.Print("  Waiting for services to be ready")
	}
	for i := 0; i < 30; i++ {
		if m.checkHealth() {
			if !m.silent {
				fmt.Println("\n✅ All services ready!")
			}
			
			// Start file watcher after Docker services are ready
			if err := m.startFileWatcher(ctx); err != nil {
				if !m.silent {
					fmt.Printf("⚠️  File watcher failed to start: %v\n", err)
				}
				// Don't fail completely if watcher fails
			} else {
				if !m.silent {
					fmt.Println("✅ File watcher started - changes will auto-index")
				}
			}
			
			return nil
		}
		if !m.silent {
			fmt.Print(".")
		}
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("services failed to become healthy")
}

// Stop stops all services
func (m *Manager) Stop() error {
	if !m.silent {
		fmt.Println("🛑 Stopping Code RAG services...")
	}
	
	// Stop file watcher first
	if m.watcher != nil {
		if err := m.watcher.Stop(); err != nil {
			if !m.silent {
				fmt.Printf("Warning: Failed to stop file watcher: %v\n", err)
			}
		}
	}
	
	// Stop Docker services
	cmd := exec.Command("docker-compose", "-f",
		filepath.Join(m.projectPath, "docker-compose.yml"), "down")
	cmd.Dir = m.projectPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try stopping containers directly
		for _, service := range m.services {
			exec.Command("docker", "stop", service).Run()
		}
	}
	
	if m.verbose && len(output) > 0 {
		fmt.Printf("Docker output: %s\n", output)
	}
	
	if !m.silent {
		fmt.Println("✅ Services stopped")
	}
	return nil
}

// Status checks and displays service status
func (m *Manager) Status() {
	fmt.Println("📊 Service Status:")
	fmt.Println("")
	
	// Check Qdrant
	qdrantRunning := m.checkContainer("code-rag-qdrant")
	qdrantHealthy := false
	if qdrantRunning {
		if err := exec.Command("curl", "-s", "-f", "http://localhost:6333/collections").Run(); err == nil {
			qdrantHealthy = true
			fmt.Println("  ✅ Qdrant: Running on http://localhost:6333")
		} else {
			fmt.Println("  🟡 Qdrant: Running but not responding")
		}
	} else {
		fmt.Println("  ❌ Qdrant: Not running")
	}
	
	// Check Embedding Service
	embeddingRunning := m.checkContainer("code-rag-embedding")
	embeddingHealthy := false
	if embeddingRunning {
		if err := exec.Command("curl", "-s", "-f", "http://localhost:8001/health").Run(); err == nil {
			embeddingHealthy = true
			fmt.Println("  ✅ Embedding Service: Running on http://localhost:8001")
		} else {
			fmt.Println("  🟡 Embedding Service: Container running but not responding")
			fmt.Println("     (May be loading model, this can take ~30 seconds)")
		}
	} else {
		fmt.Println("  ❌ Embedding Service: Not running")
		fmt.Println("     Tip: Run 'services start' to start it")
	}
	
	// Check File Watcher
	watcherStatus := m.GetWatcherStatus()
	if watcherStatus.Running {
		fmt.Printf("  ✅ File Watcher: Monitoring %d files\n", watcherStatus.FilesWatched)
		if watcherStatus.ChangesDetected > 0 {
			fmt.Printf("     Changes detected: %d (last: %s ago)\n", 
				watcherStatus.ChangesDetected, 
				time.Since(watcherStatus.LastChange).Round(time.Second))
		}
	} else {
		fmt.Println("  ❌ File Watcher: Not running")
	}
	
	// Overall health
	allHealthy := qdrantHealthy && embeddingHealthy && watcherStatus.Running
	someRunning := qdrantRunning || embeddingRunning || watcherStatus.Running
	
	if allHealthy {
		fmt.Println("\n  🟢 All services healthy")
	} else if someRunning {
		fmt.Println("\n  🟡 Services partially available")
		if !embeddingHealthy && embeddingRunning {
			fmt.Println("     Note: Embedding service may need more memory (2GB+ recommended)")
		}
	} else {
		fmt.Println("\n  🔴 No services running")
	}
}

// checkDocker verifies Docker is available
func (m *Manager) checkDocker() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker not found. Please install Docker Desktop")
	}
	return nil
}

// checkRunning checks if services are already running
func (m *Manager) checkRunning() (bool, error) {
	for _, service := range m.services {
		if m.checkContainer(service) {
			return true, nil
		}
	}
	return false, nil
}

// checkContainer checks if a specific container is running
func (m *Manager) checkContainer(name string) bool {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	containers := strings.Split(string(output), "\n")
	for _, container := range containers {
		if strings.TrimSpace(container) == name {
			return true
		}
	}
	return false
}

// checkHealth verifies services are healthy
func (m *Manager) checkHealth() bool {
	// Check Qdrant health
	qdrantCmd := exec.Command("curl", "-s", "-f", "http://localhost:6333/collections")
	if err := qdrantCmd.Run(); err != nil {
		return false
	}
	
	// Check Embedding service health
	embeddingCmd := exec.Command("curl", "-s", "-f", "http://localhost:8001/health")
	if err := embeddingCmd.Run(); err != nil {
		// Try to restart the embedding service if it's not responding
		if m.checkContainer("code-rag-embedding") {
			// Container exists but not responding, try restart
			exec.Command("docker", "restart", "code-rag-embedding").Run()
			time.Sleep(3 * time.Second)
			// Check again
			if err := exec.Command("curl", "-s", "-f", "http://localhost:8001/health").Run(); err != nil {
				return false
			}
		} else {
			return false
		}
	}
	
	return true
}

// EnsureRunning makes sure services are running
func (m *Manager) EnsureRunning(ctx context.Context) error {
	running, err := m.checkRunning()
	if err != nil {
		return err
	}
	
	if !running {
		return m.Start(ctx)
	}
	
	// Check health even if running
	if !m.checkHealth() {
		if !m.silent {
			fmt.Println("⚠️  Services are running but not healthy, restarting...")
		}
		m.Stop()
		return m.Start(ctx)
	}
	
	return nil
}
// startFileWatcher starts the file watching service
func (m *Manager) startFileWatcher(ctx context.Context) error {
	// Get project configuration
	configPath := filepath.Join(".code-rag", "config.json")
	var projectName string
	
	if data, err := os.ReadFile(configPath); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if name, ok := config["project_name"].(string); ok {
				projectName = name
			}
		}
	}
	
	if projectName == "" {
		projectName = filepath.Base(m.projectPath)
	}
	
	// Get the actual project path to watch
	watchPath := m.projectPath
	if data, err := os.ReadFile(configPath); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if path, ok := config["project_path"].(string); ok {
				watchPath = path
			}
		}
	}
	
	// Create file watcher
	m.watcher = NewFileWatcher(watchPath, projectName)
	
	// Start watcher
	return m.watcher.Start(ctx)
}

// GetWatcherStatus returns the status of the file watcher
func (m *Manager) GetWatcherStatus() WatcherStatus {
	if m.watcher == nil {
		return WatcherStatus{Running: false}
	}
	return m.watcher.GetStatus()
}