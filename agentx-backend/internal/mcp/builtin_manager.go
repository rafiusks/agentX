package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/agentx/agentx-backend/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BuiltinMCPServer represents a built-in MCP server configuration
type BuiltinMCPServer struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Category    string                 `json:"category"`
	Tools       []BuiltinMCPTool       `json:"tools"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	Required    []string               `json:"required"` // Required dependencies
}

type BuiltinMCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
}

// BuiltinMCPManager manages built-in MCP servers
type BuiltinMCPManager struct {
	servers       map[string]*BuiltinMCPServer
	userSettings  map[uuid.UUID]map[string]bool // userID -> serverID -> enabled
	processes     map[string]*exec.Cmd          // serverID -> process
	clients       map[string]*SimpleMCPClient   // processKey -> client (using simple client for now)
	mu            sync.RWMutex
	logger        *logrus.Logger
	backendPath   string
}

// NewBuiltinMCPManager creates a new built-in MCP manager
func NewBuiltinMCPManager(backendPath string) *BuiltinMCPManager {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	manager := &BuiltinMCPManager{
		servers:      make(map[string]*BuiltinMCPServer),
		userSettings: make(map[uuid.UUID]map[string]bool),
		processes:    make(map[string]*exec.Cmd),
		clients:      make(map[string]*SimpleMCPClient),
		logger:       logger,
		backendPath:  backendPath,
	}

	manager.initializeBuiltinServers()
	return manager
}

// initializeBuiltinServers registers all built-in MCP servers
func (m *BuiltinMCPManager) initializeBuiltinServers() {
	// Web Search MCP Server
	webSearch := &BuiltinMCPServer{
		ID:          "builtin-websearch",
		Name:        "Web Search",
		Description: "Robust web search with multiple providers (Bing, Brave, DuckDuckGo) and content extraction",
		Version:     "1.0.0",
		Category:    "Search & Research",
		Tools: []BuiltinMCPTool{
			{
				Name:        "web_search",
				Description: "Search the web using multiple search engines with fallback support",
				Example: map[string]interface{}{
					"query":          "latest AI developments 2024",
					"maxResults":     5,
					"searchEngine":   "auto",
					"includeContent": false,
					"timeRange":      "month",
				},
			},
			{
				Name:        "fetch_page",
				Description: "Extract content from any web page URL",
				Example: map[string]interface{}{
					"url":          "https://example.com/article",
					"format":       "markdown",
					"maxLength":    10000,
					"includeLinks": true,
				},
			},
			{
				Name:        "search_and_summarize",
				Description: "Research topics with automatic summarization",
				Example: map[string]interface{}{
					"query":         "climate change renewable energy solutions",
					"maxPages":      5,
					"summaryLength": "detailed",
					"includeLinks":  true,
				},
			},
		},
		Config: map[string]interface{}{
			"LOG_LEVEL":           "info",
			"BROWSER_HEADLESS":    "true",
			"MAX_SEARCH_RESULTS":  "10",
			"MAX_CONTENT_LENGTH":  "50000",
			"LOG_SEARCH_QUERIES":  "false",
			"BING_ENABLED":        "true",
			"BRAVE_ENABLED":       "true",
			"DUCKDUCKGO_ENABLED":  "true",
		},
		Enabled:  false, // Disabled by default, user can enable
		Required: []string{"node", "npm"},
	}

	m.servers[webSearch.ID] = webSearch
	m.logger.WithField("server", webSearch.Name).Info("Registered built-in MCP server")
}

// GetBuiltinServers returns all available built-in MCP servers
func (m *BuiltinMCPManager) GetBuiltinServers() []BuiltinMCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]BuiltinMCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, *server)
	}
	return servers
}

// GetBuiltinServer returns a specific built-in MCP server
func (m *BuiltinMCPManager) GetBuiltinServer(serverID string) (*BuiltinMCPServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("built-in server %s not found", serverID)
	}
	return server, nil
}

// SetUserServerEnabled enables or disables a built-in server for a specific user
func (m *BuiltinMCPManager) SetUserServerEnabled(userID uuid.UUID, serverID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server exists
	server, exists := m.servers[serverID]
	if !exists {
		return fmt.Errorf("built-in server %s not found", serverID)
	}

	// Initialize user settings if needed
	if m.userSettings[userID] == nil {
		m.userSettings[userID] = make(map[string]bool)
	}

	// Update setting
	m.userSettings[userID][serverID] = enabled

	// Start or stop the server process if needed
	if enabled {
		return m.startServerForUser(userID, server)
	} else {
		return m.stopServerForUser(userID, serverID)
	}
}

// IsServerEnabledForUser checks if a server is enabled for a specific user
func (m *BuiltinMCPManager) IsServerEnabledForUser(userID uuid.UUID, serverID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if userSettings, exists := m.userSettings[userID]; exists {
		return userSettings[serverID]
	}
	return false
}

// GetUserEnabledServers returns all servers enabled for a user
func (m *BuiltinMCPManager) GetUserEnabledServers(userID uuid.UUID) []BuiltinMCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var enabledServers []BuiltinMCPServer
	
	if userSettings, exists := m.userSettings[userID]; exists {
		for serverID, enabled := range userSettings {
			if enabled {
				if server, exists := m.servers[serverID]; exists {
					enabledServers = append(enabledServers, *server)
				}
			}
		}
	}
	
	return enabledServers
}

// startServerForUser starts a built-in MCP server process for a user
func (m *BuiltinMCPManager) startServerForUser(userID uuid.UUID, server *BuiltinMCPServer) error {
	processKey := fmt.Sprintf("%s-%s", server.ID, userID.String())
	
	// Check if client already exists
	if _, exists := m.clients[processKey]; exists {
		return nil // Already running
	}

	// Check requirements
	if err := m.checkRequirements(server); err != nil {
		return fmt.Errorf("requirements not met for %s: %w", server.Name, err)
	}

	// Create the server command based on type
	var cmd *exec.Cmd
	switch server.ID {
	case "builtin-websearch":
		cmd = m.createWebSearchCommand(userID, server)
	default:
		return fmt.Errorf("unknown built-in server type: %s", server.ID)
	}

	if cmd == nil {
		return fmt.Errorf("failed to create process for %s", server.Name)
	}

	// Create simple MCP client
	client := NewSimpleMCPClient(server.ID, userID, cmd, m.logger)
	
	// Start the server
	if err := client.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", server.Name, err)
	}

	// Initialize the connection
	if err := client.Initialize(); err != nil {
		client.Close()
		return fmt.Errorf("failed to initialize %s: %w", server.Name, err)
	}

	// Get tools to verify connection
	tools, err := client.GetTools()
	if err != nil {
		m.logger.WithError(err).Warn("Failed to get tools")
	}

	// Store the client and process
	m.clients[processKey] = client
	m.processes[processKey] = cmd
	
	m.logger.WithFields(logrus.Fields{
		"server": server.Name,
		"user":   userID,
		"pid":    cmd.Process.Pid,
		"tools":  len(tools),
	}).Info("Connected to built-in MCP server")

	return nil
}

// createWebSearchCommand creates the command for the web search MCP server
func (m *BuiltinMCPManager) createWebSearchCommand(userID uuid.UUID, server *BuiltinMCPServer) *exec.Cmd {
	// Build the web search server if needed
	webSearchPath := filepath.Join(m.backendPath, "internal", "mcp", "builtin", "websearch")
	
	// Check if TypeScript build exists, if not build it
	distPath := filepath.Join(webSearchPath, "dist", "index.js")
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		m.logger.Info("Building web search MCP server...")
		if err := m.buildWebSearchServer(webSearchPath); err != nil {
			m.logger.WithError(err).Error("Failed to build web search server")
			return nil
		}
	}

	// Create command
	cmd := exec.Command("node", distPath)
	
	// Set environment variables
	env := os.Environ()
	for key, value := range server.Config {
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}
	cmd.Env = env

	// Set working directory
	cmd.Dir = webSearchPath

	return cmd
}

// buildWebSearchServer builds the TypeScript web search server
func (m *BuiltinMCPManager) buildWebSearchServer(webSearchPath string) error {
	// Check if package.json exists
	packageJSONPath := filepath.Join(webSearchPath, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		// Copy package.json from the standalone version
		return m.setupWebSearchServer(webSearchPath)
	}

	// Install dependencies
	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = webSearchPath
	if err := npmInstall.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Build TypeScript
	npmBuild := exec.Command("npm", "run", "build")
	npmBuild.Dir = webSearchPath
	if err := npmBuild.Run(); err != nil {
		return fmt.Errorf("npm build failed: %w", err)
	}

	return nil
}

// setupWebSearchServer sets up the web search server files
func (m *BuiltinMCPManager) setupWebSearchServer(webSearchPath string) error {
	// Copy package.json and tsconfig.json from standalone version
	sourcePackageJSON := `{
  "name": "agentx-builtin-websearch",
  "version": "1.0.0",
  "description": "Built-in web search MCP server for AgentX",
  "main": "dist/index.js",
  "type": "module",
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.0.0",
    "axios": "^1.6.0",
    "playwright": "^1.40.0",
    "turndown": "^7.1.2",
    "cheerio": "^1.0.0-rc.12",
    "zod": "^3.22.0"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "@types/turndown": "^5.0.4",
    "typescript": "^5.3.0"
  }
}`

	sourceTSConfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "node",
    "outDir": "./dist",
    "rootDir": "./",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "sourceMap": true,
    "resolveJsonModule": true,
    "allowSyntheticDefaultImports": true,
    "lib": ["ES2022", "DOM"]
  },
  "include": ["**/*"],
  "exclude": ["node_modules", "dist", "**/*.test.ts"]
}`

	// Write files
	if err := os.WriteFile(filepath.Join(webSearchPath, "package.json"), []byte(sourcePackageJSON), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(webSearchPath, "tsconfig.json"), []byte(sourceTSConfig), 0644); err != nil {
		return err
	}

	return nil
}

