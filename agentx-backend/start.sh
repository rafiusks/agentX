#!/bin/bash

# Start script for AgentX backend

echo "Starting AgentX backend..."

# Check if .env exists
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo "Please edit .env with your configuration"
fi

# Source environment variables
export $(grep -v '^#' .env | xargs)

# Start PostgreSQL if using Docker
if command -v docker &> /dev/null; then
    echo "Starting PostgreSQL with Docker..."
    docker-compose up -d postgres
    
    # Wait for PostgreSQL to be ready
    echo "Waiting for PostgreSQL..."
    sleep 3
fi

# Ensure PostgreSQL is running
if command -v docker &> /dev/null; then
    if ! docker ps | grep -q "postgres"; then
        echo "PostgreSQL is not running. Starting it now..."
        docker-compose up -d postgres
        echo "Waiting for PostgreSQL to be ready..."
        sleep 5
    fi
fi

# Run the full server with database
echo "Starting Go server on port ${AGENTX_PORT:-8080}..."
go run cmd/server/main.go