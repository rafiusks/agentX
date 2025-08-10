#!/bin/bash

# Test script for Code RAG MCP Server

echo "Testing Code RAG MCP Server..."
echo ""

# Test 1: Initialize
echo "Test 1: Initialize"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' | go run cmd/server/main.go 2>/dev/null | jq .
echo ""

# Test 2: List tools
echo "Test 2: List available tools"
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | go run cmd/server/main.go 2>/dev/null | jq '.result.tools[].name'
echo ""

# Test 3: List resources
echo "Test 3: List available resources"
echo '{"jsonrpc":"2.0","id":3,"method":"resources/list","params":{}}' | go run cmd/server/main.go 2>/dev/null | jq '.result.resources[].name'
echo ""

# Test 4: Call code_search tool
echo "Test 4: Search for code"
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"code_search","arguments":{"query":"function implementation","language":"go","limit":5}}}' | go run cmd/server/main.go 2>/dev/null | jq '.result.content[0].text' | head -20
echo ""

echo "Tests completed!"