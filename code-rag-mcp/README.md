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
- **Works** with Claude and other AI tools
- **Stays** in your project (no global install)

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

## Examples

```bash
# Search for authentication code
./code-rag search "authentication"

# Find database connections
./code-rag search "database connection"

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