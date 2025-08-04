# MCP Integration Example for AgentX

## Overview

We've successfully integrated Model Context Protocol (MCP) with AgentX's function calling system. This creates a powerful bridge between AI models and external tools.

## What We Built

### Backend Architecture:
1. **MCP Module** (`/src-tauri/src/mcp/`)
   - `mod.rs` - Core MCP types and conversions
   - `server.rs` - MCP server communication (JSON-RPC)
   - `manager.rs` - Multi-server management and tool routing

2. **MCP Enhanced Provider** (`/src-tauri/src/providers/mcp_enhanced.rs`)
   - Wraps any LLM provider and adds MCP tool capabilities
   - Auto-discovers tools from connected MCP servers
   - Routes function calls to appropriate MCP servers

3. **Tauri Commands**
   - `add_mcp_server` - Connect to new MCP servers
   - `list_mcp_tools` - Get all available tools
   - `get_mcp_servers` - List connected servers

### Frontend Interface:
1. **MCP Servers Tab** - New tab in the main interface
2. **MCPServers Component** - Manage MCP server connections
3. **Tool Discovery** - Shows available tools from all servers
4. **Example Servers** - Built-in examples for popular MCP servers

## The Flow

```
User → Chat → AI (with MCP tools) → Function Call → MCP Manager → MCP Server → Tool Execution
                                        ↑                                          ↓
                                        ←──────── Result ←─────────────────────────
```

## Example Usage

### 1. Add a File System MCP Server
```typescript
await invoke('add_mcp_server', {
  name: 'filesystem',
  command: 'npx',
  args: ['@modelcontextprotocol/server-filesystem', '/Users/rafael/Code']
})
```

### 2. AI Gets Access to File Operations
The AI can now:
- Read files: `read_file({ path: "src/main.rs" })`
- Write files: `write_file({ path: "output.txt", content: "Hello" })`
- List directories: `list_directory({ path: "src/" })`

### 3. Chat Example
**User**: "Read my README.md file and summarize it"

**AI**: 
```
[Function Call → read_file]
Arguments: { "path": "README.md" }

Based on your README.md file, this project is...
```

## Available MCP Servers

### Core Servers:
- **File System** - Read/write files and directories
- **Git** - Repository operations (status, commit, diff)
- **PostgreSQL** - Database queries and operations
- **Brave Search** - Web search capabilities

### Custom Integration:
The system can connect to any MCP-compatible server, enabling:
- Custom business logic tools
- API integrations
- Database connections
- External service calls

## Key Benefits

1. **Universal Tool Access** - Any MCP server becomes available to the AI
2. **Type Safety** - Full Rust type system for tool definitions
3. **Auto Discovery** - Tools are automatically exposed as AI functions
4. **Error Handling** - Proper error propagation from tools to AI
5. **Multi-Server** - Connect multiple MCP servers simultaneously

## Next Steps

To fully utilize this integration:

1. **Install MCP Servers**:
   ```bash
   npm install -g @modelcontextprotocol/server-filesystem
   npm install -g @modelcontextprotocol/server-git
   ```

2. **Connect Servers** via the MCP tab in AgentX

3. **Chat with Enhanced AI** that can now use real tools!

The infrastructure is complete - AgentX now has the foundation for true agentic AI that can interact with the real world through MCP servers.