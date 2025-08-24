# DayBoard Project Makefile
.PHONY: help build test clean run stop logs docker-build docker-run docker-stop

# Default target
help:
	@echo "DayBoard Project Commands:"
	@echo "  build           - Build all services"
	@echo "  test            - Run tests for all services"
	@echo "  clean           - Clean build artifacts"
	@echo "  run             - Run the entire stack with Docker Compose"
	@echo "  stop            - Stop all running containers"
	@echo "  logs            - Show logs from all services"
	@echo "  docker-build    - Build Docker images"
	@echo "  docker-run      - Run containers in background"
	@echo "  docker-stop     - Stop and remove containers"

# Build all services
build:
	@echo "🔨 Building Go backend..."
	cd backend && go build -o dayboard-server ./cmd/server
	@echo "🔨 Building Java microservice..."
	cd document-processor && mvn clean package -DskipTests
	@echo "✅ All services built successfully"

# Run tests
test:
	@echo "🧪 Running Go tests..."
	cd backend && go test ./... -v
	@echo "🧪 Running Java tests..."
	cd document-processor && mvn test
	@echo "✅ All tests passed"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	cd backend && rm -f dayboard-server
	cd document-processor && mvn clean
	docker system prune -f
	@echo "✅ Cleanup completed"

# Run the entire stack
run:
	@echo "🚀 Starting DayBoard stack..."
	docker-compose up --build

# Stop all containers
stop:
	@echo "🛑 Stopping DayBoard stack..."
	docker-compose down

# Show logs
logs:
	docker-compose logs -f

# Build Docker images
docker-build:
	@echo "🐳 Building Docker images..."
	docker-compose build

# Run containers in background
docker-run:
	@echo "🚀 Starting containers in background..."
	docker-compose up -d --build

# Stop and remove containers
docker-stop:
	@echo "🛑 Stopping and removing containers..."
	docker-compose down -v

# Development setup
dev-setup:
	@echo "⚙️ Setting up development environment..."
	cp backend/.env.example backend/.env
	@echo "📝 Please edit backend/.env with your API keys"
	@echo "✅ Development setup completed"

# Database migrations
migrate:
	@echo "📊 Running database migrations..."
	cd backend && go run cmd/migrate/main.go up

# Quick development start
dev:
	@echo "🚀 Starting development environment..."
	make docker-run
	@echo "🌐 Backend: http://localhost:8080"
	@echo "🔧 Document Processor: http://localhost:8081"
	@echo "📊 PostgreSQL: localhost:5432"

# Production deployment check
prod-check:
	@echo "🔍 Running production readiness checks..."
	docker-compose config
	@echo "✅ Docker Compose configuration is valid"
