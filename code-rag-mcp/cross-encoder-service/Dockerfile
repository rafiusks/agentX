FROM python:3.10-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Model will be downloaded on first use
# RUN python -c "from sentence_transformers import CrossEncoder; CrossEncoder('cross-encoder/ms-marco-MiniLM-L-6-v2')"

# Copy application
COPY cross_encoder_service.py .

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV CROSS_ENCODER_MODEL=cross-encoder/ms-marco-MiniLM-L-6-v2
ENV CROSS_ENCODER_PORT=8002

EXPOSE 8002

CMD ["python", "cross_encoder_service.py"]