# PostgreSQL MCP Server Setup for AgentX

## Overview

The PostgreSQL MCP (Model Context Protocol) server provides AI assistants with read-only access to the AgentX database. This enables Claude and other AI models to help with database queries, schema analysis, and data insights while maintaining security through read-only access.

## Installation

### Method 1: Using npm (Recommended for local development)

```bash
# Install globally
npm install -g mcp-postgres-server

# Or use the deprecated but functional official package
npm install -g @modelcontextprotocol/server-postgres
```

### Method 2: Using the provided script

```bash
# Run the MCP server
./start-mcp-postgres.sh
```

### Method 3: Using Docker Compose

```bash
# Start all services including MCP server
docker-compose -f docker-compose.full.yml up -d
```

## Configuration

The MCP server is configured in `mcp-config.json`:

```json
{
  "mcpServers": {
    "postgres-agentx": {
      "command": "npx",
      "args": ["-y", "mcp-postgres-server"],
      "env": {
        "DATABASE_URL": "postgresql://agentx:agentx@localhost:5432/agentx"
      }
    }
  }
}
```

## Database Connection Details

- **Host**: localhost (or `postgres` within Docker network)
- **Port**: 5432
- **Database**: agentx
- **Username**: agentx
- **Password**: agentx
- **Connection String**: `postgresql://agentx:agentx@localhost:5432/agentx`

## Available Capabilities

Once connected, the MCP server provides:

### 1. Schema Introspection
- List all tables in the database
- View table structures and column definitions
- Identify primary keys, foreign keys, and indexes
- Understand relationships between tables

### 2. Read-Only Queries
- Execute SELECT statements
- Perform JOINs across tables
- Use aggregate functions (COUNT, SUM, AVG, etc.)
- Filter and sort data

### 3. Data Analysis
- Generate statistics about data
- Find patterns and anomalies
- Create summary reports
- Analyze data distribution

## Database Schema

The AgentX database includes the following main tables:

### Core Tables
- **sessions**: Chat sessions with AI models
- **messages**: Individual messages within sessions
- **users**: User accounts
- **api_keys**: API key management

### Configuration Tables
- **provider_configs**: AI provider configurations
- **provider_connections**: User-specific provider settings
- **configs**: Application configurations

### Activity Tables
- **user_sessions**: User authentication sessions
- **audit_logs**: System activity logs

## Example Queries

### Get recent chat sessions
```sql
SELECT id, title, provider, model, created_at 
FROM sessions 
ORDER BY created_at DESC 
LIMIT 10;
```

### Count messages by role
```sql
SELECT role, COUNT(*) as message_count 
FROM messages 
GROUP BY role;
```

### Find active users
```sql
SELECT u.email, COUNT(s.id) as session_count 
FROM users u 
LEFT JOIN sessions s ON s.metadata->>'user_id' = u.id::text 
GROUP BY u.email 
ORDER BY session_count DESC;
```

### Analyze provider usage
```sql
SELECT provider, model, COUNT(*) as usage_count 
FROM sessions 
WHERE provider IS NOT NULL 
GROUP BY provider, model 
ORDER BY usage_count DESC;
```

## Security Features

1. **Read-Only Access**: The MCP server only allows SELECT queries
2. **Transaction Isolation**: All queries run in read-only transactions
3. **No DDL Operations**: Cannot modify schema (CREATE, ALTER, DROP)
4. **No DML Operations**: Cannot modify data (INSERT, UPDATE, DELETE)
5. **Connection Security**: Uses standard PostgreSQL authentication

## Troubleshooting

### Connection Issues
1. Ensure PostgreSQL is running: `docker ps | grep postgres`
2. Verify connection details in the connection string
3. Check network connectivity to port 5432

### Permission Issues
- The MCP server requires SELECT permissions on all tables
- Default `agentx` user has appropriate permissions

### Query Limitations
- Complex queries may timeout after 30 seconds
- Result sets are limited to prevent memory issues
- Some PostgreSQL extensions may not be available

## Integration with AI Assistants

### For Claude Desktop App
Add to your Claude desktop configuration:

```json
{
  "mcpServers": {
    "agentx-db": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-postgres",
        "postgresql://agentx:agentx@localhost:5432/agentx"
      ]
    }
  }
}
```

### For AgentX Application
The MCP server can be integrated with the AgentX application to provide database insights directly within the chat interface.

## Best Practices

1. **Use Specific Queries**: Be specific about what data you need
2. **Limit Result Sets**: Use LIMIT clauses for large tables
3. **Index Usage**: Leverage existing indexes for better performance
4. **Avoid Cartesian Products**: Be careful with JOINs
5. **Monitor Performance**: Check query execution times

## Support

For issues or questions about the PostgreSQL MCP server:
1. Check the logs: `docker logs agentx-mcp-postgres`
2. Verify database connectivity
3. Review the MCP server documentation
4. Contact the AgentX team for support