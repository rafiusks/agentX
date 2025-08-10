package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BuiltinMCPClient handles communication with a built-in MCP server
type BuiltinMCPClient struct {
	serverID  string
	userID    uuid.UUID
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	encoder   *json.Encoder
	mu        sync.Mutex
	requestID int
	pending   map[int]chan *MCPResponse
	logger    *logrus.Logger
	connected bool
	tools     []MCPToolDefinition
	resources []MCPResourceDefinition
}

// MCP Protocol types (reusing from services package for consistency)
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

type MCPToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type MCPToolCallResult struct {
	Content []MCPContent `json:"content"`
}

type MCPContent struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// NewBuiltinMCPClient creates a new client for a built-in MCP server
func NewBuiltinMCPClient(serverID string, userID uuid.UUID, cmd *exec.Cmd, logger *logrus.Logger) *BuiltinMCPClient {
	return &BuiltinMCPClient{
		serverID:  serverID,
		userID:    userID,
		cmd:       cmd,
		requestID: 0,
		pending:   make(map[int]chan *MCPResponse),
		logger:    logger,
		connected: false,
		tools:     []MCPToolDefinition{},
		resources: []MCPResourceDefinition{},
	}
}

// Connect establishes the connection and performs MCP initialization
func (c *BuiltinMCPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Create pipes
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = stdout

	// Set up stderr to log (MCP servers often log to stderr)
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Set up JSON encoder and scanner
	c.encoder = json.NewEncoder(c.stdin)
	c.scanner = bufio.NewScanner(c.stdout)
	// Increase scanner buffer size for large responses
	c.scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start stderr logger
	go c.logStderr(stderr)

	// Start response handler
	go c.handleResponses()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Initialize MCP connection
	if err := c.initialize(); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	// Discover capabilities
	if err := c.discoverCapabilities(); err != nil {
		c.logger.WithError(err).Warn("Failed to discover capabilities")
	}

	c.connected = true
	c.logger.WithFields(logrus.Fields{
		"server": c.serverID,
		"user":   c.userID,
		"pid":    c.cmd.Process.Pid,
	}).Info("Connected to built-in MCP server")

	return nil
}

// initialize performs the MCP initialization handshake
func (c *BuiltinMCPClient) initialize() error {
	c.logger.Debug("Sending MCP initialize request")
	
	params := MCPInitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: MCPClientInfo{
			Name:    "agentx-builtin",
			Version: "1.0.0",
		},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	c.logger.WithField("params", string(paramsJSON)).Debug("Initialize params")

	resp, err := c.sendRequest("initialize", paramsJSON)
	if err != nil {
		c.logger.WithError(err).Error("Failed to send initialize request")
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("initialization error: %s", resp.Error.Message)
	}

	var result MCPInitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"server":   result.ServerInfo.Name,
		"version":  result.ServerInfo.Version,
		"protocol": result.ProtocolVersion,
	}).Info("MCP server initialized")

	// Send initialized notification
	notif := MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	c.logger.Debug("Sending initialized notification")
	if err := c.encoder.Encode(notif); err != nil {
		return err
	}
	
	// Flush if possible
	if flusher, ok := c.stdin.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
	
	return nil
}

// discoverCapabilities discovers tools and resources from the server
func (c *BuiltinMCPClient) discoverCapabilities() error {
	// Discover tools
	if err := c.discoverTools(); err != nil {
		return err
	}

	// Discover resources
	if err := c.discoverResources(); err != nil {
		return err
	}

	return nil
}

// discoverTools discovers available tools
func (c *BuiltinMCPClient) discoverTools() error {
	resp, err := c.sendRequest("tools/list", nil)
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

	c.tools = result.Tools
	c.logger.WithField("count", len(c.tools)).Debug("Discovered MCP tools")

	return nil
}

// discoverResources discovers available resources
func (c *BuiltinMCPClient) discoverResources() error {
	resp, err := c.sendRequest("resources/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		// Resources might not be implemented
		c.logger.Debug("Server does not support resources")
		return nil
	}

	var result MCPResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return err
	}

	c.resources = result.Resources
	c.logger.WithField("count", len(c.resources)).Debug("Discovered MCP resources")

	return nil
}

