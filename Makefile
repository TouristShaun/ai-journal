.PHONY: dev test build clean install-deps db-setup db-migrate tail-log

# Development
dev:
	@echo "Starting development services..."
	@mkdir -p logs
	@if pgrep -f "journal-server" > /dev/null; then \
		echo "Journal server already running. Use 'make tail-log' to view logs."; \
	else \
		cd backend && go run cmd/server/main.go > ../logs/server.log 2>&1 & \
		echo "Backend server started. PID: $$!"; \
	fi
	@if pgrep -f "vite" > /dev/null; then \
		echo "Frontend dev server already running."; \
	else \
		cd frontend && npm run dev > ../logs/frontend.log 2>&1 & \
		echo "Frontend server started. PID: $$!"; \
	fi

tail-log:
	@echo "=== Backend Logs ==="
	@tail -f logs/server.log & \
	echo "=== Frontend Logs ===" && \
	tail -f logs/frontend.log

# Testing
test:
	cd backend && go test -v ./...

test-quick:
	cd backend && go test ./...

# Building
build: build-backend build-frontend

build-backend:
	cd backend && go build -o ../bin/journal-server cmd/server/main.go

build-frontend:
	cd frontend && npm run build

# Dependencies
install-deps: install-go-deps install-node-deps install-ollama-models

install-go-deps:
	cd backend && go mod download

install-node-deps:
	cd frontend && npm install

install-ollama-models:
	@echo "Installing Ollama models..."
	ollama pull qwen2.5:7b
	ollama pull nomic-embed-text

# Database
db-setup:
	@echo "Setting up PostgreSQL with pgvector..."
	createdb journal_db || echo "Database already exists"
	psql -d journal_db -c "CREATE EXTENSION IF NOT EXISTS vector;"
	@echo "Database setup complete"

db-migrate:
	@echo "Running database migrations..."
	cd backend && go run cmd/migrate/main.go up

db-reset:
	@echo "Resetting database..."
	dropdb journal_db || echo "Database doesn't exist"
	make db-setup
	make db-migrate

# Cleanup
clean:
	rm -rf bin/ logs/
	cd frontend && rm -rf dist/

# MCP Agent
run-mcp-agent:
	cd mcp-agent && go run main.go

# Development helpers
lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint

format:
	cd backend && go fmt ./...
	cd frontend && npm run format

# Evaluation commands
eval-generate:
	@echo "Generating test data for evaluation..."
	cd backend && go run cmd/evaluate/main.go -cmd generate -size 100

eval-run:
	@echo "Running search evaluation..."
	cd backend && go run cmd/evaluate/main.go -cmd evaluate -mode all

eval-report:
	@echo "Generating evaluation report..."
	cd backend && go run cmd/evaluate/main.go -cmd report -format html

eval-full: eval-generate eval-run eval-report
	@echo "Full evaluation complete. Check evaluation_results/reports/ for the report."

eval-clean:
	rm -rf backend/evaluation_results/data/*
	rm -rf backend/evaluation_results/reports/*