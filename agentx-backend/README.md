# AgentX Go Backend

A high-performance Go backend for AgentX, providing a unified API for multiple LLM providers.

## Features

- ✅ Multiple LLM provider support (OpenAI, Anthropic, Ollama, LM Studio)
- ✅ Real-time streaming responses via WebSocket
- ✅ Function/tool calling support
- ✅ Dynamic model discovery
- ✅ Session management
- ✅ Configuration management with environment variables
- ✅ CORS support for web clients

## Quick Start

```bash
# Clone and setup
cd agentx-backend
make setup

# Copy and edit config
cp config.example.json config.json
# Edit config.json with your API keys

# Run with hot reloading
make dev

# Or run without hot reloading
make run
```

## Getting Started

### Prerequisites

- Go 1.22 or higher
- PostgreSQL 14 or higher
- LLM provider API keys (for cloud providers)

### Installation

```bash
# Clone the repository
cd agentx-backend

# Download dependencies
go mod download

# Build the application
go build -o agentx-server cmd/server/main.go

# (Optional) Install Air for hot reloading during development
go install github.com/cosmtrek/air@latest
```

### Database Setup

```bash
# Create PostgreSQL database
createdb agentx

# Or using psql
psql -U postgres -c "CREATE DATABASE agentx;"

# Create user (optional)
psql -U postgres -c "CREATE USER agentx WITH PASSWORD 'your-password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE agentx TO agentx;"
```

### Configuration

Create a `config.json` file or use environment variables:

```json
{
  "server": {
    "host": "localhost",
    "port": 3000
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "agentx",
    "password": "your-password",
    "database": "agentx",
    "sslmode": "disable"
  },
  "providers": {
    "openai": {
      "type": "openai",
      "name": "OpenAI",
      "api_key": "your-api-key",
      "models": ["gpt-4", "gpt-3.5-turbo"],
      "default_model": "gpt-3.5-turbo"
    },
    "anthropic": {
      "type": "anthropic",
      "name": "Anthropic",
      "api_key": "your-api-key",
      "models": ["claude-3-opus-20240229", "claude-3-sonnet-20240229"],
      "default_model": "claude-3-sonnet-20240229"
    },
    "ollama": {
      "type": "openai-compatible",
      "name": "Ollama",
      "base_url": "http://localhost:11434",
      "models": [],
      "default_model": ""
    }
  },
  "default_provider": "openai"
}
```

Environment variables:
- `AGENTX_PORT`: Server port (default: 3000)
- `AGENTX_HOST`: Server host (default: localhost)
- `AGENTX_CORS_ORIGINS`: Allowed CORS origins
- `POSTGRES_HOST`: PostgreSQL host
- `POSTGRES_PORT`: PostgreSQL port
- `POSTGRES_USER`: PostgreSQL user
- `POSTGRES_PASSWORD`: PostgreSQL password
- `POSTGRES_DB`: PostgreSQL database name
- `OPENAI_API_KEY`: OpenAI API key
- `ANTHROPIC_API_KEY`: Anthropic API key

### Running the Server

```bash
# Run with default configuration
./agentx-server

# Or run directly with Go
go run cmd/server/main.go

# With environment variables
AGENTX_PORT=8080 OPENAI_API_KEY=sk-... ./agentx-server

# Development mode with hot reloading (requires Air)
air

# Or specify a custom config
air -c .air.toml
```

## API Documentation

### REST Endpoints

#### Providers
- `GET /api/v1/providers` - List all providers
- `PUT /api/v1/providers/:id/config` - Update provider configuration
- `POST /api/v1/providers/:id/discover` - Discover available models

#### Chat Sessions
- `POST /api/v1/chat/sessions` - Create new session
- `GET /api/v1/chat/sessions` - List all sessions
- `GET /api/v1/chat/sessions/:id` - Get session details
- `DELETE /api/v1/chat/sessions/:id` - Delete session
- `POST /api/v1/chat/sessions/:id/messages` - Send message (non-streaming)

#### Settings
- `GET /api/v1/settings` - Get application settings
- `PUT /api/v1/settings` - Update settings

#### Health
- `GET /api/v1/health` - Health check

### WebSocket Endpoints

#### Streaming Chat
- `WS /ws/chat` - Real-time streaming chat

Example WebSocket usage:
```javascript
const ws = new WebSocket('ws://localhost:3000/ws/chat');

ws.onopen = () => {
  ws.send(JSON.stringify({
    session_id: "uuid",
    message: "Hello!",
    provider_id: "openai",
    model: "gpt-3.5-turbo"
  }));
};

ws.onmessage = (event) => {
  const chunk = JSON.parse(event.data);
  if (chunk.delta) {
    console.log('Received:', chunk.delta);
  } else if (chunk.finish_reason) {
    console.log('Complete!');
  }
};
```

## Development

### Project Structure

```
agentx-backend/
├── cmd/server/          # Application entry point
├── internal/            # Internal packages
│   ├── api/            # HTTP/WebSocket handlers
│   ├── config/         # Configuration management
│   ├── models/         # Data models
│   ├── providers/      # LLM provider implementations
│   └── services/       # Business logic
└── pkg/                # Public packages
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/providers/...
```

### Building for Production

```bash
# Build for current platform
go build -ldflags="-s -w" -o agentx-server cmd/server/main.go

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o agentx-server-linux
GOOS=darwin GOARCH=amd64 go build -o agentx-server-darwin
GOOS=windows GOARCH=amd64 go build -o agentx-server.exe
```

## Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o agentx-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agentx-server .
EXPOSE 3000
CMD ["./agentx-server"]
```

Build and run:
```bash
docker build -t agentx-backend .
docker run -p 3000:3000 -e OPENAI_API_KEY=sk-... agentx-backend
```

## License

[Your License Here]