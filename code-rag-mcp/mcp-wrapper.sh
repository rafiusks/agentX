#!/bin/bash

# MCP Wrapper Script - Ensures services are running before starting MCP server

# Log to stderr for debugging
echo "[MCP Wrapper] Starting..." >&2

# Check if Code RAG services are running
if ! curl -s http://localhost:6333/collections > /dev/null 2>&1; then
    echo "[MCP Wrapper] Starting Code RAG services..." >&2
    (cd /Users/rafael/Code/agentX/code-rag-mcp && docker-compose up -d) >&2
    
    # Wait for services to be ready
    echo "[MCP Wrapper] Waiting for services..." >&2
    for i in {1..30}; do
        if curl -s http://localhost:6333/collections > /dev/null 2>&1 && curl -s http://localhost:8001/health > /dev/null 2>&1; then
            echo "[MCP Wrapper] Services ready" >&2
            break
        fi
        sleep 1
    done
else
    echo "[MCP Wrapper] Services already running" >&2
fi

# Change to the correct directory for config loading
cd /Users/rafael/Code/agentX/code-rag-mcp

echo "[MCP Wrapper] Starting MCP server..." >&2

# Run the MCP server in stdio mode
exec /Users/rafael/Code/agentX/code-rag-mcp/.code-rag/code-rag mcp-server