.PHONY: proto backend frontend test clean install dev docker-build docker-run docker-up docker-down frontend-deps frontend-build frontend-dev

# Generate Protocol Buffers with Connect
proto:
	@echo "Generating protobuf code with Connect..."
	@mkdir -p backend/pkg/proto
	PATH="$$HOME/go/bin:$$PATH" protoc --go_out=backend/pkg/proto --go_opt=paths=source_relative \
		--connect-go_out=backend/pkg/proto --connect-go_opt=paths=source_relative \
		-I proto proto/*.proto

# Backend commands
backend-deps:
	cd backend && go mod download && go mod tidy

backend-build: proto
	cd backend && go build -o bin/server ./cmd/server

backend-run: proto
	cd backend && go run ./cmd/server

backend-test:
	cd backend && go test ./... -v

frontend-test:
	bun frontend/test/import-validator.test.js

# Frontend (Svelte + Vite + TypeScript SPA, outputs to frontend/static/)
frontend-deps:
	cd frontend && bun install

frontend-build: frontend-deps
	cd frontend && bun run build

frontend-dev: frontend-deps
	cd frontend && bun run dev

# Development - runs backend which serves both API and static frontend
dev: backend-run

dev-backend: backend-run

# Install dependencies
install: backend-deps proto frontend-deps
	@echo "Setup complete!"

# Run all tests
test: backend-test frontend-test

# Cleanup
clean:
	rm -rf backend/bin
	rm -rf backend/data/*.db
	rm -rf proto/gen

# Docker commands
docker-build:
	podman build -t splitwiser .

docker-run:
	podman run -p 9090:8080 -v splitwiser-data:/app/data splitwiser

docker-up:
	podman compose up --build

docker-down:
	podman compose down
