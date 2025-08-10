"""
CodeBERT/CodeT5 Embedding Service
Provides embeddings for code using Microsoft's CodeBERT or Salesforce's CodeT5
"""

import os
import logging
from typing import List, Optional
from contextlib import asynccontextmanager

import torch
import numpy as np
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from transformers import AutoTokenizer, AutoModel

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global model and tokenizer
model = None
tokenizer = None
device = None

class EmbeddingRequest(BaseModel):
    texts: List[str]
    model: Optional[str] = "microsoft/codebert-base"

class EmbeddingResponse(BaseModel):
    embeddings: List[List[float]]
    model: str
    dimension: int

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup, cleanup on shutdown"""
    global model, tokenizer, device
    
    model_name = os.getenv("EMBEDDING_MODEL", "microsoft/codebert-base")
    logger.info(f"Loading model: {model_name}")
    
    # Detect device
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    logger.info(f"Using device: {device}")
    
    # Load model and tokenizer
    try:
        tokenizer = AutoTokenizer.from_pretrained(model_name)
        model = AutoModel.from_pretrained(model_name)
        model.to(device)
        model.eval()
        logger.info(f"Model loaded successfully: {model_name}")
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        # Fall back to a smaller model if the main one fails
        model_name = "microsoft/codebert-base"
        tokenizer = AutoTokenizer.from_pretrained(model_name)
        model = AutoModel.from_pretrained(model_name)
        model.to(device)
        model.eval()
    
    yield
    
    # Cleanup
    if model:
        del model
    if tokenizer:
        del tokenizer
    torch.cuda.empty_cache()

app = FastAPI(
    title="Code Embedding Service",
    description="Provides code embeddings using CodeBERT/CodeT5",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "device": str(device) if device else "not initialized"
    }

@app.post("/embed", response_model=EmbeddingResponse)
async def create_embeddings(request: EmbeddingRequest):
    """Generate embeddings for given texts"""
    if not model or not tokenizer:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    try:
        embeddings = []
        
        # Process in batches for efficiency
        batch_size = 32
        for i in range(0, len(request.texts), batch_size):
            batch_texts = request.texts[i:i + batch_size]
            
            # Tokenize
            inputs = tokenizer(
                batch_texts,
                padding=True,
                truncation=True,
                max_length=512,
                return_tensors="pt"
            )
            
            # Move to device
            inputs = {key: val.to(device) for key, val in inputs.items()}
            
            # Generate embeddings
            with torch.no_grad():
                outputs = model(**inputs)
                
                # Use pooled output if available, otherwise mean pooling
                if hasattr(outputs, 'pooler_output'):
                    batch_embeddings = outputs.pooler_output
                else:
                    # Mean pooling over token embeddings
                    token_embeddings = outputs.last_hidden_state
                    attention_mask = inputs['attention_mask']
                    mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
                    sum_embeddings = torch.sum(token_embeddings * mask_expanded, 1)
                    sum_mask = torch.clamp(mask_expanded.sum(1), min=1e-9)
                    batch_embeddings = sum_embeddings / sum_mask
                
                # Convert to numpy and then to list
                batch_embeddings = batch_embeddings.cpu().numpy().tolist()
                embeddings.extend(batch_embeddings)
        
        # Get embedding dimension
        dimension = len(embeddings[0]) if embeddings else 768
        
        return EmbeddingResponse(
            embeddings=embeddings,
            model=request.model or "microsoft/codebert-base",
            dimension=dimension
        )
    
    except Exception as e:
        logger.error(f"Error generating embeddings: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/embed_single")
async def create_single_embedding(text: str, model_name: Optional[str] = None):
    """Generate embedding for a single text"""
    request = EmbeddingRequest(texts=[text], model=model_name)
    response = await create_embeddings(request)
    return {
        "embedding": response.embeddings[0],
        "model": response.model,
        "dimension": response.dimension
    }

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("EMBEDDING_SERVICE_PORT", "8001"))
    uvicorn.run(app, host="0.0.0.0", port=port)