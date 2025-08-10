package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MCPService handles MCP server management and communication
type MCPService struct {
	repo        *postgres.MCPServerRepository
	connections map[uuid.UUID]*MCPConnection
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// MCPConnection represents an active connection to an MCP server
type MCPConnection struct {
	ServerID uuid.UUID
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	scanner  *bufio.Scanner
	encoder  *json.Encoder
	mu       sync.Mutex
	requestID int
	pending  map[int]chan *MCPResponse
}

// MCP Protocol types
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPInitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      MCPClientInfo          `json:"clientInfo"`
}

type MCPClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPInitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      MCPServerInfo          `json:"serverInfo"`
}

type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPToolsResult struct {
	Tools []MCPToolDefinition `json:"tools"`
}

type MCPToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type MCPResourcesResult struct {
	Resources []MCPResourceDefinition `json:"resources"`
}

type MCPResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// NewMCPService creates a new MCP service
func NewMCPService(repo *postgres.MCPServerRepository) *MCPService {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &MCPService{
		repo:        repo,
		connections: make(map[uuid.UUID]*MCPConnection),
		logger:      logger,
	}
}

// CreateServer creates a new MCP server configuration
func (s *MCPService) CreateServer(ctx context.Context, userID uuid.UUID, req *models.MCPServerCreateRequest) (*models.MCPServer, error) {
	server, err := s.repo.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// If enabled, try to connect
	if server.Enabled {
		go s.connectServer(server)
	}

	return server, nil
}

// GetServer retrieves an MCP server
func (s *MCPService) GetServer(ctx context.Context, userID, serverID uuid.UUID) (*models.MCPServer, error) {
	return s.repo.Get(ctx, userID, serverID)
}

// ListServers lists all MCP servers for a user
func (s *MCPService) ListServers(ctx context.Context, userID uuid.UUID) ([]models.MCPServer, error) {
	return s.repo.List(ctx, userID)
}

// UpdateServer updates an MCP server configuration
func (s *MCPService) UpdateServer(ctx context.Context, userID, serverID uuid.UUID, req *models.MCPServerUpdateRequest) (*models.MCPServer, error) {
	server, err := s.repo.Update(ctx, userID, serverID, req)
	if err != nil {
		return nil, err
	}

	// Handle connection state changes
	s.mu.Lock()
	_, exists := s.connections[serverID]
	s.mu.Unlock()

	if req.Enabled != nil {
		if *req.Enabled && !exists {
			// Server was enabled, connect
			go s.connectServer(server)
		} else if !*req.Enabled && exists {
			// Server was disabled, disconnect
			s.disconnectServer(serverID)
		}
	} else if exists && (req.Command != "" || req.Args != nil || req.Env != nil) {
		// Configuration changed, reconnect
		s.disconnectServer(serverID)
		go s.connectServer(server)
	}

	return server, nil
}

// DeleteServer deletes an MCP server
func (s *MCPService) DeleteServer(ctx context.Context, userID, serverID uuid.UUID) error {
	// Disconnect if connected
	s.disconnectServer(serverID)
	
	return s.repo.Delete(ctx, userID, serverID)
}

