#!/bin/bash

# Start PostgreSQL MCP Server for AgentX
# This provides read-only access to the PostgreSQL database for AI assistants

echo "Starting PostgreSQL MCP Server..."
echo "Database: agentx@localhost:5432"
echo "Access: Read-only for safety"
echo "-----------------------------------"

# Export the database URL
export DATABASE_URL="postgresql://agentx:agentx@localhost:5432/agentx"

# Try the newer mcp-postgres-server first
echo "Attempting to start mcp-postgres-server..."
npx -y mcp-postgres-server

# If that fails, fall back to the deprecated but functional version
if [ $? -ne 0 ]; then
    echo "Falling back to @modelcontextprotocol/server-postgres..."
    npx -y @modelcontextprotocol/server-postgres "postgresql://agentx:agentx@localhost:5432/agentx"
fi