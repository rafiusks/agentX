package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/agentx/agentx-backend/internal/mcp"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MCPToolIntegration handles integration of MCP tools with chat
type MCPToolIntegration struct {
	builtinManager *mcp.BuiltinMCPManager
	mcpService     *MCPService
	logger         *logrus.Logger
}

// NewMCPToolIntegration creates a new MCP tool integration service
func NewMCPToolIntegration(builtinManager *mcp.BuiltinMCPManager, mcpService *MCPService) *MCPToolIntegration {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &MCPToolIntegration{
		builtinManager: builtinManager,
		mcpService:     mcpService,
		logger:         logger,
	}
}

// ToolInvocation represents a detected tool invocation in a message
type ToolInvocation struct {
	Type      string          `json:"type"`      // "builtin" or "custom"
	ServerID  string          `json:"server_id"`
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool invocation
type ToolResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// DetectToolInvocation detects if a message contains a tool invocation request
func (m *MCPToolIntegration) DetectToolInvocation(message string) (*ToolInvocation, error) {
	// Look for explicit tool invocation patterns
	// Pattern 1: @tool:toolname {args}
	toolPattern := regexp.MustCompile(`@tool:(\w+)\s*({.*?})`)
	if matches := toolPattern.FindStringSubmatch(message); len(matches) > 2 {
		return &ToolInvocation{
			Type:      "builtin",
			ServerID:  "builtin-websearch", // Default to web search for now
			ToolName:  matches[1],
			Arguments: json.RawMessage(matches[2]),
		}, nil
	}
	
	// Pattern 2: Natural language detection for web search
	if m.shouldUseWebSearch(message) {
		query := m.extractSearchQuery(message)
		args, _ := json.Marshal(map[string]interface{}{
			"query":      query,
			"maxResults": 5,
		})
		return &ToolInvocation{
			Type:      "builtin",
			ServerID:  "builtin-websearch",
			ToolName:  "web_search",
			Arguments: args,
		}, nil
	}
	
	// Pattern 3: URL fetch detection
	if url := m.extractURL(message); url != "" {
		args, _ := json.Marshal(map[string]interface{}{
			"url":    url,
			"format": "markdown",
		})
		return &ToolInvocation{
			Type:      "builtin",
			ServerID:  "builtin-websearch",
			ToolName:  "fetch_page",
			Arguments: args,
		}, nil
	}
	
	return nil, nil
}

// shouldUseWebSearch determines if a message should trigger web search
func (m *MCPToolIntegration) shouldUseWebSearch(message string) bool {
	message = strings.ToLower(message)
	
	// Keywords that indicate web search need
	searchKeywords := []string{
		"search for",
		"find information about",
		"what is the latest",
		"recent news about",
		"current status of",
		"look up",
		"research",
		"find out about",
		"what's new with",
		"latest updates on",
		"tell me about",
		"what can you tell me about",
		"information about",
		"details about",
		"news about",
		"what is claude",
		"what is gpt",
		"has been released",
		"was released",
		"is claude",
		"is gpt",
	}
	
	for _, keyword := range searchKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	
	// Questions about current events or recent information
	if strings.Contains(message, "?") {
		timeKeywords := []string{"latest", "recent", "current", "today", "2024", "2025", "now", "new", "update", "news"}
		for _, keyword := range timeKeywords {
			if strings.Contains(message, keyword) {
				return true
			}
		}
	}
	
	// Topics that likely need current information
	techTopics := []string{"gpt-5", "gpt5", "claude", "gemini", "llama", "ai model", "openai", "anthropic", "google ai", "opus", "sonnet", "haiku"}
	for _, topic := range techTopics {
		if strings.Contains(message, topic) {
			return true
		}
	}
	
	// Always search for questions about AI models with version numbers
	if strings.Contains(message, "claude") && (strings.Contains(message, "4.1") || strings.Contains(message, "opus")) {
		return true
	}
	if strings.Contains(message, "gpt") && regexp.MustCompile(`\d`).MatchString(message) {
		return true
	}
	
	return false
}

// extractSearchQuery extracts the search query from a message
func (m *MCPToolIntegration) extractSearchQuery(message string) string {
	message = strings.ToLower(message)
	
	// Remove common prefixes
	prefixes := []string{
		"search for",
		"find information about",
		"look up",
		"research",
		"find out about",
		"what is",
		"what are",
		"tell me about",
		"can you find",
		"please search",
	}
	
	for _, prefix := range prefixes {
		if idx := strings.Index(message, prefix); idx != -1 {
			query := message[idx+len(prefix):]
			query = strings.TrimSpace(query)
			// Remove trailing punctuation
			query = strings.TrimSuffix(query, "?")
			query = strings.TrimSuffix(query, ".")
			query = strings.TrimSuffix(query, "!")
			return query
		}
	}
	
	// If no prefix found, use the whole message (cleaned)
	message = strings.TrimSpace(message)
	message = strings.TrimSuffix(message, "?")
	return message
}

// extractURL extracts a URL from a message
func (m *MCPToolIntegration) extractURL(message string) string {
	// Simple URL regex
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	if matches := urlPattern.FindStringSubmatch(message); len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// InvokeToolForUser invokes a detected tool for a specific user
func (m *MCPToolIntegration) InvokeToolForUser(ctx context.Context, userID uuid.UUID, invocation *ToolInvocation) (*ToolResult, error) {
	if invocation.Type == "builtin" {
		// Ensure the builtin server is enabled for the user
		if !m.builtinManager.IsServerEnabledForUser(userID, invocation.ServerID) {
			// Try to enable it
			if err := m.builtinManager.SetUserServerEnabled(userID, invocation.ServerID, true); err != nil {
				return &ToolResult{
					Success: false,
					Error:   fmt.Sprintf("Failed to enable %s: %v", invocation.ServerID, err),
				}, nil
			}
		}
		
		// Call the tool
		result, err := m.builtinManager.CallTool(userID, invocation.ServerID, invocation.ToolName, invocation.Arguments)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
		
		return &ToolResult{
			Success: true,
			Result:  result,
		}, nil
	}
	
	// Handle custom MCP servers (not implemented yet)
	return &ToolResult{
		Success: false,
		Error:   "Custom MCP servers not yet supported",
	}, nil
}

// FormatToolResultForChat formats a tool result for display in chat
func (m *MCPToolIntegration) FormatToolResultForChat(result *ToolResult, toolName string) string {
	if !result.Success {
		return fmt.Sprintf("⚠️ Tool error: %s", result.Error)
	}
	
	switch toolName {
	case "web_search":
		return m.formatWebSearchResult(result.Result)
	case "fetch_page":
		return m.formatFetchPageResult(result.Result)
	case "search_and_summarize":
		return m.formatSummarizeResult(result.Result)
	default:
		// Generic formatting
		resultJSON, _ := json.MarshalIndent(result.Result, "", "  ")
		return fmt.Sprintf("Tool result:\n```json\n%s\n```", string(resultJSON))
	}
}

// formatWebSearchResult formats web search results for chat
func (m *MCPToolIntegration) formatWebSearchResult(result interface{}) string {
	// Parse the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("Search results: %v", result)
	}
	
	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "No search results found."
	}
	
	// Get the text content
	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return fmt.Sprintf("Search results: %v", result)
	}
	
	text, ok := firstContent["text"].(string)
	if !ok {
		return fmt.Sprintf("Search results: %v", result)
	}
	
	// Parse the JSON text
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(text), &searchData); err != nil {
		return text
	}
	
	results, ok := searchData["results"].([]interface{})
	if !ok {
		return text
	}
	
	// Format results nicely
	var formatted strings.Builder
	formatted.WriteString("🔍 **Search Results:**\n\n")
	
	for i, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		
		title := result["title"].(string)
		url := result["url"].(string)
		snippet := result["snippet"].(string)
		
		formatted.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, title))
		formatted.WriteString(fmt.Sprintf("   %s\n", snippet))
		formatted.WriteString(fmt.Sprintf("   [Link](%s)\n\n", url))
	}
	
	// Add metadata if available
	if metadata, ok := searchData["metadata"].(map[string]interface{}); ok {
		if provider, ok := metadata["provider"].(string); ok {
			formatted.WriteString(fmt.Sprintf("_Source: %s_", provider))
		}
	}
	
	return formatted.String()
}

