package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/providers"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
)

// Provider implements the Anthropic provider
type Provider struct {
	id     string
	config config.ProviderConfig
	client *http.Client
}

// AnthropicRequest represents a request to Anthropic's API
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []AnthropicMessage       `json:"messages"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature *float32                 `json:"temperature,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	System      string                   `json:"system,omitempty"`
	Tools       []AnthropicTool          `json:"tools,omitempty"`
	ToolChoice  *AnthropicToolChoice     `json:"tool_choice,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string                   `json:"role"`
	Content []AnthropicContent       `json:"content"`
}

// AnthropicContent represents content in a message
type AnthropicContent struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     json.RawMessage        `json:"input,omitempty"`
}

// AnthropicTool represents a tool definition
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AnthropicToolChoice represents tool selection
type AnthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// AnthropicResponse represents a response from Anthropic's API
type AnthropicResponse struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Content      []AnthropicContent  `json:"content"`
	Model        string              `json:"model"`
	StopReason   string              `json:"stop_reason,omitempty"`
	StopSequence string              `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage      `json:"usage"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent represents a streaming event
type AnthropicStreamEvent struct {
	Type  string                   `json:"type"`
	Index int                      `json:"index,omitempty"`
	Delta *AnthropicStreamDelta    `json:"delta,omitempty"`
	Message *AnthropicResponse       `json:"message,omitempty"`
	ContentBlock *AnthropicContent   `json:"content_block,omitempty"`
	Usage *AnthropicUsage           `json:"usage,omitempty"`
}

// AnthropicStreamDelta represents a delta in streaming
type AnthropicStreamDelta struct {
	Type         string          `json:"type"`
	Text         string          `json:"text,omitempty"`
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	PartialJSON  json.RawMessage `json:"partial_json,omitempty"`
}

// NewProvider creates a new Anthropic provider
func NewProvider(id string, cfg config.ProviderConfig) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("Anthropic API key is required")
	}

	return &Provider{
		id:     id,
		config: cfg,
		client: &http.Client{},
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.config.Name
}

// Complete performs a non-streaming completion
func (p *Provider) Complete(ctx context.Context, req providers.CompletionRequest) (*providers.CompletionResponse, error) {
	anthropicReq := p.convertRequest(req)
	anthropicReq.Stream = false

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, err
	}

	return p.convertResponse(&anthropicResp), nil
}

// StreamComplete performs a streaming completion
func (p *Provider) StreamComplete(ctx context.Context, req providers.CompletionRequest) (<-chan providers.StreamChunk, error) {
	chunks := make(chan providers.StreamChunk)

	go func() {
		defer close(chunks)

		anthropicReq := p.convertRequest(req)
		anthropicReq.Stream = true

		body, err := json.Marshal(anthropicReq)
		if err != nil {
			chunks <- providers.StreamChunk{Error: err.Error()}
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
		if err != nil {
			chunks <- providers.StreamChunk{Error: err.Error()}
			return
		}

		p.setHeaders(httpReq)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			chunks <- providers.StreamChunk{Error: err.Error()}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			chunks <- providers.StreamChunk{Error: fmt.Sprintf("Anthropic API error: %s - %s", resp.Status, string(bodyBytes))}
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				chunks <- providers.StreamChunk{Error: err.Error()}
				return
			}

			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- providers.StreamChunk{FinishReason: "stop"}
				break
			}

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue // Skip malformed events
			}

			chunk := p.convertStreamEvent(&event)
			if chunk != nil {
				chunks <- *chunk
			}
		}
	}()

	return chunks, nil
}

// GetModels returns available models
func (p *Provider) GetModels(ctx context.Context) ([]providers.Model, error) {
	// Anthropic doesn't have a models endpoint, so we return hardcoded models
	models := []providers.Model{
		{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-sonnet-20240229", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-haiku-20240307", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-2.1", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-2.0", Object: "model", OwnedBy: "anthropic"},
	}
	return models, nil
}

// ValidateConfig validates the provider configuration
func (p *Provider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return errors.New("API key is required")
	}
	return nil
}

// setHeaders sets the required headers for Anthropic API
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
}

// convertRequest converts internal request to Anthropic request
func (p *Provider) convertRequest(req providers.CompletionRequest) AnthropicRequest {
	anthropicReq := AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   4096, // Default max tokens
		Temperature: req.Temperature,
	}

	if req.MaxTokens != nil {
		anthropicReq.MaxTokens = *req.MaxTokens
	}

	// Convert messages
	anthropicMessages := []AnthropicMessage{}
	var systemMessage string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
			continue
		}

		content := []AnthropicContent{{Type: "text", Text: msg.Content}}
		
		// Handle tool calls in assistant messages
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				content = append(content, AnthropicContent{
					Type: "tool_use",
					ID:   tc.ID,
					Name: tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				})
			}
		}

		// Handle tool responses
		if msg.Role == "tool" && msg.ToolCallID != "" {
			content = []AnthropicContent{{
				Type:  "tool_result",
				ID:    msg.ToolCallID,
				Input: json.RawMessage(msg.Content),
			}}
			msg.Role = "user" // Anthropic expects tool results as user messages
		}

		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	anthropicReq.Messages = anthropicMessages
	if systemMessage != "" {
		anthropicReq.System = systemMessage
	}

	// Convert tools
	if len(req.Functions) > 0 || len(req.Tools) > 0 {
		anthropicReq.Tools = []AnthropicTool{}
		
		// Convert functions to tools
		for _, fn := range req.Functions {
			anthropicReq.Tools = append(anthropicReq.Tools, AnthropicTool{
				Name:        fn.Name,
				Description: fn.Description,
				InputSchema: fn.Parameters,
			})
		}

		// Add tools
		for _, tool := range req.Tools {
			anthropicReq.Tools = append(anthropicReq.Tools, AnthropicTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			})
		}
	}

	// Convert tool choice
	if req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "auto":
			anthropicReq.ToolChoice = &AnthropicToolChoice{Type: "auto"}
		case "none":
			anthropicReq.ToolChoice = &AnthropicToolChoice{Type: "none"}
		case "function":
			if req.ToolChoice.Function != nil {
				anthropicReq.ToolChoice = &AnthropicToolChoice{
					Type: "tool",
					Name: req.ToolChoice.Function.Name,
				}
			}
		}
	}

	return anthropicReq
}

// convertResponse converts Anthropic response to internal response
func (p *Provider) convertResponse(resp *AnthropicResponse) *providers.CompletionResponse {
	message := providers.Message{
		Role: "assistant",
	}

	// Process content blocks
	var textContent strings.Builder
	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			textContent.WriteString(content.Text)
		case "tool_use":
			if message.ToolCalls == nil {
				message.ToolCalls = []providers.ToolCall{}
			}
			message.ToolCalls = append(message.ToolCalls, providers.ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: providers.FunctionCall{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			})
		}
	}

	message.Content = textContent.String()

	return &providers.CompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: 0, // Anthropic doesn't provide creation time
		Model:   resp.Model,
		Choices: []providers.Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: p.convertStopReason(resp.StopReason),
			},
		},
		Usage: providers.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// convertStreamEvent converts Anthropic stream event to internal stream chunk
func (p *Provider) convertStreamEvent(event *AnthropicStreamEvent) *providers.StreamChunk {
	switch event.Type {
	case "message_start":
		if event.Message != nil {
			return &providers.StreamChunk{
				ID:      event.Message.ID,
				Object:  "chat.completion.chunk",
				Model:   event.Message.Model,
				Role:    "assistant",
			}
		}

	case "content_block_start":
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			return &providers.StreamChunk{
				ToolCalls: []providers.ToolCall{
					{
						ID:   event.ContentBlock.ID,
						Type: "function",
						Function: providers.FunctionCall{
							Name: event.ContentBlock.Name,
						},
					},
				},
			}
		}

	case "content_block_delta":
		if event.Delta != nil {
			if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				return &providers.StreamChunk{
					Delta: event.Delta.Text,
				}
			} else if event.Delta.Type == "input_json_delta" && len(event.Delta.PartialJSON) > 0 {
				// Handle partial JSON for tool calls
				return &providers.StreamChunk{
					ToolCalls: []providers.ToolCall{
						{
							Function: providers.FunctionCall{
								Arguments: string(event.Delta.PartialJSON),
							},
						},
					},
				}
			}
		}

	case "message_delta":
		if event.Delta != nil && event.Delta.StopReason != "" {
			return &providers.StreamChunk{
				FinishReason: p.convertStopReason(event.Delta.StopReason),
			}
		}

	case "message_stop":
		return &providers.StreamChunk{
			FinishReason: "stop",
		}
	}

	return nil
}

// convertStopReason converts Anthropic stop reason to OpenAI format
func (p *Provider) convertStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}