# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agentx-server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

WORKDIR /app/

# Copy binary from builder
COPY --from=builder /app/agentx-server .
COPY --from=builder /app/config.example.json .

# Create non-root user
RUN addgroup -g 1000 agentx && \
    adduser -D -u 1000 -G agentx agentx && \
    chown -R agentx:agentx /app

USER agentx

EXPOSE 3000

CMD ["./agentx-server"]