#!/bin/bash

# Code RAG - Project-Local Installation Script
# This script installs Code RAG into the current project

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🔍 Code RAG - Project-Local AI Code Search${NC}"
echo ""

# Check if already installed
if [ -d ".code-rag" ]; then
    echo -e "${YELLOW}Code RAG is already installed in this project.${NC}"
    echo "To reinstall, first run: rm -rf .code-rag"
    exit 0
fi

# Create .code-rag directory
echo "Installing Code RAG in current project..."
mkdir -p .code-rag/index

# Download the binary (or build from source)
if command -v go &> /dev/null; then
    echo "Building Code RAG from source..."
    # Build from source
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    git clone --quiet https://github.com/rafael/code-rag-mcp.git
    cd code-rag-mcp
    go build -o code-rag cmd/code-rag/main.go
    mv code-rag "$OLDPWD/.code-rag/"
    cd "$OLDPWD"
    rm -rf "$TEMP_DIR"
else
    echo "Downloading Code RAG binary..."
    # Download pre-built binary
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then
        ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        ARCH="arm64"
    fi
    
    BINARY_URL="https://github.com/rafael/code-rag-mcp/releases/latest/download/code-rag-${OS}-${ARCH}"
    curl -L -o .code-rag/code-rag "$BINARY_URL" 2>/dev/null || {
        echo -e "${YELLOW}Pre-built binary not available. Please install Go and run again.${NC}"
        exit 1
    }
fi

# Make executable
chmod +x .code-rag/code-rag

# Create project config
cat > .code-rag/config.json << EOF
{
  "project_path": "$(pwd)",
  "project_name": "$(basename $(pwd))",
  "indexed": false,
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

# Create wrapper script in project root
cat > code-rag << 'EOF'
#!/bin/bash
# Code RAG wrapper script
exec .code-rag/code-rag "$@"
EOF
chmod +x code-rag

# Add to .gitignore if it exists
if [ -f .gitignore ]; then
    if ! grep -q "^.code-rag/$" .gitignore; then
        echo "" >> .gitignore
        echo "# Code RAG" >> .gitignore
        echo ".code-rag/" >> .gitignore
        echo "code-rag" >> .gitignore
        echo -e "${GREEN}✓ Added .code-rag/ to .gitignore${NC}"
    fi
fi

# Start Qdrant if Docker is available
if command -v docker &> /dev/null; then
    if ! curl -s http://localhost:6333/health > /dev/null 2>&1; then
        echo "Starting vector database..."
        docker run -d --name qdrant -p 6333:6333 qdrant/qdrant:latest > /dev/null 2>&1 || true
    fi
fi

echo ""
echo -e "${GREEN}✨ Code RAG installed successfully!${NC}"
echo ""
echo "Next steps:"
echo "  ./code-rag              # Start and auto-index"
echo "  ./code-rag search 'query'"
echo "  ./code-rag help"
echo ""
echo "Code RAG is now configured for this project only."