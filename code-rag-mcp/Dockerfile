# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./

# Download dependencies
RUN go mod download || go mod tidy

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o code-rag-mcp cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/code-rag-mcp .

# Copy default configs
COPY --from=builder /app/configs ./configs

# Create data directory
RUN mkdir -p /app/data

# Expose port (for documentation, MCP uses stdio)
EXPOSE 3333

# Run the binary
ENTRYPOINT ["./code-rag-mcp"]