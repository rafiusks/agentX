#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting AgentX Development Environment${NC}"

# Stop any existing services first
echo -e "${YELLOW}Stopping any existing services...${NC}"
pkill -f "air" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
pkill -f "go run cmd/server/main.go" 2>/dev/null || true
# Also kill any process on our ports
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
lsof -ti:1420 | xargs kill -9 2>/dev/null || true
lsof -ti:5173 | xargs kill -9 2>/dev/null || true
lsof -ti:3000 | xargs kill -9 2>/dev/null || true

# Check if .env exists (for non-sensitive config only)
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env file from template...${NC}"
    cp .env.example .env
fi

# Load environment variables (if any)
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

echo -e "${GREEN}AgentX stores API keys securely in the database.${NC}"
echo -e "${GREEN}Configure providers through the web UI after startup.${NC}"

# Start PostgreSQL
echo -e "${GREEN}Starting PostgreSQL...${NC}"
(cd agentx-backend && docker-compose -f docker-compose.dev.yml up -d)

# Start Code RAG services (Qdrant + Embedding service)
echo -e "${GREEN}Starting Code RAG services...${NC}"
(cd code-rag-mcp && docker-compose up -d)

# Wait for Code RAG services to be ready
echo -e "${GREEN}Waiting for Code RAG services to be ready...${NC}"
for i in {1..60}; do
    if curl -s http://localhost:6333/collections > /dev/null 2>&1 && curl -s http://localhost:8001/health > /dev/null 2>&1; then
        echo -e "${GREEN}Code RAG services are ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Wait for PostgreSQL to be ready
echo -e "${GREEN}Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
    if (cd agentx-backend && docker-compose -f docker-compose.dev.yml exec -T postgres pg_isready -U agentx > /dev/null 2>&1); then
        echo -e "${GREEN}PostgreSQL is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Check if config.json exists
if [ ! -f agentx-backend/config.json ]; then
    echo -e "${YELLOW}Creating config.json from template...${NC}"
    cp agentx-backend/config.example.json agentx-backend/config.json
fi

# Start backend with Air
echo -e "${GREEN}Starting Go backend with hot reloading...${NC}"
# Use full path to air since it might not be in PATH yet
(cd agentx-backend && $HOME/go/bin/air) &
BACKEND_PID=$!

# Wait for backend to be ready
echo -e "${GREEN}Waiting for backend to be ready...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        echo -e "${GREEN}Backend is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Start Code RAG MCP server
echo -e "${GREEN}Starting Code RAG MCP HTTP server...${NC}"
(cd code-rag-mcp && ./.code-rag/code-rag mcp-http > /tmp/code-rag-mcp.log 2>&1) &
MCP_PID=$!
echo -e "${GREEN}Code RAG MCP HTTP server started on :9000 (PID: $MCP_PID)${NC}"

# Start frontend
echo -e "${GREEN}Starting React frontend...${NC}"
npm run dev &
FRONTEND_PID=$!

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Shutting down...${NC}"
    kill $BACKEND_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    kill $MCP_PID 2>/dev/null
    (cd agentx-backend && docker-compose -f docker-compose.dev.yml down)
    (cd code-rag-mcp && docker-compose down)
    echo -e "${GREEN}Shutdown complete${NC}"
}

# Set up trap to cleanup on script exit
trap cleanup EXIT

echo -e "\n${GREEN}AgentX is running!${NC}"
echo -e "Frontend: ${GREEN}http://localhost:1420${NC}"
echo -e "Backend API: ${GREEN}http://localhost:8080/api/v1${NC}"
echo -e "PostgreSQL: ${GREEN}localhost:5432${NC}"
echo -e "Code RAG MCP: ${GREEN}http://localhost:9000 (log: /tmp/code-rag-mcp.log)${NC}"
echo -e "Qdrant Vector DB: ${GREEN}http://localhost:6333${NC}"
echo -e "CodeBERT Service: ${GREEN}http://localhost:8001${NC}"
echo -e "\nPress ${YELLOW}Ctrl+C${NC} to stop all services"

# Wait for background processes
wait