// stopServerForUser stops a built-in MCP server process for a user
func (m *BuiltinMCPManager) stopServerForUser(userID uuid.UUID, serverID string) error {
	processKey := fmt.Sprintf("%s-%s", serverID, userID.String())
	
	// Close the client if it exists
	if client, exists := m.clients[processKey]; exists {
		if err := client.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close MCP client gracefully")
		}
		delete(m.clients, processKey)
	}
	
	// Clean up process reference
	if _, exists := m.processes[processKey]; exists {
		delete(m.processes, processKey)
	}
	
	m.logger.WithFields(logrus.Fields{
		"server": serverID,
		"user":   userID,
	}).Info("Stopped built-in MCP server")
	
	return nil
}

// checkRequirements checks if the system meets the requirements for a server
func (m *BuiltinMCPManager) checkRequirements(server *BuiltinMCPServer) error {
	for _, req := range server.Required {
		if _, err := exec.LookPath(req); err != nil {
			return fmt.Errorf("required dependency '%s' not found", req)
		}
	}
	return nil
}

// isProcessRunning checks if a process is still running
func isProcessRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	
	// Try to send signal 0 (doesn't actually send a signal, just checks if process exists)
	return cmd.Process.Signal(os.Signal(nil)) == nil
}

// Cleanup stops all running server processes
func (m *BuiltinMCPManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all clients
	for processKey, client := range m.clients {
		if err := client.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close MCP client during cleanup")
		}
		delete(m.clients, processKey)
		delete(m.processes, processKey)
	}
	
	m.logger.Info("Cleaned up all built-in MCP server processes")
}

