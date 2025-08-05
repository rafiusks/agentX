.PHONY: help dev docker start stop clean setup install backend-dev frontend-dev

# Default target
help:
	@echo "AgentX Development Commands:"
	@echo "  make setup      - Initial setup (install deps, create configs)"
	@echo "  make dev        - Start development environment (all services)"
	@echo "  make docker     - Start with Docker Compose"
	@echo "  make stop       - Stop all services"
	@echo "  make clean      - Clean up containers and data"
	@echo "  make backend-dev - Start only backend in dev mode"
	@echo "  make frontend-dev - Start only frontend in dev mode"

# Initial setup
setup:
	@echo "Setting up AgentX development environment..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@if [ ! -f agentx-backend/config.json ]; then cp agentx-backend/config.example.json agentx-backend/config.json; fi
	@echo "Installing frontend dependencies..."
	npm install
	@echo "Installing backend dependencies..."
	cd agentx-backend && go mod download
	@echo "Installing Air for Go hot reloading..."
	go install github.com/cosmtrek/air@latest
	@echo ""
	@echo "✅ Setup complete!"
	@echo "⚠️  Don't forget to add your API keys to .env file"
	@echo "Run 'make dev' to start the development environment"

# Install/update dependencies
install:
	npm install
	cd agentx-backend && go mod download

# Start development environment
dev:
	./start-dev.sh

# Start with Docker
docker:
	./start-docker.sh

# Start individual services
backend-dev:
	cd agentx-backend && make db-up && air

frontend-dev:
	npm run dev

# Stop all services
stop:
	@echo "Stopping all services..."
	docker-compose -f docker-compose.full.yml down 2>/dev/null || true
	cd agentx-backend && docker-compose -f docker-compose.dev.yml down 2>/dev/null || true
	@pkill -f "air" 2>/dev/null || true
	@pkill -f "npm run dev" 2>/dev/null || true
	@echo "All services stopped"

# Clean up everything
clean: stop
	@echo "Cleaning up..."
	docker-compose -f docker-compose.full.yml down -v 2>/dev/null || true
	cd agentx-backend && docker-compose -f docker-compose.dev.yml down -v 2>/dev/null || true
	rm -rf agentx-backend/tmp
	rm -rf node_modules
	rm -rf agentx-backend/vendor
	@echo "Cleanup complete"

# Docker commands
docker-build:
	docker-compose -f docker-compose.full.yml build

docker-up:
	docker-compose -f docker-compose.full.yml up -d

docker-down:
	docker-compose -f docker-compose.full.yml down

docker-logs:
	docker-compose -f docker-compose.full.yml logs -f

docker-reset: docker-down
	docker-compose -f docker-compose.full.yml down -v
	docker-compose -f docker-compose.full.yml up -d

# Database commands (proxy to backend)
db-up:
	cd agentx-backend && make db-up

db-down:
	cd agentx-backend && make db-down

db-reset:
	cd agentx-backend && make db-reset

# Backend commands (proxy)
backend-build:
	cd agentx-backend && make build

backend-test:
	cd agentx-backend && make test

# Frontend commands
frontend-build:
	npm run build

frontend-test:
	npm test

# Full build
build: backend-build frontend-build

# Run all tests
test: backend-test frontend-test