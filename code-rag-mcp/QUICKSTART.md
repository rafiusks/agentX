# Code RAG MCP - Quick Start Guide

## What This Is
Code RAG MCP is a semantic code search tool that uses CodeBERT to understand your code's meaning, not just keywords.

## NEW: Integrated Mode (v3.4+)
**The CLI now automatically manages Docker services!** When you start the CLI, it starts the services. When you exit, it stops them.

## Simple Usage

### Just Run It!
```bash
./.code-rag/code-rag
```
This will:
1. ✅ Automatically start Docker services (Qdrant + CodeBERT)
2. ✅ Show you an interactive prompt
3. ✅ Stop services when you exit

### First Time Setup
On first run, it will:
1. Ask what project to index
2. Start the services
3. Index your code (~30 seconds)
4. Ready to search!

### 2. Index Your Code (First Time or After Changes)
```bash
make index
```
This analyzes all your code files and stores their semantic embeddings. Takes ~30 seconds for a medium project.

### 3. Search Your Code
```bash
# Direct search
./.code-rag/code-rag search "websocket handler"

# Interactive mode
./.code-rag/code-rag
> authentication middleware
> exit
```

### 4. Check What's Running
```bash
make status
```
Shows if services are healthy.

### 5. Stop Services (When Done)
```bash
make stop
```
Stops the Docker containers but preserves the index.

## Common Workflows

### Daily Development
```bash
# Morning: Start services once
make start

# Throughout the day: Just search
./.code-rag/code-rag search "whatever you need"

# End of day: Stop services
make stop
```

### After Code Changes
```bash
# Re-index to include new code
make index
```

### See What Files Are Indexed
```bash
./.code-rag/code-rag list
```

## Understanding the Output

When you run `make start`, you'll see:
```
✅ Services are running in the background!
```
Then you return to your shell prompt. **This is normal!** The services are running in Docker.

## Troubleshooting

### "Port already allocated" Error
```bash
# Stop any existing containers
docker stop code-rag-qdrant code-rag-embedding
docker rm code-rag-qdrant code-rag-embedding
# Try again
make start
```

### Services Not Responding
```bash
# Check Docker logs
docker logs code-rag-embedding
docker logs code-rag-qdrant
```

### Reset Everything
```bash
make clean
make build
make start
make index
```

## How It Works

1. **You type**: `search "error handling"`
2. **CodeBERT converts** your query to a 768-dimensional vector representing its meaning
3. **Qdrant finds** code with similar meaning vectors
4. **You get** semantically relevant code, not just text matches

## Advanced Features

### View All Indexed Files
```bash
./.code-rag/code-rag list -v
```

### Use Without Docker (Fallback Mode)
```bash
# Just Qdrant is enough for basic functionality
docker run -d --name qdrant -p 6333:6333 qdrant/qdrant
./.code-rag/code-rag index
./.code-rag/code-rag search "your query"
```
This uses simpler embeddings but still works!

### Access Web Interfaces
- Qdrant Dashboard: http://localhost:6333/dashboard
- Embedding API Docs: http://localhost:8001/docs