// formatFetchPageResult formats fetched page content for chat
func (m *MCPToolIntegration) formatFetchPageResult(result interface{}) string {
	// Parse the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("Page content: %v", result)
	}
	
	content, ok := resultMap["content"].(string)
	if ok && content != "" {
		// Truncate if too long
		if len(content) > 1000 {
			content = content[:1000] + "...\n\n_[Content truncated]_"
		}
		return fmt.Sprintf("📄 **Page Content:**\n\n%s", content)
	}
	
	// Try to extract from nested structure
	if contentData, ok := resultMap["content"].(map[string]interface{}); ok {
		if text, ok := contentData["text"].(string); ok {
			if len(text) > 1000 {
				text = text[:1000] + "...\n\n_[Content truncated]_"
			}
			return fmt.Sprintf("📄 **Page Content:**\n\n%s", text)
		}
	}
	
	return fmt.Sprintf("Page fetched: %v", result)
}

// formatSummarizeResult formats summarized content for chat
func (m *MCPToolIntegration) formatSummarizeResult(result interface{}) string {
	// Parse the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("Summary: %v", result)
	}
	
	if summary, ok := resultMap["summary"].(string); ok {
		return fmt.Sprintf("📝 **Summary:**\n\n%s", summary)
	}
	
	// Try nested structure
	if content, ok := resultMap["content"].([]interface{}); ok && len(content) > 0 {
		if firstContent, ok := content[0].(map[string]interface{}); ok {
			if text, ok := firstContent["text"].(string); ok {
				return fmt.Sprintf("📝 **Research Summary:**\n\n%s", text)
			}
		}
	}
	
	return fmt.Sprintf("Summary result: %v", result)
}