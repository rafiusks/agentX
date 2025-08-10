package main

import (
	"encoding/json"
	"os"
)

func main() {
	// Test JSON response with correct protocol version
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
				"logging":   map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "code-rag-mcp",
				"version": "1.0.0",
			},
		},
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(response)
}