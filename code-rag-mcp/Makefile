.PHONY: help build start stop clean index test

help:
	@echo "🔍 Code RAG MCP - Semantic Code Search with CodeBERT"
	@echo ""
	@echo "Quick Start:"
	@echo "  make start    - Start services in background (do this once)"
	@echo "  make index    - Index your code with CodeBERT (~30s)"
	@echo "  make test     - Test search to verify it works"
	@echo ""
	@echo "Daily Use:"
	@echo "  ./.code-rag/code-rag search \"your query\"   - Search code"
	@echo "  ./.code-rag/code-rag list                  - Show indexed files"
	@echo "  make status                                 - Check services"
	@echo ""
	@echo "Management:"
	@echo "  make stop     - Stop background services"
	@echo "  make clean    - Remove everything and start fresh"
	@echo "  make build    - Rebuild from source"
	@echo ""
	@echo "📚 Full guide: cat QUICKSTART.md"

build:
	@echo "Building Go binary..."
	go build -o .code-rag/code-rag cmd/code-rag/main.go
	@echo "Building Docker services..."
	docker-compose build

start:
	@echo "Starting services..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo ""
	@echo "✅ Services are running in the background!"
	@echo ""
	@echo "  📊 Qdrant Dashboard: http://localhost:6333/dashboard"
	@echo "  🤖 Embedding API: http://localhost:8001/docs"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Re-index with CodeBERT: make index"
	@echo "  2. Test search: make test"
	@echo "  3. Check status: make status"
	@echo "  4. Stop services: make stop"
	@echo ""

stop:
	@echo "Stopping services..."
	docker-compose down

clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -rf .code-rag/code-rag

index:
	@echo "Re-indexing codebase with CodeBERT..."
	EMBEDDING_SERVICE_URL=http://localhost:8001 .code-rag/code-rag index

test:
	@echo "Testing search..."
	.code-rag/code-rag search "websocket handler"

status:
	@echo "Service Status:"
	@curl -s http://localhost:8001/health | jq . || echo "Embedding service not running"
	@curl -s http://localhost:6333/collections | jq . || echo "Qdrant not running"