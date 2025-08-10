package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func loadOrCreateConfig() *Config {
	configPath := getConfigPath()
	
	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		var config Config
		if json.Unmarshal(data, &config) == nil {
			return &config
		}
	}
	
	// Create new config
	return &Config{
		ProjectsIndexed: []string{},
	}
}

func (c *CLI) saveConfig() error {
	configPath := getConfigPath()
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Save config
	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".code-rag", "config.json")
}

func (c *CLI) checkEnvironment() error {
	// Check for Docker or Go
	hasDocker := commandExists("docker")
	hasGo := commandExists("go")
	
	if !hasDocker && !hasGo {
		return fmt.Errorf("neither Docker nor Go found. Please install one")
	}
	
	return nil
}

func (c *CLI) startServices() error {
	// Check if Qdrant is already running
	if c.checkQdrantRunning() {
		return nil
	}
	
	// Try to start with Docker
	if commandExists("docker") {
		cmd := exec.Command("docker", "run", "-d", 
			"--name", "qdrant",
			"-p", "6333:6333",
			"-p", "6334:6334",
			"qdrant/qdrant:latest")
		
		if err := cmd.Run(); err != nil {
			// Container might already exist
			cmd = exec.Command("docker", "start", "qdrant")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to start Qdrant: %w", err)
			}
		}
		
		// Wait for Qdrant to be ready
		for i := 0; i < 10; i++ {
			if c.checkQdrantRunning() {
				c.config.QdrantRunning = true
				return nil
			}
			time.Sleep(time.Second)
		}
	}
	
	return fmt.Errorf("could not start vector database")
}

func (c *CLI) checkQdrantRunning() bool {
	resp, err := http.Get("http://localhost:6333/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *CLI) autoConfigureClients() error {
	// Try to configure Claude Code
	if err := c.configureClaudeCode(); err == nil {
		return nil
	}
	
	// Try other clients in future
	// c.configureCursor()
	// c.configureVSCode()
	
	return fmt.Errorf("no AI clients found to configure")
}

func (c *CLI) configureClaudeCode() error {
	// Detect Claude Code configuration location
	configPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "claude", "mcp_config.json"),
		filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Claude", "mcp_config.json"),
	}
	
	var configPath string
	for _, path := range configPaths {
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); err == nil {
			configPath = path
			break
		}
	}
	
	if configPath == "" {
		// Try to create in standard location
		if runtime.GOOS == "darwin" {
			configPath = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Claude", "mcp_config.json")
		} else {
			configPath = filepath.Join(os.Getenv("HOME"), ".config", "claude", "mcp_config.json")
		}
	}
	
	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Load existing config or create new
	mcpConfig := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &mcpConfig)
	}
	
	// Ensure mcpServers exists
	if mcpConfig["mcpServers"] == nil {
		mcpConfig["mcpServers"] = make(map[string]interface{})
	}
	servers := mcpConfig["mcpServers"].(map[string]interface{})
	
	// Add our server configuration
	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "code-rag"
	}
	
	servers["code-rag"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"mcp-server"},
	}
	
	// Save configuration
	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}