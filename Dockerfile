# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install protoc and plugins for proto generation
RUN apk add --no-cache protobuf protobuf-dev
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Copy go mod files first for caching
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy proto and generate
COPY proto/ ./proto/
RUN mkdir -p backend/pkg/proto
RUN protoc --go_out=backend/pkg/proto --go_opt=paths=source_relative \
    --connect-go_out=backend/pkg/proto --connect-go_opt=paths=source_relative \
    -I proto proto/*.proto

# Copy backend source and build
COPY backend/ ./backend/
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Create non-root user
RUN adduser -D -u 1000 splitwiser

# Copy binary
COPY --from=builder /app/server /app/server

# Copy static files
COPY frontend/static/ /app/frontend/static/

# Create data directory and set permissions
RUN mkdir -p /app/data && chown -R splitwiser:splitwiser /app

USER splitwiser

# Set environment variables for paths
ENV DB_PATH=/app/data/bills.db
ENV STATIC_PATH=/app/frontend/static

EXPOSE 8080

# Volume for SQLite database
VOLUME ["/app/data"]

CMD ["/app/server"]
