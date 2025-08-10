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
	Success  bool        `json:"success"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"` // Additional metadata for UI
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
		// Determine if we should fetch full content based on the query
		includeContent := m.shouldFetchFullContent(message)
		maxResults := 5
		if includeContent {
			maxResults = 3 // Reduce number when fetching full content
		}
		
		args, _ := json.Marshal(map[string]interface{}{
			"query":         query,
			"maxResults":    maxResults,
			"includeContent": includeContent,
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

// shouldFetchFullContent determines if we should fetch full page content
func (m *MCPToolIntegration) shouldFetchFullContent(message string) bool {
	message = strings.ToLower(message)
	
	// Keywords that indicate need for detailed information
	detailKeywords := []string{
		"explain",
		"describe",
		"tell me about",
		"what is",
		"how does",
		"how to",
		"details about",
		"information about",
		"deep dive",
		"comprehensive",
		"full",
		"complete",
		"documentation",
		"tutorial",
		"guide",
		"research",
		"analyze",
		"understand",
	}
	
	for _, keyword := range detailKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	
	// If asking about specific technical topics, fetch full content
	technicalTopics := []string{
		"api", "sdk", "implementation", "architecture", 
		"algorithm", "code", "programming", "technical",
		"specification", "configuration", "setup",
	}
	
	for _, topic := range technicalTopics {
		if strings.Contains(message, topic) {
			return true
		}
	}
	
	// Default to just snippets for simple queries
	return false
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
		"how is",
		"what happened",
		"any updates",
		"what's happening with",
		"status of",
		"explain",
		"describe",
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
	techTopics := []string{"gpt-5", "gpt5", "claude", "gemini", "llama", "ai model", "openai", "anthropic", "google ai", "opus", "sonnet", "haiku", "mistral", "qwen", "deepseek", "kimi", "moonshot"}
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
	
	// Always search for anything mentioning specific years (likely needs current info)
	yearPattern := regexp.MustCompile(`20\d{2}`)
	if yearPattern.MatchString(message) {
		return true
	}
	
	// Search for questions (?) about entities that change frequently
	if strings.Contains(message, "?") {
		volatileTopics := []string{"price", "stock", "weather", "president", "champion", "winner", "release", "launch", "announce"}
		for _, topic := range volatileTopics {
			if strings.Contains(message, topic) {
				return true
			}
		}
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
		
		// Extract metadata from result if it's a web search
		var metadata interface{}
		if invocation.ToolName == "web_search" {
			metadata = m.extractSearchMetadata(result)
		}
		
		return &ToolResult{
			Success:  true,
			Result:   result,
			Metadata: metadata,
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
		return ""
	}
	
	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		return ""
	}
	
	// Get the text content
	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return ""
	}
	
	text, ok := firstContent["text"].(string)
	if !ok {
		return ""
	}
	
	// Clean the text first before parsing
	text = m.cleanPageContent(text, len(text))
	
	// Parse the JSON text
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(text), &searchData); err != nil {
		// If we can't parse as JSON, it might be raw text content - don't return it
		m.logger.Warnf("Failed to parse search result as JSON: %v", err)
		return ""
	}
	
	results, ok := searchData["results"].([]interface{})
	if !ok {
		// Don't return raw text as it might contain processing notes
		return "Search results retrieved."
	}
	
	// Format results as reference material with clear sources
	var formatted strings.Builder
	
	// Don't add any prefix if we don't have valid results
	if len(results) == 0 {
		return ""
	}
	
	formatted.WriteString("Based on current web information:\n\n")
	
	for i, r := range results {
		if i >= 5 { // Limit to 5 results to avoid overwhelming
			break
		}
		result, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		
		title, _ := result["title"].(string)
		url, _ := result["url"].(string)
		snippet, _ := result["snippet"].(string)
		
		// Check if we have full content
		if contentData, ok := result["content"].(map[string]interface{}); ok {
			// We have full page content, extract the most relevant part
			if markdown, ok := contentData["markdown"].(string); ok && len(markdown) > 0 {
				// Limit content length and clean it
				cleanContent := m.cleanPageContent(markdown, 500) // Max 500 chars per result
				if len(cleanContent) > 50 { // Only use if we have meaningful content
					snippet = cleanContent
				}
			} else if plainText, ok := contentData["plainText"].(string); ok && len(plainText) > 0 {
				// Fallback to plain text if markdown not available
				cleanContent := m.cleanPageContent(plainText, 500)
				if len(cleanContent) > 50 {
					snippet = cleanContent
				}
			}
		}
		
		// Extract domain from URL for source display
		domain := url
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			if parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"), "/"); len(parts) > 0 {
				domain = parts[0]
			}
		}
		
		// Format with clear source attribution that LLM should preserve
		formatted.WriteString(fmt.Sprintf("• According to %s: %s\n", domain, snippet))
	}
	
	// Add metadata if available
	if metadata, ok := searchData["metadata"].(map[string]interface{}); ok {
		if provider, ok := metadata["provider"].(string); ok {
			formatted.WriteString(fmt.Sprintf("_Source: %s_", provider))
		}
	}
	
	// Final safety check - clean the output one more time
	finalOutput := formatted.String()
	finalOutput = strings.ReplaceAll(finalOutput, "....???........?......?................................ .....", "")
	finalOutput = strings.ReplaceAll(finalOutput, "We need answer. Use sources.", "")
	finalOutput = strings.TrimSpace(finalOutput)
	
	return finalOutput
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

