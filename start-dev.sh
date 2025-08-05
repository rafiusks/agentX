#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting AgentX Development Environment${NC}"

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
cd agentx-backend
docker-compose -f docker-compose.dev.yml up -d

# Wait for PostgreSQL to be ready
echo -e "${GREEN}Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
    if docker-compose -f docker-compose.dev.yml exec -T postgres pg_isready -U agentx > /dev/null 2>&1; then
        echo -e "${GREEN}PostgreSQL is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Check if config.json exists
if [ ! -f config.json ]; then
    echo -e "${YELLOW}Creating config.json from template...${NC}"
    cp config.example.json config.json
fi

# Start backend with Air
echo -e "${GREEN}Starting Go backend with hot reloading...${NC}"
# Use full path to air since it might not be in PATH yet
$HOME/go/bin/air &
BACKEND_PID=$!

cd ..

# Wait for backend to be ready
echo -e "${GREEN}Waiting for backend to be ready...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:3000/api/v1/health > /dev/null 2>&1; then
        echo -e "${GREEN}Backend is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Start frontend
echo -e "${GREEN}Starting React frontend...${NC}"
npm run dev &
FRONTEND_PID=$!

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Shutting down...${NC}"
    kill $BACKEND_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    cd agentx-backend
    docker-compose -f docker-compose.dev.yml down
    cd ..
    echo -e "${GREEN}Shutdown complete${NC}"
}

# Set up trap to cleanup on script exit
trap cleanup EXIT

echo -e "\n${GREEN}AgentX is running!${NC}"
echo -e "Frontend: ${GREEN}http://localhost:5173${NC}"
echo -e "Backend API: ${GREEN}http://localhost:3000/api/v1${NC}"
echo -e "PostgreSQL: ${GREEN}localhost:5432${NC}"
echo -e "\nPress ${YELLOW}Ctrl+C${NC} to stop all services"

# Wait for background processes
wait