// sendRequest sends a request to the MCP server
func (c *BuiltinMCPClient) sendRequest(method string, params json.RawMessage) (*MCPResponse, error) {
	c.mu.Lock()
	c.requestID++
	requestID := c.requestID

	// Create response channel
	respChan := make(chan *MCPResponse, 1)
	c.pending[requestID] = respChan
	c.mu.Unlock()

	// Send request
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  params,
	}

	reqJSON, _ := json.Marshal(req)
	c.logger.WithFields(logrus.Fields{
		"method": method,
		"id":     requestID,
	}).Debug("Sending request: " + string(reqJSON))

	if err := c.encoder.Encode(req); err != nil {
		c.mu.Lock()
		delete(c.pending, requestID)
		c.mu.Unlock()
		return nil, err
	}
	
	// Ensure the request is actually sent
	if flusher, ok := c.stdin.(interface{ Flush() error }); ok {
		flusher.Flush()
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		respJSON, _ := json.Marshal(resp)
		c.logger.Debug("Received response: " + string(respJSON))
		return resp, nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.pending, requestID)
		c.mu.Unlock()
		c.logger.WithField("method", method).Error("Request timeout")
		return nil, fmt.Errorf("request timeout for method %s", method)
	}
}

// logStderr logs stderr output from the MCP server
func (c *BuiltinMCPClient) logStderr(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			c.logger.WithFields(logrus.Fields{
				"server": c.serverID,
				"user":   c.userID,
			}).Debug(line)
		}
	}
}

// handleResponses handles responses from the MCP server
func (c *BuiltinMCPClient) handleResponses() {
	c.logger.Debug("Starting response handler")
	
	if c.scanner == nil {
		c.logger.Error("Scanner is nil!")
		return
	}
	
	for {
		if !c.scanner.Scan() {
			c.logger.Debug("Scanner stopped scanning")
			break
		}
		
		line := c.scanner.Bytes()
		c.logger.WithField("raw_len", len(line)).Debug("Scanned line")
		
		if len(line) == 0 {
			continue
		}

		c.logger.WithField("raw", string(line)).Debug("Received line from server")

		var resp MCPResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			c.logger.WithError(err).WithField("line", string(line)).Debug("Failed to unmarshal MCP response")
			continue
		}

		c.logger.WithFields(logrus.Fields{
			"id":     resp.ID,
			"method": resp.JSONRPC,
		}).Debug("Parsed MCP response")

		// Find pending request
		c.mu.Lock()
		ch, exists := c.pending[resp.ID]
		if exists {
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()
		
		if exists {
			c.logger.WithField("id", resp.ID).Debug("Sending response to waiting channel")
			select {
			case ch <- &resp:
				c.logger.WithField("id", resp.ID).Debug("Response sent successfully")
			default:
				c.logger.WithField("id", resp.ID).Error("Channel blocked, couldn't send response")
			}
		} else {
			c.logger.WithField("id", resp.ID).Debug("No pending request for response")
		}
	}

	if err := c.scanner.Err(); err != nil {
		c.logger.WithError(err).Error("Scanner error")
	}

	// Scanner stopped, connection closed
	c.connected = false
	c.logger.Debug("Response handler stopped")
}

// CallTool calls a tool on the MCP server
func (c *BuiltinMCPClient) CallTool(ctx context.Context, toolName string, arguments json.RawMessage) (*MCPToolCallResult, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	params := MCPToolCallParams{
		Name:      toolName,
		Arguments: arguments,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.sendRequest("tools/call", paramsJSON)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tool call error: %s", resp.Error.Message)
	}

	var result MCPToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTools returns the discovered tools
func (c *BuiltinMCPClient) GetTools() []MCPToolDefinition {
	return c.tools
}

// GetResources returns the discovered resources
func (c *BuiltinMCPClient) GetResources() []MCPResourceDefinition {
	return c.resources
}

// IsConnected returns whether the client is connected
func (c *BuiltinMCPClient) IsConnected() bool {
	return c.connected
}

// Close closes the connection to the MCP server
func (c *BuiltinMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false

	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}

	// Terminate process
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	c.logger.WithFields(logrus.Fields{
		"server": c.serverID,
		"user":   c.userID,
	}).Info("Disconnected from built-in MCP server")

	return nil
}