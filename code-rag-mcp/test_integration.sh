#!/bin/bash

echo "=== Code RAG Integration Test ==="
echo ""

# Check if Qdrant is running
echo "1. Checking Qdrant..."
if curl -s http://localhost:6333/collections > /dev/null 2>&1; then
    echo "   ✅ Qdrant is running"
else
    echo "   ❌ Qdrant is not running"
    echo "   Starting Qdrant..."
    docker start qdrant || docker run -d --name qdrant -p 6333:6333 qdrant/qdrant
    sleep 3
fi

# Check if embedding service would work (without actually running it)
echo ""
echo "2. Checking Python environment..."
if python3 -c "import transformers" 2>/dev/null; then
    echo "   ✅ Python transformers library is available"
else
    echo "   ⚠️  Python transformers not installed locally"
    echo "   The service will use fallback embeddings"
fi

# Test with fallback embeddings
echo ""
echo "3. Testing with current setup (fallback embeddings)..."
export EMBEDDING_SERVICE_URL=""  # Force fallback
./.code-rag/code-rag search "authentication handler" | head -10

echo ""
echo "=== Test Complete ==="
echo ""
echo "To use real CodeBERT embeddings:"
echo "1. Build the Docker image: docker-compose build embedding-service"
echo "2. Start the service: docker-compose up -d embedding-service"
echo "3. Re-index: EMBEDDING_SERVICE_URL=http://localhost:8001 ./code-rag/code-rag index"
echo "4. Search: ./code-rag/code-rag search 'your query'"