# AgentX Full Stack Setup

This guide covers running the complete AgentX stack including the React frontend, Go backend, and PostgreSQL database.

## Quick Start

### Option 1: Development Mode (Recommended)

```bash
# Initial setup
make setup

# Edit .env file with your API keys
nano .env

# Start everything
make dev
```

This will start:
- PostgreSQL on port 5432
- Go backend with hot reloading on port 3000
- React frontend with hot reloading on port 5173

### Option 2: Docker Compose

```bash
# Setup and start
make setup
make docker
```

This builds and runs everything in Docker containers.

## Services

### Frontend (React + Vite)
- **URL**: http://localhost:5173
- **Features**: Hot reloading, TypeScript, Tailwind CSS
- **Config**: Environment variables in `.env`

### Backend (Go + Fiber)
- **API**: http://localhost:3000/api/v1
- **WebSocket**: ws://localhost:3000/ws/chat
- **Features**: Hot reloading with Air, PostgreSQL storage
- **Config**: `agentx-backend/config.json` and environment variables

### Database (PostgreSQL)
- **Connection**: localhost:5432
- **Database**: agentx
- **User**: agentx
- **Password**: agentx

## Available Commands

```bash
# Development
make dev          # Start all services in development mode
make stop         # Stop all services
make clean        # Clean up everything

# Docker
make docker       # Start with Docker Compose
make docker-logs  # View logs
make docker-reset # Reset and restart containers

# Individual services
make backend-dev  # Start only backend
make frontend-dev # Start only frontend
make db-up        # Start only PostgreSQL

# Building
make build        # Build both frontend and backend
make test         # Run all tests
```

## Configuration

### Environment Variables (.env)

```bash
# Frontend
VITE_API_URL=http://localhost:3000/api/v1

# Backend
AGENTX_PORT=3000
AGENTX_HOST=0.0.0.0

# Database (if not using Docker)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=agentx
POSTGRES_PASSWORD=agentx
POSTGRES_DB=agentx
```

### API Key Configuration

API keys are now stored securely in PostgreSQL database. After starting the application:

1. Open http://localhost:5173
2. Go to Settings
3. Configure your providers:
   - **OpenAI**: Add your API key from https://platform.openai.com/api-keys
   - **Anthropic**: Add your API key from https://console.anthropic.com/
   - **Ollama**: Auto-detected at http://localhost:11434
   - **LM Studio**: Auto-detected at http://localhost:1234

### Backend Config (agentx-backend/config.json)

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
    "password": "agentx",
    "database": "agentx",
    "sslmode": "disable"
  },
  "providers": {
    // Provider configurations
  }
}
```

## Development Workflow

1. **Start services**: `make dev`
2. **Make changes**: Files are watched and auto-reloaded
3. **View logs**: Check terminal output
4. **Stop services**: `Ctrl+C` or `make stop`

## Troubleshooting

### Port already in use
```bash
# Find and kill process using port
lsof -ti:3000 | xargs kill -9
lsof -ti:5173 | xargs kill -9
lsof -ti:5432 | xargs kill -9
```

### Database connection issues
```bash
# Reset database
make db-reset

# Check PostgreSQL logs
docker logs agentx-postgres
```

### Backend not starting
```bash
# Check Go dependencies
cd agentx-backend
go mod tidy

# Check Air is installed
which air || go install github.com/cosmtrek/air@latest
```

### Frontend not starting
```bash
# Clear node modules
rm -rf node_modules package-lock.json
npm install
```

## Production Deployment

### Using Docker

```bash
# Build images
docker-compose -f docker-compose.full.yml build

# Start in production mode
docker-compose -f docker-compose.full.yml up -d
```

### Manual Deployment

1. Build frontend:
```bash
npm run build
# Deploy dist/ to CDN or static hosting
```

2. Build backend:
```bash
cd agentx-backend
go build -o agentx-server cmd/server/main.go
# Deploy binary with config
```

3. Set up PostgreSQL on your server

4. Configure environment variables and reverse proxy

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│  React Frontend │────▶│   Go Backend    │────▶│   PostgreSQL    │
│   (Port 5173)   │     │  (Port 3000)    │     │  (Port 5432)    │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                        │
        │                        ├── OpenAI API
        │                        ├── Anthropic API
        └── WebSocket ───────────┤── Ollama (local)
                                └── LM Studio (local)
```

## Next Steps

1. Run `make dev` to start the application
2. Open http://localhost:5173 in your browser
3. Go to Settings and configure your AI providers
4. Start chatting with AI!

No need to edit `.env` files - all API keys are managed through the web UI!

For more details:
- Frontend docs: See main README.md
- Backend docs: See agentx-backend/README.md