// ToggleServer enables or disables an MCP server
func (s *MCPService) ToggleServer(ctx context.Context, userID, serverID uuid.UUID) (*models.MCPServer, error) {
	server, err := s.repo.Get(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	enabled := !server.Enabled
	req := &models.MCPServerUpdateRequest{
		Enabled: &enabled,
	}

	return s.UpdateServer(ctx, userID, serverID, req)
}

// ConnectServer establishes a connection to an MCP server
func (s *MCPService) connectServer(server *models.MCPServer) {
	s.logger.WithField("server", server.Name).Info("Connecting to MCP server")

	// Update status to connecting
	ctx := context.Background()
	s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusConnecting)

	// Create command
	cmd := exec.Command(server.Command, server.Args...)
	
	// Set environment variables
	if server.Env != nil {
		var env map[string]string
		if err := json.Unmarshal(server.Env, &env); err == nil {
			for k, v := range env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	// Create pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.logger.WithError(err).Error("Failed to create stdin pipe")
		s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusError)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.logger.WithError(err).Error("Failed to create stdout pipe")
		s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusError)
		return
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		s.logger.WithError(err).Error("Failed to start MCP server")
		s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusError)
		return
	}

	// Create connection
	conn := &MCPConnection{
		ServerID:  server.ID,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		scanner:   bufio.NewScanner(stdout),
		encoder:   json.NewEncoder(stdin),
		requestID: 0,
		pending:   make(map[int]chan *MCPResponse),
	}

	// Store connection
	s.mu.Lock()
	s.connections[server.ID] = conn
	s.mu.Unlock()

	// Start response handler
	go s.handleResponses(conn)

	// Initialize the connection
	if err := s.initializeConnection(ctx, server.ID); err != nil {
		s.logger.WithError(err).Error("Failed to initialize MCP connection")
		s.disconnectServer(server.ID)
		s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusError)
		return
	}

	// Discover tools and resources
	if err := s.discoverCapabilities(ctx, server.ID); err != nil {
		s.logger.WithError(err).Error("Failed to discover MCP capabilities")
	}

	// Update status to connected
	s.repo.UpdateStatus(ctx, server.ID, models.MCPServerStatusConnected)
	s.logger.WithField("server", server.Name).Info("Successfully connected to MCP server")
}

// disconnectServer disconnects from an MCP server
func (s *MCPService) disconnectServer(serverID uuid.UUID) {
	s.mu.Lock()
	conn, exists := s.connections[serverID]
	if exists {
		delete(s.connections, serverID)
	}
	s.mu.Unlock()

	if !exists {
		return
	}

	// Close pipes
	conn.stdin.Close()
	conn.stdout.Close()

	// Terminate process
	if conn.cmd != nil && conn.cmd.Process != nil {
		conn.cmd.Process.Kill()
		conn.cmd.Wait()
	}

	// Update status
	ctx := context.Background()
	s.repo.UpdateStatus(ctx, serverID, models.MCPServerStatusDisconnected)
}

// initializeConnection performs the MCP initialization handshake
func (s *MCPService) initializeConnection(ctx context.Context, serverID uuid.UUID) error {
	params := MCPInitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: MCPClientInfo{
			Name:    "agentx",
			Version: "1.0.0",
		},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	resp, err := s.sendRequest(serverID, "initialize", paramsJSON)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("initialization error: %s", resp.Error.Message)
	}

	var result MCPInitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return err
	}

	// Store capabilities
	return s.repo.UpdateCapabilities(ctx, serverID, result.Capabilities)
}

// discoverCapabilities discovers tools and resources from the MCP server
func (s *MCPService) discoverCapabilities(ctx context.Context, serverID uuid.UUID) error {
	// Discover tools
	if err := s.discoverTools(ctx, serverID); err != nil {
		return err
	}

	// Discover resources
	if err := s.discoverResources(ctx, serverID); err != nil {
		return err
	}

	return nil
}

// discoverTools discovers available tools from the MCP server
func (s *MCPService) discoverTools(ctx context.Context, serverID uuid.UUID) error {
	resp, err := s.sendRequest(serverID, "tools/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tools discovery error: %s", resp.Error.Message)
	}

	var result MCPToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return err
	}

	// Store tools in database
	for _, toolDef := range result.Tools {
		tool := &models.MCPTool{
			Name:        toolDef.Name,
			Description: toolDef.Description,
			InputSchema: toolDef.InputSchema,
			Enabled:     true,
		}
		
		if err := s.repo.UpsertTool(ctx, serverID, tool); err != nil {
			s.logger.WithError(err).WithField("tool", toolDef.Name).Error("Failed to store tool")
		}
	}

	return nil
}

