.PHONY: proto backend frontend test clean install

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

# Frontend commands
frontend-deps:
	cd frontend && bun install

frontend-dev:
	cd frontend && bun run dev

frontend-build:
	cd frontend && bun run build

# Development
install: backend-deps frontend-deps proto
	@echo "Setup complete!"

dev-backend: backend-run

dev-frontend: frontend-dev

test: backend-test

# Cleanup
clean:
	rm -rf backend/bin
	rm -rf proto/gen
	rm -rf frontend/.next
	rm -rf frontend/node_modules
	rm -rf frontend/out
