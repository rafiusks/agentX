package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/rafael/code-rag-mcp/internal/rag"
	"github.com/sirupsen/logrus"
)

type Server struct {
	mu        sync.RWMutex
	logger    *logrus.Logger
	ragEngine *rag.Engine
	input     io.Reader
	output    io.Writer
	
	capabilities Capabilities
	tools        []Tool
	resources    []Resource
}

func NewServer(ragEngine *rag.Engine) *Server {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &Server{
		logger:    logger,
		ragEngine: ragEngine,
		input:     os.Stdin,
		output:    os.Stdout,
		capabilities: Capabilities{
			Tools: &ToolsCapabilities{
				ListChanged: false,
			},
			Resources: &ResourcesCapabilities{
				Subscribe:   false,
				ListChanged: false,
			},
			Logging: &LoggingCapabilities{},
		},
		tools:     initializeTools(),
		resources: initializeResources(),
	}
}

func initializeTools() []Tool {
	return []Tool{
		{
			Name:        "code_search",
			Description: "Search for code using semantic understanding",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "The search query",
					},
					"language": {
						Type:        "string",
						Description: "Programming language filter",
						Enum:        []string{"go", "javascript", "typescript", "python", "rust", "any"},
						Default:     "any",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of results",
						Default:     10,
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "explain_code",
			Description: "Get detailed explanation of code with context",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"code": {
						Type:        "string",
						Description: "Code snippet to explain",
					},
					"file_path": {
						Type:        "string",
						Description: "Optional file path for additional context",
					},
				},
				Required: []string{"code"},
			},
		},
		{
			Name:        "find_similar",
			Description: "Find code similar to the provided example",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"code": {
						Type:        "string",
						Description: "Example code to find similar patterns",
					},
					"threshold": {
						Type:        "number",
						Description: "Similarity threshold (0.0 to 1.0)",
						Default:     0.7,
					},
				},
				Required: []string{"code"},
			},
		},
		{
			Name:        "index_repository",
			Description: "Index a repository for searching",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository",
					},
					"incremental": {
						Type:        "boolean",
						Description: "Only index changed files",
						Default:     true,
					},
					"force_clean": {
						Type:        "boolean",
						Description: "Force a complete cleanup and rebuild of the index",
						Default:     false,
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_dependencies",
			Description: "Analyze code dependencies",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {
						Type:        "string",
						Description: "Path to the file to analyze",
					},
					"include_transitive": {
						Type:        "boolean",
						Description: "Include transitive dependencies",
						Default:     false,
					},
				},
				Required: []string{"file_path"},
			},
		},
		{
			Name:        "suggest_improvements",
			Description: "Get improvement suggestions for code",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"code": {
						Type:        "string",
						Description: "Code to analyze for improvements",
					},
					"focus": {
						Type:        "string",
						Description: "Focus area for improvements",
						Enum:        []string{"performance", "readability", "security", "best_practices", "all"},
						Default:     "all",
					},
				},
				Required: []string{"code"},
			},
		},
	}
}

func initializeResources() []Resource {
	return []Resource{
		{
			URI:         "indexed_repositories",
			Name:        "Indexed Repositories",
			Description: "List of repositories that have been indexed",
			MimeType:    "application/json",
		},
		{
			URI:         "search_statistics",
			Name:        "Search Statistics",
			Description: "Usage statistics for code search",
			MimeType:    "application/json",
		},
		{
			URI:         "model_capabilities",
			Name:        "Model Capabilities",
			Description: "Available embedding models and their capabilities",
			MimeType:    "application/json",
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.input)
	encoder := json.NewEncoder(s.output)
	
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.WithError(err).Error("Failed to unmarshal request")
			continue
		}
		
		response := s.handleRequest(&req)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				s.logger.WithError(err).Error("Failed to encode response")
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	
	return nil
}

func (s *Server) handleRequest(req *Request) *Response {
	s.logger.WithField("method", req.Method).Debug("Handling request")
	
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodListTools:
		return s.handleListTools(req)
	case MethodCallTool:
		return s.handleCallTool(req)
	case MethodListResources:
		return s.handleListResources(req)
	case MethodReadResource:
		return s.handleReadResource(req)
	case MethodSetLoggingLevel:
		return s.handleSetLoggingLevel(req)
	case MethodPing:
		return s.handlePing(req)
	default:
		return NewErrorResponse(req.ID, NewError(-32601, "Method not found", nil))
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, NewError(-32602, "Invalid params", err.Error()))
	}
	
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    s.capabilities,
		ServerInfo: Implementation{
			Name:    "code-rag-mcp",
			Version: "1.0.0",
		},
		Instructions: "Code RAG MCP Server - Provides intelligent code search and analysis using retrieval-augmented generation.",
	}
	
	resp, err := NewResponse(req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handleListTools(req *Request) *Response {
	result := map[string]interface{}{
		"tools": s.tools,
	}
	
	resp, err := NewResponse(req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handleCallTool(req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, NewError(-32602, "Invalid params", err.Error()))
	}
	
	result, err := s.executeTool(params.Name, params.Arguments)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Tool execution error", err.Error()))
	}
	
	resp, err := NewResponse(req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handleListResources(req *Request) *Response {
	result := map[string]interface{}{
		"resources": s.resources,
	}
	
	resp, err := NewResponse(req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handleReadResource(req *Request) *Response {
	var params ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, NewError(-32602, "Invalid params", err.Error()))
	}
	
	result, err := s.readResource(params.URI)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Resource read error", err.Error()))
	}
	
	resp, err := NewResponse(req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handleSetLoggingLevel(req *Request) *Response {
	var params map[string]string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, NewError(-32602, "Invalid params", err.Error()))
	}
	
	level, ok := params["level"]
	if !ok {
		return NewErrorResponse(req.ID, NewError(-32602, "Missing level parameter", nil))
	}
	
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32602, "Invalid log level", err.Error()))
	}
	
	s.logger.SetLevel(logLevel)
	
	resp, err := NewResponse(req.ID, map[string]interface{}{})
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

func (s *Server) handlePing(req *Request) *Response {
	resp, err := NewResponse(req.ID, map[string]interface{}{})
	if err != nil {
		return NewErrorResponse(req.ID, NewError(-32603, "Internal error", err.Error()))
	}
	
	return resp
}

// HTTPServer provides MCP over HTTP
type HTTPServer struct {
	server    *Server
	address   string
	logger    *logrus.Logger
}

// NewHTTPServer creates a new HTTP MCP server
func NewHTTPServer(ragEngine *rag.Engine, address string) *HTTPServer {
	return &HTTPServer{
		server:  NewServer(ragEngine),
		address: address,
		logger:  logrus.New(),
	}
}

// Run starts the HTTP MCP server
func (hs *HTTPServer) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Handle MCP requests via POST
	mux.HandleFunc("/mcp", hs.handleMCP)
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	server := &http.Server{
		Addr:    hs.address,
		Handler: hs.corsMiddleware(mux),
	}
	
	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logger.Errorf("HTTP server error: %v", err)
		}
	}()
	
	hs.logger.Infof("MCP HTTP server listening on %s", hs.address)
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Graceful shutdown
	return server.Shutdown(context.Background())
}

// handleMCP processes MCP requests over HTTP
func (hs *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Decode JSON-RPC request
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Process request using existing server logic
	resp := hs.server.handleRequest(&req)
	
	// Send JSON-RPC response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		hs.logger.Errorf("Error encoding response: %v", err)
	}
}

// corsMiddleware adds CORS headers
func (hs *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}