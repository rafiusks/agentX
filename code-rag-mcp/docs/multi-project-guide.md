# Multi-Project Setup Guide

## The Challenge
When working with multiple projects, you don't want code from different projects mixing in search results. Here are several strategies to handle this.

## Solution 1: Project-Specific Collections (Recommended)

Each project gets its own collection in the vector database:

```bash
# Index project A
PROJECT_NAME=projectA ./quickstart.sh

# Index project B  
PROJECT_NAME=projectB ./quickstart.sh
```

### How it works:
- Each project's code goes into a separate collection
- Collections named like: `code_rag_projectA`, `code_rag_projectB`
- Search is isolated per project by default
- Optional cross-project search when needed

### Implementation:
```json
// Tool calls include project context
{
  "name": "code_search",
  "arguments": {
    "query": "websocket handler",
    "project": "projectA",  // Search only in projectA
    "language": "go"
  }
}
```

## Solution 2: Multiple MCP Server Instances

Run a separate MCP server for each project:

```bash
# Start server for project A (port 3333)
./start-project.sh projectA /path/to/projectA 3333

# Start server for project B (port 3334)
./start-project.sh projectB /path/to/projectB 3334
```

### Claude Code Configuration:
```json
{
  "mcpServers": {
    "code-rag-projectA": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-p", "3333:3333", "code-rag-mcp:latest"]
    },
    "code-rag-projectB": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-p", "3334:3334", "code-rag-mcp:latest"]
    }
  }
}
```

### Pros:
- Complete isolation between projects
- Can run different configurations per project
- Easy to start/stop individual projects

### Cons:
- More resource usage (multiple servers)
- Need to manage multiple ports

## Solution 3: Workspace Mode

Single server with workspace awareness:

```bash
# Start in workspace mode
WORKSPACE_MODE=true ./quickstart.sh
```

### Directory Structure:
```
~/Code/
├── projectA/
│   └── .code-rag/      # Project-specific index
├── projectB/
│   └── .code-rag/      # Project-specific index
└── projectC/
    └── .code-rag/      # Project-specific index
```

### How it works:
1. Server detects current working directory
2. Automatically switches context based on where you're working
3. Can search across all projects or current project only

### Usage in Claude Code:
```bash
# Claude Code automatically uses current project context
cd ~/Code/projectA
# Now searches only projectA

cd ~/Code/projectB  
# Now searches only projectB
```

## Solution 4: Metadata Filtering

Single collection with metadata-based filtering:

```json
// Each chunk has metadata
{
  "code": "function handleWebSocket() {...}",
  "metadata": {
    "project": "projectA",
    "repository": "github.com/user/projectA",
    "path": "/src/websocket.go",
    "language": "go"
  }
}
```

### Search with filters:
```json
{
  "name": "code_search",
  "arguments": {
    "query": "websocket",
    "filters": {
      "project": "projectA",
      "language": "go"
    }
  }
}
```

## Recommended Approach

For most users, **Solution 1 (Project-Specific Collections)** is best because:
- ✅ Simple to understand and use
- ✅ Good isolation between projects
- ✅ Efficient resource usage
- ✅ Easy to manage

### Quick Setup:
```bash
# Index multiple projects
for project in projectA projectB projectC; do
  PROJECT_NAME=$project ./quickstart.sh index /path/to/$project
done

# Search specific project
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"code_search","arguments":{"query":"test","project":"projectA"}}}' | docker run -i --rm --network host code-rag-mcp:latest
```

## Migration Between Projects

To switch between projects without re-indexing:

```bash
# Save current project state
docker exec code-rag-mcp backup projectA

# Switch to different project
docker exec code-rag-mcp switch projectB

# List all indexed projects
docker exec code-rag-mcp list-projects
```

## Best Practices

1. **Name projects clearly**: Use repository names or org/repo format
2. **Regular re-indexing**: Set up cron jobs for active projects
3. **Archive old projects**: Remove collections for archived projects
4. **Document project mappings**: Keep a README of which projects are indexed
5. **Use consistent naming**: Helps with cross-project operations

## Advanced: Cross-Project Search

Sometimes you want to search across all your projects:

```json
{
  "name": "code_search",
  "arguments": {
    "query": "authentication middleware",
    "project": "*",  // Search all projects
    "show_project": true  // Include project name in results
  }
}
```

This is useful for:
- Finding similar implementations across projects
- Discovering reusable code
- Checking for consistency across microservices
- Learning from past solutions