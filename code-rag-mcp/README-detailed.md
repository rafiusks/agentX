# Code RAG MCP Server

A standalone MCP (Model Context Protocol) server that provides intelligent code search and analysis using Retrieval-Augmented Generation (RAG). Works with any MCP-compatible client including Claude Code, AgentX, and other AI development tools.

## Features

- 🔍 **Semantic Code Search**: Find code using natural language queries
- 🧠 **Multi-Model Embeddings**: Support for CodeT5, CodeBERT, and OpenAI embeddings
- 📊 **AST-Aware Chunking**: Intelligent code splitting that preserves semantic boundaries
- ⚡ **Hybrid Search**: Combines semantic, keyword, and structural search
- 🔄 **Incremental Indexing**: Git-aware updates for efficient re-indexing
- 🐳 **Easy Deployment**: Docker-based setup with Qdrant vector database
- 🔌 **Universal Compatibility**: Works with any MCP client
- 📁 **Multi-Project Support**: Manage multiple codebases with isolated collections

## Quick Start

### 🚀 One-Command Setup

```bash
# Clone and start everything with one command
git clone https://github.com/yourusername/code-rag-mcp
cd code-rag-mcp
./start.sh  # Quick setup
# OR
./quickstart.sh  # Interactive setup with menu
```

#### Quick Start (`start.sh`)
Simple, fast setup that:
- ✅ Checks for Docker or Go
- ✅ Starts Qdrant vector database
- ✅ Builds the server if needed
- ✅ Shows usage commands

#### Interactive Setup (`quickstart.sh`)
Full-featured setup with:
- ✅ Prerequisites checking
- ✅ Automatic configuration
- ✅ Server testing
- ✅ Sample repository indexing
- ✅ Claude Code config generation
- ✅ Interactive menu for operations

### Manual Setup

#### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/code-rag-mcp
cd code-rag-mcp

# Start the services
docker-compose up -d

# The server is now running and ready for MCP connections
```

#### Using Go

```bash
# Install dependencies
go mod download

# Start Qdrant (required for vector storage)
docker run -p 6333:6333 qdrant/qdrant

# Run the server
go run cmd/server/main.go
```

## Integration

### With Claude Code

Add to your Claude Code configuration:

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "code-rag-mcp:latest"]
    }
  }
}
```

Or for local development:

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "/path/to/code-rag-mcp/code-rag-mcp"
    }
  }
}
```

### With AgentX

In your AgentX backend:

```go
import "github.com/your/mcp-client"

client := mcp.NewClient("localhost:3333")
results, err := client.CallTool("code_search", map[string]any{
    "query": "websocket handler implementation",
    "language": "go",
    "limit": 10,
})
```

### Standalone CLI Usage

```bash
# Index a repository
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_repository","arguments":{"path":"/path/to/repo"}}}' | docker run -i code-rag-mcp

# Search for code
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"code_search","arguments":{"query":"authentication middleware"}}}' | docker run -i code-rag-mcp
```

## Available Tools

### `code_search`
Search for code using semantic understanding.

**Parameters:**
- `query` (string, required): The search query
- `language` (string): Programming language filter (go, javascript, typescript, python, rust, any)
- `limit` (integer): Maximum number of results (default: 10)

**Example:**
```json
{
  "name": "code_search",
  "arguments": {
    "query": "implement websocket connection",
    "language": "go",
    "limit": 5
  }
}
```

### `explain_code`
Get detailed explanation of code with context.

**Parameters:**
- `code` (string, required): Code snippet to explain
- `file_path` (string): Optional file path for additional context

### `find_similar`
Find code similar to the provided example.

**Parameters:**
- `code` (string, required): Example code to find similar patterns
- `threshold` (number): Similarity threshold 0.0-1.0 (default: 0.7)

### `index_repository`
Index a repository for searching.

**Parameters:**
- `path` (string, required): Path to the repository
- `incremental` (boolean): Only index changed files (default: true)

### `get_dependencies`
Analyze code dependencies.

**Parameters:**
- `file_path` (string, required): Path to the file to analyze
- `include_transitive` (boolean): Include transitive dependencies (default: false)

### `suggest_improvements`
Get improvement suggestions for code.

**Parameters:**
- `code` (string, required): Code to analyze
- `focus` (string): Focus area (performance, readability, security, best_practices, all)

## Resources

The server provides these read-only resources:

- `indexed_repositories`: List of repositories that have been indexed
- `search_statistics`: Usage statistics for code search
- `model_capabilities`: Available embedding models and their capabilities

## Configuration

Create a `config.yaml` file:

```yaml
server:
  port: 3333
  host: 0.0.0.0

