package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SimpleMCPClient is a simplified MCP client for testing
type SimpleMCPClient struct {
	serverID string
	userID   uuid.UUID
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	logger   *logrus.Logger
	mu       sync.Mutex
}

// NewSimpleMCPClient creates a new simple MCP client
func NewSimpleMCPClient(serverID string, userID uuid.UUID, cmd *exec.Cmd, logger *logrus.Logger) *SimpleMCPClient {
	return &SimpleMCPClient{
		serverID: serverID,
		userID:   userID,
		cmd:      cmd,
		logger:   logger,
	}
}

// Start starts the MCP server process and establishes connection
func (c *SimpleMCPClient) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set up pipes
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

	// Log stderr to debug
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			c.logger.Debug("[STDERR] " + scanner.Text())
		}
	}()

	c.logger.WithField("pid", c.cmd.Process.Pid).Info("MCP server process started")
	return nil
}

// CallMethod calls a method on the MCP server and returns the raw response
func (c *SimpleMCPClient) CallMethod(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin == nil || c.stdout == nil {
		return nil, fmt.Errorf("client not started")
	}

	// Prepare request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}

	// Send request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.logger.WithField("request", string(requestJSON)).Debug("Sending request")
	
	_, err = c.stdin.Write(requestJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	_, err = c.stdin.Write([]byte("\n"))
	if err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(c.stdout)
	if scanner.Scan() {
		line := scanner.Text()
		c.logger.WithField("response", line).Debug("Received response")
		
		var response map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if errMsg, exists := response["error"]; exists {
			return nil, fmt.Errorf("MCP error: %s", string(errMsg))
		}

		if result, exists := response["result"]; exists {
			return result, nil
		}

		return nil, fmt.Errorf("no result in response")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, fmt.Errorf("no response received")
}

// Initialize performs the MCP initialization handshake
func (c *SimpleMCPClient) Initialize() error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "agentx-simple",
			"version": "1.0.0",
		},
	}

	result, err := c.CallMethod("initialize", params)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.logger.WithField("result", string(result)).Info("MCP server initialized")

	// Send initialized notification
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifJSON, _ := json.Marshal(notification)
	c.stdin.Write(notifJSON)
	c.stdin.Write([]byte("\n"))

	return nil
}

// GetTools retrieves available tools from the server
func (c *SimpleMCPClient) GetTools() ([]MCPToolDefinition, error) {
	result, err := c.CallMethod("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var toolsResult struct {
		Tools []MCPToolDefinition `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResult); err != nil {
		return nil, err
	}

	return toolsResult.Tools, nil
}

// CallTool calls a specific tool
func (c *SimpleMCPClient) CallTool(toolName string, arguments json.RawMessage) (json.RawMessage, error) {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": arguments,
	}

	return c.CallMethod("tools/call", params)
}

// Close closes the client and stops the server
func (c *SimpleMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	return nil
}