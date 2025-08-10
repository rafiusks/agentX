#!/usr/bin/env python3
"""
Cross-Encoder Reranking Service
Provides high-precision reranking of search results using cross-encoder models
"""

import os
import time
from typing import List, Dict, Any
from contextlib import asynccontextmanager
import logging
import hashlib
from collections import OrderedDict

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
import uvicorn
from sentence_transformers import CrossEncoder
import torch
import numpy as np

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global model variable
cross_encoder = None
cache = None

class LRUCache:
    """Simple LRU cache for reranking results"""
    def __init__(self, capacity: int = 1000):
        self.cache = OrderedDict()
        self.capacity = capacity
    
    def get(self, key: str):
        if key in self.cache:
            # Move to end (most recently used)
            self.cache.move_to_end(key)
            return self.cache[key]
        return None
    
    def put(self, key: str, value: Any):
        if key in self.cache:
            # Update and move to end
            self.cache[key] = value
            self.cache.move_to_end(key)
        else:
            self.cache[key] = value
            if len(self.cache) > self.capacity:
                # Remove least recently used
                self.cache.popitem(last=False)
    
    def clear(self):
        self.cache.clear()

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage model lifecycle"""
    global cross_encoder, cache
    
    # Load model on startup
    logger.info("Loading cross-encoder model...")
    model_name = os.getenv("CROSS_ENCODER_MODEL", "cross-encoder/ms-marco-MiniLM-L-6-v2")
    
    try:
        cross_encoder = CrossEncoder(model_name, max_length=512)
        cache = LRUCache(capacity=1000)
        logger.info(f"Model {model_name} loaded successfully")
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        raise
    
    yield
    
    # Cleanup on shutdown
    logger.info("Shutting down cross-encoder service")

app = FastAPI(title="Cross-Encoder Reranking Service", lifespan=lifespan)

class CodeCandidate(BaseModel):
    """A code search result candidate for reranking"""
    code: str = Field(..., description="The code snippet")
    file_path: str = Field(..., description="File path")
    score: float = Field(..., description="Original search score")
    language: str = Field(default="", description="Programming language")
    type: str = Field(default="", description="Code type (function, class, etc)")
    name: str = Field(default="", description="Function/class name")

class RerankRequest(BaseModel):
    """Request for reranking code search results"""
    query: str = Field(..., description="The search query")
    candidates: List[CodeCandidate] = Field(..., description="List of candidates to rerank")
    top_k: int = Field(default=10, description="Number of top results to return")
    use_cache: bool = Field(default=True, description="Whether to use caching")

class RerankResponse(BaseModel):
    """Response with reranked results"""
    results: List[Dict[str, Any]]
    rerank_time_ms: float
    cache_hit: bool = False

def generate_cache_key(query: str, candidates: List[CodeCandidate]) -> str:
    """Generate a cache key for the query and candidates"""
    # Create a hash of query + candidate file paths
    content = query + "|".join([c.file_path for c in candidates[:20]])  # Use first 20 for key
    return hashlib.md5(content.encode()).hexdigest()

def prepare_pairs(query: str, candidates: List[CodeCandidate]) -> List[List[str]]:
    """Prepare query-document pairs for the cross-encoder"""
    pairs = []
    for candidate in candidates:
        # Create context-rich input for better ranking
        # Include metadata in the document for better understanding
        doc_context = f"{candidate.type}: {candidate.name}\n" if candidate.type else ""
        doc_context += f"Language: {candidate.language}\n" if candidate.language else ""
        doc_context += f"File: {candidate.file_path}\n"
        doc_context += f"\n{candidate.code[:1500]}"  # Truncate very long code
        
        pairs.append([query, doc_context])
    
    return pairs

@app.post("/rerank", response_model=RerankResponse)
async def rerank(request: RerankRequest):
    """Rerank code search results using cross-encoder"""
    global cross_encoder, cache
    
    if cross_encoder is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    start_time = time.time()
    cache_hit = False
    
    # Check cache if enabled
    if request.use_cache and cache is not None:
        cache_key = generate_cache_key(request.query, request.candidates)
        cached_result = cache.get(cache_key)
        if cached_result is not None:
            logger.info(f"Cache hit for query: {request.query[:50]}")
            return RerankResponse(
                results=cached_result,
                rerank_time_ms=(time.time() - start_time) * 1000,
                cache_hit=True
            )
    
    # Prepare input pairs
    pairs = prepare_pairs(request.query, request.candidates)
    
    # Get cross-encoder scores
    try:
        with torch.no_grad():
            scores = cross_encoder.predict(pairs)
    except Exception as e:
        logger.error(f"Cross-encoder prediction failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
    
    # Combine with original scores using weighted combination
    results = []
    for i, candidate in enumerate(request.candidates):
        # Normalize cross-encoder score to 0-1 range
        # Cross-encoder scores are typically logits, so we apply sigmoid
        cross_score = 1 / (1 + np.exp(-scores[i]))
        
        # Weighted combination: 70% cross-encoder, 30% original
        final_score = 0.7 * cross_score + 0.3 * candidate.score
        
        results.append({
            "file_path": candidate.file_path,
            "code": candidate.code,
            "language": candidate.language,
            "type": candidate.type,
            "name": candidate.name,
            "original_score": candidate.score,
            "cross_encoder_score": float(cross_score),
            "final_score": float(final_score)
        })
    
    # Sort by final score
    results.sort(key=lambda x: x["final_score"], reverse=True)
    
    # Take top k
    results = results[:request.top_k]
    
    # Cache the result if enabled
    if request.use_cache and cache is not None:
        cache_key = generate_cache_key(request.query, request.candidates)
        cache.put(cache_key, results)
    
    elapsed_ms = (time.time() - start_time) * 1000
    logger.info(f"Reranked {len(request.candidates)} candidates in {elapsed_ms:.2f}ms")
    
    return RerankResponse(
        results=results,
        rerank_time_ms=elapsed_ms,
        cache_hit=cache_hit
    )

@app.get("/health")
async def health():
    """Health check endpoint"""
    global cross_encoder
    return {
        "status": "healthy" if cross_encoder is not None else "unhealthy",
        "model_loaded": cross_encoder is not None,
        "cache_size": len(cache.cache) if cache else 0
    }

@app.post("/clear_cache")
async def clear_cache():
    """Clear the reranking cache"""
    global cache
    if cache:
        cache.clear()
        return {"message": "Cache cleared"}
    return {"message": "No cache to clear"}

@app.get("/")
async def root():
    """Root endpoint with service info"""
    return {
        "service": "Cross-Encoder Reranking Service",
        "version": "1.0.0",
        "endpoints": [
            "/rerank - POST - Rerank search results",
            "/health - GET - Health check",
            "/clear_cache - POST - Clear cache"
        ]
    }

if __name__ == "__main__":
    port = int(os.getenv("CROSS_ENCODER_PORT", "8002"))
    uvicorn.run(app, host="0.0.0.0", port=port)