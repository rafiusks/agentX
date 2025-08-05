#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting AgentX with Docker Compose${NC}"

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

# Check if backend config exists
if [ ! -f agentx-backend/config.json ]; then
    echo -e "${YELLOW}Creating backend config.json from template...${NC}"
    cp agentx-backend/config.example.json agentx-backend/config.json
fi

# Build and start services
echo -e "${GREEN}Building and starting services...${NC}"
docker-compose -f docker-compose.full.yml up --build -d

# Wait for services to be ready
echo -e "${GREEN}Waiting for services to be ready...${NC}"

# Wait for PostgreSQL
for i in {1..30}; do
    if docker-compose -f docker-compose.full.yml exec -T postgres pg_isready -U agentx > /dev/null 2>&1; then
        echo -e "${GREEN}PostgreSQL is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Wait for backend
for i in {1..30}; do
    if curl -s http://localhost:3000/api/v1/health > /dev/null 2>&1; then
        echo -e "${GREEN}Backend is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Show logs
echo -e "\n${GREEN}AgentX is running!${NC}"
echo -e "Frontend: ${GREEN}http://localhost:5173${NC}"
echo -e "Backend API: ${GREEN}http://localhost:3000/api/v1${NC}"
echo -e "PostgreSQL: ${GREEN}localhost:5432${NC}"
echo -e "\nTo view logs: ${YELLOW}docker-compose -f docker-compose.full.yml logs -f${NC}"
echo -e "To stop: ${YELLOW}docker-compose -f docker-compose.full.yml down${NC}"