embeddings:
  provider: codet5  # Options: codet5, codebert, openai, local
  model: Salesforce/codet5p-110m-embedding
  cache_size: 10000
  batch_size: 32
  # api_key: your-api-key  # For OpenAI or HuggingFace

chunking:
  strategy: ast_aware  # Options: ast_aware, sliding_window, semantic
  max_chunk_size: 512
  chunk_overlap: 128
  min_chunk_size: 50
  languages:
    - go
    - javascript
    - typescript
    - python
    - rust

vectordb:
  type: qdrant  # Options: qdrant, chroma, weaviate
  url: http://localhost:6333
  collection_name: code_embeddings
  dimension: 768
  distance: cosine  # Options: cosine, euclidean, dot
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   MCP Client                         │
│        (Claude Code, AgentX, Cursor, etc.)          │
└─────────────────────────────────────────────────────┘
                          │
                    JSON-RPC 2.0
                          │
┌─────────────────────────────────────────────────────┐
│                Code RAG MCP Server                   │
├─────────────────────────────────────────────────────┤
│                  MCP Protocol Handler                │
├─────────────────────────────────────────────────────┤
│                    RAG Engine                        │
├──────────┬──────────┬──────────┬───────────────────┤
│ Embedder │ Chunker  │ Retriever│    Reranker       │
├──────────┴──────────┴──────────┴───────────────────┤
│              Vector Database (Qdrant)                │
└─────────────────────────────────────────────────────┘
```

## Performance

- **Indexing Speed**: ~1000 files/minute
- **Search Latency**: <100ms for 95% of queries
- **Memory Usage**: ~200MB base + cache
- **Supported Scale**: 100k+ files per repository

## Development

### Building from Source

```bash
# Build binary
go build -o code-rag-mcp cmd/server/main.go

# Build Docker image
docker build -t code-rag-mcp:latest .
```

### Running Tests

```bash
go test ./...
```

### Adding New Language Support

1. Implement parser in `internal/parsers/`
2. Add language detection in `internal/rag/chunker.go`
3. Update configuration options

## Multi-Project Setup

Working with multiple projects? See [Multi-Project Guide](docs/multi-project-guide.md) for:
- Project-specific collections
- Running multiple server instances  
- Workspace-aware mode
- Cross-project search

Quick example:
```bash
# Index different projects into separate collections
PROJECT_NAME=frontend ./quickstart.sh index ~/Code/frontend
PROJECT_NAME=backend ./quickstart.sh index ~/Code/backend
PROJECT_NAME=mobile ./quickstart.sh index ~/Code/mobile-app
```

## Troubleshooting

### Server not responding
- Check if Qdrant is running: `curl http://localhost:6333/health`
- Verify server logs: `docker logs code-rag-mcp-server`

### Poor search results
- Ensure repositories are properly indexed
- Try adjusting the similarity threshold
- Check if the correct embedding model is configured

### High memory usage
- Reduce `cache_size` in configuration
- Use incremental indexing for large repositories
- Consider using a cloud vector database

## Roadmap

- [ ] Support for more programming languages (Java, C++, Ruby)
- [ ] Web UI for repository management
- [ ] Real-time indexing with file watchers
- [ ] Integration with LSP for better code understanding
- [ ] Support for code generation and refactoring
- [ ] Multi-repository search
- [ ] Team collaboration features

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- CodeT5 and CodeBERT teams for embedding models
- Qdrant team for the excellent vector database
- MCP specification contributors
- The open-source community

## Support

- Issues: [GitHub Issues](https://github.com/yourusername/code-rag-mcp/issues)
- Discussions: [GitHub Discussions](https://github.com/yourusername/code-rag-mcp/discussions)
- Documentation: [Wiki](https://github.com/yourusername/code-rag-mcp/wiki)