// discoverResources discovers available resources from the MCP server
func (s *MCPService) discoverResources(ctx context.Context, serverID uuid.UUID) error {
	resp, err := s.sendRequest(serverID, "resources/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("resources discovery error: %s", resp.Error.Message)
	}

	var result MCPResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return err
	}

	// Store resources in database
	for _, resDef := range result.Resources {
		resource := &models.MCPResource{
			URI:         resDef.URI,
			Name:        resDef.Name,
			Description: resDef.Description,
			MimeType:    resDef.MimeType,
		}
		
		if err := s.repo.UpsertResource(ctx, serverID, resource); err != nil {
			s.logger.WithError(err).WithField("resource", resDef.URI).Error("Failed to store resource")
		}
	}

	return nil
}

// sendRequest sends a request to an MCP server
func (s *MCPService) sendRequest(serverID uuid.UUID, method string, params json.RawMessage) (*MCPResponse, error) {
	s.mu.RLock()
	conn, exists := s.connections[serverID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server not connected")
	}

	conn.mu.Lock()
	conn.requestID++
	requestID := conn.requestID
	
	// Create response channel
	respChan := make(chan *MCPResponse, 1)
	conn.pending[requestID] = respChan
	conn.mu.Unlock()

	// Send request
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  params,
	}

	if err := conn.encoder.Encode(req); err != nil {
		conn.mu.Lock()
		delete(conn.pending, requestID)
		conn.mu.Unlock()
		return nil, err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		conn.mu.Lock()
		delete(conn.pending, requestID)
		conn.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// handleResponses handles responses from an MCP server
func (s *MCPService) handleResponses(conn *MCPConnection) {
	for conn.scanner.Scan() {
		line := conn.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp MCPResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			s.logger.WithError(err).Error("Failed to unmarshal MCP response")
			continue
		}

		// Find pending request
		conn.mu.Lock()
		if ch, exists := conn.pending[resp.ID]; exists {
			delete(conn.pending, resp.ID)
			conn.mu.Unlock()
			ch <- &resp
		} else {
			conn.mu.Unlock()
		}
	}

	// Scanner stopped, connection closed
	s.disconnectServer(conn.ServerID)
}

// CallTool calls a tool on an MCP server
func (s *MCPService) CallTool(ctx context.Context, req *models.MCPToolCallRequest) (*models.MCPToolCallResponse, error) {
	// Prepare parameters
	params := map[string]interface{}{
		"name":      req.ToolName,
		"arguments": req.Arguments,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	// Send request
	resp, err := s.sendRequest(req.ServerID, "tools/call", paramsJSON)
	if err != nil {
		return nil, err
	}

	// Handle response
	response := &models.MCPToolCallResponse{}
	if resp.Error != nil {
		errorMsg := resp.Error.Message
		response.Error = &errorMsg
	} else {
		var result interface{}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, err
		}
		response.Result = result
	}

	// Update tool usage
	// TODO: Get tool ID from database and update usage

	return response, nil
}

// ReadResource reads a resource from an MCP server
func (s *MCPService) ReadResource(ctx context.Context, req *models.MCPResourceReadRequest) (*models.MCPResourceReadResponse, error) {
	// Prepare parameters
	params := map[string]interface{}{
		"uri": req.URI,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	// Send request
	resp, err := s.sendRequest(req.ServerID, "resources/read", paramsJSON)
	if err != nil {
		return nil, err
	}

	// Handle response
	response := &models.MCPResourceReadResponse{}
	if resp.Error != nil {
		errorMsg := resp.Error.Message
		response.Error = &errorMsg
	} else {
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, err
		}
		
		if content, ok := result["contents"]; ok {
			response.Content = content
		}
		if mimeType, ok := result["mimeType"].(string); ok {
			response.MimeType = mimeType
		}
	}

	return response, nil
}

// ConnectEnabledServers connects to all enabled servers for a user
func (s *MCPService) ConnectEnabledServers(ctx context.Context, userID uuid.UUID) {
	servers, err := s.repo.List(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list servers for connection")
		return
	}

	for _, server := range servers {
		if server.Enabled {
			go s.connectServer(&server)
		}
	}
}