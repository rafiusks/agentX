package mcp

import (
	"encoding/json"
	"fmt"
)

func (s *Server) readResource(uri string) (*ReadResourceResult, error) {
	switch uri {
	case "indexed_repositories":
		return s.getIndexedRepositories()
	case "search_statistics":
		return s.getSearchStatistics()
	case "model_capabilities":
		return s.getModelCapabilities()
	default:
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

func (s *Server) getIndexedRepositories() (*ReadResourceResult, error) {
	repos, err := s.ragEngine.GetIndexedRepositories()
	if err != nil {
		return nil, err
	}
	
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return nil, err
	}
	
	return &ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      "indexed_repositories",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *Server) getSearchStatistics() (*ReadResourceResult, error) {
	stats := s.ragEngine.GetStatistics()
	
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, err
	}
	
	return &ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      "search_statistics",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *Server) getModelCapabilities() (*ReadResourceResult, error) {
	capabilities := map[string]interface{}{
		"embedding_models": []map[string]interface{}{
			{
				"name":        "CodeT5",
				"provider":    "HuggingFace",
				"model_id":    "Salesforce/codet5p-110m-embedding",
				"dimensions":  768,
				"max_tokens":  512,
				"languages":   []string{"python", "java", "javascript", "go", "ruby", "php"},
			},
			{
				"name":        "CodeBERT",
				"provider":    "HuggingFace",
				"model_id":    "microsoft/codebert-base",
				"dimensions":  768,
				"max_tokens":  512,
				"languages":   []string{"python", "java", "javascript", "go", "ruby", "php"},
			},
			{
				"name":        "OpenAI",
				"provider":    "OpenAI",
				"model_id":    "text-embedding-3-large",
				"dimensions":  3072,
				"max_tokens":  8191,
				"languages":   []string{"all"},
			},
		},
		"chunking_strategies": []string{
			"ast_aware",
			"sliding_window",
			"semantic_boundaries",
		},
		"search_modes": []string{
			"semantic",
			"keyword",
			"hybrid",
			"structural",
		},
		"supported_languages": []string{
			"go",
			"javascript",
			"typescript",
			"python",
			"rust",
			"java",
			"c",
			"cpp",
		},
	}
	
	data, err := json.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		return nil, err
	}
	
	return &ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      "model_capabilities",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}