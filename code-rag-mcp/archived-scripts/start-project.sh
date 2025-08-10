#!/bin/bash

# Start script for project-specific Code RAG MCP Server

set -e

PROJECT_NAME="${1:-default}"
PROJECT_PATH="${2:-$(pwd)}"
PORT="${3:-3333}"

echo "🚀 Starting Code RAG MCP for project: $PROJECT_NAME"
echo "📁 Project path: $PROJECT_PATH"
echo "🔌 Port: $PORT"

# Create project-specific config
CONFIG_DIR="$HOME/.config/code-rag-mcp/projects/$PROJECT_NAME"
mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  port: $PORT
  project_name: $PROJECT_NAME
  project_path: $PROJECT_PATH

vectordb:
  collection_name: "code_rag_${PROJECT_NAME}"
  
search:
  project_filter: $PROJECT_NAME
EOF

# Start Qdrant if needed
if ! curl -s http://localhost:6333/health > /dev/null 2>&1; then
    echo "Starting Qdrant..."
    docker run -d --name qdrant -p 6333:6333 qdrant/qdrant:latest 2>/dev/null || true
    sleep 3
fi

# Run project-specific instance
echo "Starting MCP server for $PROJECT_NAME..."
docker run -d \
    --name "code-rag-mcp-$PROJECT_NAME" \
    --network host \
    -e PROJECT_NAME="$PROJECT_NAME" \
    -e PROJECT_PATH="$PROJECT_PATH" \
    -e CONFIG_PATH="/config/config.yaml" \
    -v "$CONFIG_DIR:/config:ro" \
    -v "$PROJECT_PATH:$PROJECT_PATH:ro" \
    code-rag-mcp:latest

echo "✅ Server running for project: $PROJECT_NAME"
echo ""
echo "Claude Code config for this project:"
echo "{"
echo '  "mcpServers": {'
echo "    \"code-rag-$PROJECT_NAME\": {"
echo '      "command": "docker",'
echo "      \"args\": [\"run\", \"-i\", \"--rm\", \"--network\", \"host\", \"-e\", \"PROJECT_NAME=$PROJECT_NAME\", \"code-rag-mcp:latest\"]"
echo '    }'
echo '  }'
echo '}'