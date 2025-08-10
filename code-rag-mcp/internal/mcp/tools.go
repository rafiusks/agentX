package mcp

import (
	"context"
	"fmt"
	"time"
)

func (s *Server) executeTool(name string, args map[string]interface{}) (*CallToolResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	switch name {
	case "code_search":
		return s.executeCodeSearch(ctx, args)
	case "explain_code":
		return s.executeExplainCode(ctx, args)
	case "find_similar":
		return s.executeFindSimilar(ctx, args)
	case "index_repository":
		return s.executeIndexRepository(ctx, args)
	case "get_dependencies":
		return s.executeGetDependencies(ctx, args)
	case "suggest_improvements":
		return s.executeSuggestImprovements(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) executeCodeSearch(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}
	
	language := "any"
	if lang, ok := args["language"].(string); ok {
		language = lang
	}
	
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	results, err := s.ragEngine.Search(ctx, query, language, limit)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error performing search: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	if len(results) == 0 {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "No results found for your query.",
			}},
		}, nil
	}
	
	var responseText string
	for i, result := range results {
		responseText += fmt.Sprintf("## Result %d (Score: %.2f)\n", i+1, result.Score)
		responseText += fmt.Sprintf("**File:** %s\n", result.FilePath)
		if result.LineStart > 0 {
			responseText += fmt.Sprintf("**Lines:** %d-%d\n", result.LineStart, result.LineEnd)
		}
		responseText += fmt.Sprintf("```%s\n%s\n```\n\n", result.Language, result.Code)
	}
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: responseText,
		}},
	}, nil
}

func (s *Server) executeExplainCode(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	code, ok := args["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code parameter is required")
	}
	
	filePath := ""
	if fp, ok := args["file_path"].(string); ok {
		filePath = fp
	}
	
	explanation, err := s.ragEngine.ExplainCode(ctx, code, filePath)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error explaining code: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: explanation,
		}},
	}, nil
}

func (s *Server) executeFindSimilar(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	code, ok := args["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code parameter is required")
	}
	
	threshold := 0.7
	if t, ok := args["threshold"].(float64); ok {
		threshold = t
	}
	
	results, err := s.ragEngine.FindSimilar(ctx, code, threshold)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error finding similar code: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	if len(results) == 0 {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "No similar code found.",
			}},
		}, nil
	}
	
	var responseText string
	for i, result := range results {
		responseText += fmt.Sprintf("## Similar Code %d (Similarity: %.2f)\n", i+1, result.Score)
		responseText += fmt.Sprintf("**File:** %s\n", result.FilePath)
		responseText += fmt.Sprintf("```%s\n%s\n```\n\n", result.Language, result.Code)
	}
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: responseText,
		}},
	}, nil
}

func (s *Server) executeIndexRepository(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	incremental := true
	if inc, ok := args["incremental"].(bool); ok {
		incremental = inc
	}
	
	forceClean := false
	if fc, ok := args["force_clean"].(bool); ok {
		forceClean = fc
	}
	
	stats, err := s.ragEngine.IndexRepositoryWithOptions(ctx, path, incremental, forceClean)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error indexing repository: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	responseText := fmt.Sprintf("Repository indexed successfully!\n\n")
	responseText += fmt.Sprintf("- Files processed: %d\n", stats.FilesProcessed)
	responseText += fmt.Sprintf("- Chunks created: %d\n", stats.ChunksCreated)
	responseText += fmt.Sprintf("- Time taken: %s\n", stats.Duration)
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: responseText,
		}},
	}, nil
}

func (s *Server) executeGetDependencies(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter is required")
	}
	
	includeTransitive := false
	if it, ok := args["include_transitive"].(bool); ok {
		includeTransitive = it
	}
	
	deps, err := s.ragEngine.GetDependencies(ctx, filePath, includeTransitive)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error analyzing dependencies: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	var responseText string
	responseText += fmt.Sprintf("# Dependencies for %s\n\n", filePath)
	
	if len(deps.Imports) > 0 {
		responseText += "## Imports\n"
		for _, imp := range deps.Imports {
			responseText += fmt.Sprintf("- %s\n", imp)
		}
		responseText += "\n"
	}
	
	if len(deps.Exports) > 0 {
		responseText += "## Exports\n"
		for _, exp := range deps.Exports {
			responseText += fmt.Sprintf("- %s\n", exp)
		}
		responseText += "\n"
	}
	
	if len(deps.Functions) > 0 {
		responseText += "## Functions\n"
		for _, fn := range deps.Functions {
			responseText += fmt.Sprintf("- %s\n", fn)
		}
		responseText += "\n"
	}
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: responseText,
		}},
	}, nil
}

func (s *Server) executeSuggestImprovements(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	code, ok := args["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code parameter is required")
	}
	
	focus := "all"
	if f, ok := args["focus"].(string); ok {
		focus = f
	}
	
	suggestions, err := s.ragEngine.SuggestImprovements(ctx, code, focus)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error generating suggestions: %v", err),
			}},
			IsError: true,
		}, nil
	}
	
	return &CallToolResult{
		Content: []Content{{
			Type: "text",
			Text: suggestions,
		}},
	}, nil
}