// CallTool calls a tool on a built-in MCP server
func (m *BuiltinMCPManager) CallTool(userID uuid.UUID, serverID string, toolName string, arguments json.RawMessage) (interface{}, error) {
	processKey := fmt.Sprintf("%s-%s", serverID, userID.String())
	
	m.mu.RLock()
	client, exists := m.clients[processKey]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("server %s not running for user", serverID)
	}
	
	// Call the tool
	result, err := client.CallTool(toolName, arguments)
	if err != nil {
		return nil, err
	}
	
	// Parse the result
	var toolResult interface{}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		// Return raw result if unmarshal fails
		return string(result), nil
	}
	
	return toolResult, nil
}

// GetServerTools returns the available tools for a built-in server
func (m *BuiltinMCPManager) GetServerTools(userID uuid.UUID, serverID string) ([]MCPToolDefinition, error) {
	processKey := fmt.Sprintf("%s-%s", serverID, userID.String())
	
	m.mu.RLock()
	client, exists := m.clients[processKey]
	m.mu.RUnlock()
	
	if !exists {
		// Server not running, return static tools from server definition
		server, err := m.GetBuiltinServer(serverID)
		if err != nil {
			return nil, err
		}
		
		// Convert static tools to MCPToolDefinition
		tools := make([]MCPToolDefinition, len(server.Tools))
		for i, tool := range server.Tools {
			tools[i] = MCPToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
			}
		}
		return tools, nil
	}
	
	// Get tools from running server
	tools, err := client.GetTools()
	if err != nil {
		return nil, err
	}
	
	return tools, nil
}

// ConvertToRegularMCPServer converts a built-in server to a regular MCP server for a user
func (m *BuiltinMCPManager) ConvertToRegularMCPServer(userID uuid.UUID, serverID string) (*models.MCPServerCreateRequest, error) {
	server, err := m.GetBuiltinServer(serverID)
	if err != nil {
		return nil, err
	}

	// Create environment variables
	envVars := make(map[string]string)
	for key, value := range server.Config {
		envVars[key] = fmt.Sprintf("%v", value)
	}

	// Determine command based on server type
	var command string
	var args []string
	
	switch serverID {
	case "builtin-websearch":
		webSearchPath := filepath.Join(m.backendPath, "internal", "mcp", "builtin", "websearch", "dist", "index.js")
		command = "node"
		args = []string{webSearchPath}
	default:
		return nil, fmt.Errorf("unknown server type for conversion: %s", serverID)
	}

	return &models.MCPServerCreateRequest{
		Name:        server.Name,
		Description: server.Description,
		Command:     command,
		Args:        args,
		Env:         envVars,
		Enabled:     true,
	}, nil
}