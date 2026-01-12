.PHONY: proto backend frontend test clean install dev

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

# Frontend is now static HTML/JS/CSS served by the backend
# No build step needed - just edit files in frontend/static/

# Development - runs backend which serves both API and static frontend
dev: backend-run

dev-backend: backend-run

# Frontend-only dev server (useful for frontend-only changes without full backend)
dev-frontend:
	@echo "Serving frontend at http://localhost:3000"
	@echo "Note: API calls will fail without backend running"
	cd frontend/static && python3 -m http.server 3000

# Install dependencies
install: backend-deps proto
	@echo "Setup complete!"

# Run all tests
test: backend-test

# Cleanup
clean:
	rm -rf backend/bin
	rm -rf backend/data/*.db
	rm -rf proto/gen
