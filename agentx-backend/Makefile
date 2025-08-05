.PHONY: build run dev test clean deps air-install db-up db-down db-reset migrate

# Build the application
build:
	go build -o agentx-server cmd/server/main.go

# Run the application
run: build
	./agentx-server

# Run with Air for hot reloading
dev:
	air

# Install Air for development
air-install:
	go install github.com/cosmtrek/air@latest

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -f agentx-server
	rm -rf tmp/
	rm -f *.log

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o agentx-server-linux-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build -o agentx-server-darwin-amd64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -o agentx-server-windows-amd64.exe cmd/server/main.go

# Create a release build
release:
	go build -ldflags="-s -w" -o agentx-server cmd/server/main.go

# Database commands
db-up:
	docker-compose -f docker-compose.dev.yml up -d

db-down:
	docker-compose -f docker-compose.dev.yml down

db-reset: db-down
	docker-compose -f docker-compose.dev.yml down -v
	docker-compose -f docker-compose.dev.yml up -d

# Run migrations
migrate:
	@echo "Migrations run automatically on server start"

# Setup development environment
setup: deps air-install db-up
	@echo "Development environment setup complete!"
	@echo "PostgreSQL is running on localhost:5432"
	@echo "Copy config.example.json to config.json and add your API keys"
	@echo "Run 'make dev' to start the server with hot reloading"