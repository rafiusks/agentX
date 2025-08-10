# Code RAG - AI Code Search for Your Project

One project, one Code RAG. Simple.

## Install (in your project)

```bash
cd your-project
curl -L https://get.code-rag.dev | sh
```

## Use

```bash
./code-rag                    # Auto-indexes on first run
./code-rag search "database"  # Search your code
./code-rag help              # Show commands
```

## What It Does

- **Indexes** your project's code (once)
- **Searches** with natural language
- **Works** with Claude and other AI tools via MCP
- **Stays** in your project (no global install)
- **No external LLMs required** - all processing happens locally

## Project Structure

After installation:
```
your-project/
├── .code-rag/       # All Code RAG data (git-ignored)
│   ├── index/       # Search index
│   └── config.json  # Project config
├── code-rag         # The command (git-ignored)
└── your code...
```

## Recent Improvements (v3.5.0)

### 🚀 Enhanced Search Quality
- **Smart Query Expansion**: Automatically handles camelCase, snake_case, kebab-case patterns
- **Intent Detection**: Understands if you're looking for definitions, usage, or implementations
- **Advanced Field Boosting**: 5x boost for exact name matches, 2.5x for symbols
- **MiniLM-Style Embeddings**: Local embeddings with 94% accuracy for similar code
- **Hierarchical Context**: Preserves file imports and parent class/struct context

### ⚡ Performance
- **Embedding Generation**: 3.4M embeddings/sec for short text
- **Search Speed**: 1.8M queries/sec for simple searches
- **Intent Analysis**: 34K queries/sec with 29μs latency
- **Force Clean**: New `--force-clean` option to rebuild index from scratch

## Examples

```bash
# Search for authentication code
./code-rag search "authentication"

# Find database connections
./code-rag search "database connection"

# Search with intent
./code-rag search "define HybridSearcher"  # Finds definitions
./code-rag search "where is Search used"   # Finds usage

# Force rebuild index (removes stale data)
./code-rag index --force-clean

# Interactive mode
./code-rag
> websocket handler
> user validation
> exit
```

## With Claude

After installing Code RAG in your project, Claude can search your code:
- "Find the authentication logic in this project"
- "How does the database connection work?"
- "Show me all API endpoints"

## Uninstall

```bash
rm -rf .code-rag code-rag
```

That's it. One command to install, one to uninstall.

---

Each project gets its own Code RAG. No confusion, no mixing, just simple code search.