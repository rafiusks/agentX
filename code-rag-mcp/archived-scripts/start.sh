#!/bin/bash

# Simple start script for Code RAG MCP Server

set -e

echo "🚀 Starting Code RAG MCP Server..."

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "✓ Using Docker"
    
    # Start Qdrant
    if ! curl -s http://localhost:6333/health > /dev/null 2>&1; then
        echo "Starting Qdrant..."
        docker run -d --name qdrant -p 6333:6333 -p 6334:6334 qdrant/qdrant:latest 2>/dev/null || echo "Qdrant already running"
        sleep 3
    fi
    
    # Build if needed
    if [[ "$(docker images -q code-rag-mcp:latest 2> /dev/null)" == "" ]]; then
        echo "Building server..."
        docker build -t code-rag-mcp:latest . > /dev/null 2>&1
    fi
    
    echo "✓ Server ready!"
    echo ""
    echo "Run interactively:"
    echo "  docker run -i --rm --network host code-rag-mcp:latest"
    echo ""
    echo "Or use docker-compose:"
    echo "  docker-compose up -d"
    
elif command -v go &> /dev/null; then
    echo "✓ Using Go"
    
    # Start Qdrant if possible
    if command -v docker &> /dev/null; then
        if ! curl -s http://localhost:6333/health > /dev/null 2>&1; then
            echo "Starting Qdrant..."
            docker run -d --name qdrant -p 6333:6333 qdrant/qdrant:latest 2>/dev/null || true
            sleep 3
        fi
    fi
    
    echo "✓ Server ready!"
    echo ""
    echo "Run the server:"
    echo "  go run cmd/server/main.go"
    
else
    echo "❌ Neither Docker nor Go found. Please install one of them."
    exit 1
fi

echo ""
echo "📚 Quick test commands:"
echo '  echo '"'"'{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'"'"' | docker run -i --rm --network host code-rag-mcp:latest'
echo ""
echo "📖 See README.md for more details"