// extractSearchMetadata extracts structured metadata from search results
func (m *MCPToolIntegration) extractSearchMetadata(result interface{}) map[string]interface{} {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	
	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		return nil
	}
	
	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return nil
	}
	
	text, ok := firstContent["text"].(string)
	if !ok {
		return nil
	}
	
	// Parse the JSON text
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(text), &searchData); err != nil {
		return nil
	}
	
	results, ok := searchData["results"].([]interface{})
	if !ok {
		return nil
	}
	
	// Build structured metadata
	metadata := map[string]interface{}{
		"type":        "web_search",
		"provider":    "DuckDuckGo",
		"resultCount": len(results),
		"results":     results,
	}
	
	if meta, ok := searchData["metadata"].(map[string]interface{}); ok {
		if provider, ok := meta["provider"].(string); ok {
			metadata["provider"] = provider
		}
	}
	
	return metadata
}

// cleanPageContent removes unwanted content and limits length
func (m *MCPToolIntegration) cleanPageContent(content string, maxLength int) string {
	// First, remove any obvious processing instruction patterns
	content = strings.ReplaceAll(content, "....???........?......?................................ .....", "")
	content = strings.ReplaceAll(content, "We need answer. Use sources.", "")
	
	// Remove any lines that look like processing instructions or meta content
	lines := strings.Split(content, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowerTrimmed := strings.ToLower(trimmed)
		
		// Skip lines that look like AI processing notes or meta instructions
		if strings.Contains(lowerTrimmed, "we need") ||
			strings.Contains(lowerTrimmed, "use sources") ||
			strings.Contains(lowerTrimmed, "probably ask") ||
			strings.Contains(lowerTrimmed, "maybe answer") ||
			strings.Contains(lowerTrimmed, "quick take-away") ||
			strings.Contains(lowerTrimmed, "they pasted") ||
			strings.Contains(lowerTrimmed, "respond concisely") ||
			strings.Contains(lowerTrimmed, "concise overview") ||
			strings.Contains(lowerTrimmed, "provide key points") ||
			strings.Contains(lowerTrimmed, "summary of") ||
			strings.HasPrefix(trimmed, "#msg-") ||
			strings.HasPrefix(trimmed, "#") && strings.Contains(lowerTrimmed, "comp") ||
			strings.Contains(trimmed, "???") ||
			strings.Contains(trimmed, "....") && strings.Contains(trimmed, "?") ||
			strings.HasPrefix(trimmed, "..............") ||
			strings.HasPrefix(trimmed, ".. ") ||
			len(trimmed) == 0 {
			continue
		}
		
		cleanLines = append(cleanLines, line)
	}
	
	cleaned := strings.Join(cleanLines, " ")
	cleaned = strings.TrimSpace(cleaned)
	
	// Limit length
	if len(cleaned) > maxLength {
		// Try to cut at a sentence boundary
		if idx := strings.LastIndex(cleaned[:maxLength], ". "); idx > 0 {
			cleaned = cleaned[:idx+1]
		} else if idx := strings.LastIndex(cleaned[:maxLength], " "); idx > 0 {
			// Cut at word boundary
			cleaned = cleaned[:idx] + "..."
		} else {
			cleaned = cleaned[:maxLength] + "..."
		}
	}
	
	return cleaned
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