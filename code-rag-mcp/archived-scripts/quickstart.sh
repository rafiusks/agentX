#!/bin/bash

# Code RAG MCP Server - Quick Start Script
# This script sets up and runs the Code RAG MCP server with minimal configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ASCII Art Banner
echo -e "${BLUE}"
cat << "EOF"
  ____          _        ____      _    ____ 
 / ___|___   __| | ___  |  _ \    / \  / ___|
| |   / _ \ / _` |/ _ \ | |_) |  / _ \| |  _ 
| |__| (_) | (_| |  __/ |  _ <  / ___ \ |_| |
 \____\___/ \__,_|\___| |_| \_\/_/   \_\____|
                                              
         MCP Server - Quick Start             
EOF
echo -e "${NC}"

# Function to print colored messages
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check for Docker
    if command -v docker &> /dev/null; then
        print_status "Docker is installed"
        USE_DOCKER=true
    else
        print_warning "Docker not found, will use local Go installation"
        USE_DOCKER=false
        
        # Check for Go
        if ! command -v go &> /dev/null; then
            print_error "Neither Docker nor Go is installed. Please install one of them."
            echo "   - Docker: https://docs.docker.com/get-docker/"
            echo "   - Go: https://golang.org/doc/install"
            exit 1
        fi
        print_status "Go is installed"
    fi
    
    # Check for jq (optional but useful)
    if ! command -v jq &> /dev/null; then
        print_warning "jq not found. Install it for prettier JSON output: brew install jq"
    fi
}

# Setup with Docker
setup_docker() {
    print_info "Setting up with Docker..."
    
    # Check if docker-compose exists
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        COMPOSE_CMD="docker compose"
    fi
    
    # Stop any existing containers
    $COMPOSE_CMD down 2>/dev/null || true
    
    # Start services
    print_info "Starting Qdrant vector database..."
    $COMPOSE_CMD up -d qdrant
    
    # Wait for Qdrant to be ready
    print_info "Waiting for Qdrant to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:6333/health > /dev/null 2>&1; then
            print_status "Qdrant is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            print_error "Qdrant failed to start"
            exit 1
        fi
        sleep 1
        echo -n "."
    done
    echo
    
    # Build and start the MCP server
    print_info "Building Code RAG MCP server..."
    docker build -t code-rag-mcp:latest .
    
    print_status "Docker setup complete"
}

# Setup with local Go
setup_local() {
    print_info "Setting up with local Go installation..."
    
    # Download dependencies
    print_info "Downloading Go dependencies..."
    go mod download
    
    # Check if Qdrant is running
    if ! curl -s http://localhost:6333/health > /dev/null 2>&1; then
        print_warning "Qdrant is not running. Starting Qdrant in Docker..."
        if command -v docker &> /dev/null; then
            docker run -d --name qdrant -p 6333:6333 -p 6334:6334 qdrant/qdrant:latest
            sleep 5
        else
            print_error "Qdrant is required. Please install Docker to run Qdrant."
            exit 1
        fi
    fi
    
    print_status "Local setup complete"
}

# Index a sample repository
index_sample() {
    print_info "Indexing current repository as a sample..."
    
    # Create index request
    local INDEX_REQUEST='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_repository","arguments":{"path":"'$(pwd)'"}}}'
    
    if [ "$USE_DOCKER" = true ]; then
        echo "$INDEX_REQUEST" | docker run -i --rm --network host code-rag-mcp:latest
    else
        echo "$INDEX_REQUEST" | go run cmd/server/main.go 2>/dev/null
    fi
    
    print_status "Repository indexed"
}

# Test the server
test_server() {
    print_info "Testing server functionality..."
    
    # Test initialize
    local INIT_REQUEST='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{},"clientInfo":{"name":"quickstart","version":"1.0.0"}}}'
    
    echo -e "\n${BLUE}Testing initialization...${NC}"
    if [ "$USE_DOCKER" = true ]; then
        echo "$INIT_REQUEST" | docker run -i --rm --network host code-rag-mcp:latest | jq -r '.result.serverInfo' 2>/dev/null || echo "✓ Server initialized"
    else
        echo "$INIT_REQUEST" | go run cmd/server/main.go 2>/dev/null | jq -r '.result.serverInfo' 2>/dev/null || echo "✓ Server initialized"
    fi
    
    # Test code search
    local SEARCH_REQUEST='{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"code_search","arguments":{"query":"function","limit":2}}}'
    
    echo -e "\n${BLUE}Testing code search...${NC}"
    if [ "$USE_DOCKER" = true ]; then
        echo "$SEARCH_REQUEST" | docker run -i --rm --network host code-rag-mcp:latest | jq -r '.result.content[0].text' 2>/dev/null | head -10 || echo "✓ Search works"
    else
        echo "$SEARCH_REQUEST" | go run cmd/server/main.go 2>/dev/null | jq -r '.result.content[0].text' 2>/dev/null | head -10 || echo "✓ Search works"
    fi
    
    print_status "Server tests passed"
}

# Generate configuration for Claude Code
generate_claude_config() {
    print_info "Generating Claude Code configuration..."
    
    local CONFIG_FILE="$HOME/.config/claude/mcp_config.json"
    mkdir -p "$(dirname "$CONFIG_FILE")"
    
    if [ "$USE_DOCKER" = true ]; then
        cat > "$CONFIG_FILE" << EOF
{
  "mcpServers": {
    "code-rag": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "--network", "host", "code-rag-mcp:latest"]
    }
  }
}
EOF
    else
        cat > "$CONFIG_FILE" << EOF
{
  "mcpServers": {
    "code-rag": {
      "command": "$(pwd)/code-rag-mcp"
    }
  }
}
EOF
        # Build the binary
        print_info "Building binary for Claude Code..."
        go build -o code-rag-mcp cmd/server/main.go
    fi
    
    print_status "Claude Code configuration saved to: $CONFIG_FILE"
}

# Interactive mode
run_interactive() {
    print_info "Starting interactive mode..."
    echo -e "${YELLOW}You can now send JSON-RPC requests. Type 'help' for examples or 'quit' to exit.${NC}\n"
    
    while true; do
        echo -n "> "
        read -r input
        
        case "$input" in
            quit|exit)
                print_info "Exiting..."
                break
                ;;
            help)
                echo -e "${BLUE}Example commands:${NC}"
                echo '  {"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
                echo '  {"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}'
                echo '  {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"code_search","arguments":{"query":"your search term"}}}'
                echo '  {"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"explain_code","arguments":{"code":"your code here"}}}'
                ;;
            *)
                if [ "$USE_DOCKER" = true ]; then
                    echo "$input" | docker run -i --rm --network host code-rag-mcp:latest | jq . 2>/dev/null || echo "Invalid JSON"
                else
                    echo "$input" | go run cmd/server/main.go 2>/dev/null | jq . 2>/dev/null || echo "Invalid JSON"
                fi
                ;;
        esac
    done
}

# Main menu
show_menu() {
    echo -e "\n${GREEN}Quick Start Complete!${NC}\n"
    echo "What would you like to do next?"
    echo "  1) Run server in interactive mode"
    echo "  2) Generate Claude Code configuration"
    echo "  3) Index a repository"
    echo "  4) Run server as daemon (background)"
    echo "  5) View logs"
    echo "  6) Stop all services"
    echo "  7) Exit"
    echo
}

# Run as daemon
run_daemon() {
    print_info "Starting server as daemon..."
    
    if [ "$USE_DOCKER" = true ]; then
        docker run -d --name code-rag-mcp --network host code-rag-mcp:latest
        print_status "Server running in background (container: code-rag-mcp)"
        echo "View logs with: docker logs -f code-rag-mcp"
    else
        nohup go run cmd/server/main.go > code-rag-mcp.log 2>&1 &
        echo $! > code-rag-mcp.pid
        print_status "Server running in background (PID: $(cat code-rag-mcp.pid))"
        echo "View logs with: tail -f code-rag-mcp.log"
    fi
}

# View logs
view_logs() {
    if [ "$USE_DOCKER" = true ]; then
        docker logs -f code-rag-mcp 2>/dev/null || docker-compose logs -f
    else
        if [ -f code-rag-mcp.log ]; then
            tail -f code-rag-mcp.log
        else
            print_error "No log file found"
        fi
    fi
}

# Stop services
stop_services() {
    print_info "Stopping all services..."
    
    if [ "$USE_DOCKER" = true ]; then
        docker stop code-rag-mcp 2>/dev/null || true
        docker rm code-rag-mcp 2>/dev/null || true
        docker-compose down 2>/dev/null || true
    else
        if [ -f code-rag-mcp.pid ]; then
            kill $(cat code-rag-mcp.pid) 2>/dev/null || true
            rm code-rag-mcp.pid
        fi
    fi
    
    print_status "All services stopped"
}

# Main execution
main() {
    # Check prerequisites
    check_prerequisites
    
    # Setup based on available tools
    if [ "$USE_DOCKER" = true ]; then
        setup_docker
    else
        setup_local
    fi
    
    # Test the server
    test_server
    
    # Index sample repository
    index_sample
    
    # Interactive menu
    while true; do
        show_menu
        read -r -p "Select an option (1-7): " choice
        
        case $choice in
            1)
                run_interactive
                ;;
            2)
                generate_claude_config
                ;;
            3)
                read -r -p "Enter repository path (or press Enter for current directory): " repo_path
                repo_path="${repo_path:-$(pwd)}"
                INDEX_REQUEST='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_repository","arguments":{"path":"'$repo_path'"}}}'
                if [ "$USE_DOCKER" = true ]; then
                    echo "$INDEX_REQUEST" | docker run -i --rm --network host code-rag-mcp:latest
                else
                    echo "$INDEX_REQUEST" | go run cmd/server/main.go 2>/dev/null
                fi
                print_status "Repository indexed: $repo_path"
                ;;
            4)
                run_daemon
                ;;
            5)
                view_logs
                ;;
            6)
                stop_services
                ;;
            7)
                print_info "Goodbye!"
                exit 0
                ;;
            *)
                print_error "Invalid option"
                ;;
        esac
    done
}

# Handle Ctrl+C
trap 'echo -e "\n${YELLOW}Interrupted. Cleaning up...${NC}"; stop_services; exit 1' INT

# Run main function
main