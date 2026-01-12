# Splitwiser

An alternative to Splitwise that removes the 3-entries-per-day limitation and provides granular item-level bill splitting with automatic tax/fee distribution.

## Features

- **Item-level splitting**: Assign individual items to specific people
- **Automatic tax distribution**: Tax and fees are proportionally distributed based on each person's subtotal
- **No usage limits**: Split as many bills as you need
- **gRPC API**: Type-safe communication between frontend and backend

## Tech Stack

- **Backend**: Go with gRPC
- **Frontend**: Next.js with TypeScript
- **Protocol**: Protocol Buffers for type-safe API contracts

## Quick Start

### Prerequisites

- Go 1.22+
- Bun 1.0+
- Protocol Buffers compiler (`protoc`)
- protoc-gen-go and protoc-gen-go-grpc plugins

Install protoc plugins:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Setup

```bash
# Install all dependencies and generate proto code
make install

# Or manually:
make proto
make backend-deps
make frontend-deps
```

### Development

Run backend (terminal 1):
```bash
make dev-backend
# Server will start on :50051
```

Run frontend (terminal 2):
```bash
make dev-frontend
# Frontend will start on http://localhost:3000
```

### Testing

```bash
# Run all backend tests
make test

# Run specific test
cd backend && go test ./internal/calculator -run TestCalculateSplit
```

## Project Structure

```
splitwiser/
├── proto/              # Protocol Buffer definitions
│   └── splitwiser.proto
├── backend/            # Go backend
│   ├── cmd/
│   │   └── server/     # Server entry point
│   ├── internal/
│   │   ├── calculator/ # Bill splitting logic
│   │   ├── models/     # Data models
│   │   └── service/    # gRPC service implementation
│   └── pkg/            # Public packages
├── frontend/           # Next.js frontend
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── api/        # gRPC client code
│   └── public/
└── Makefile
```

## How It Works

The core splitting algorithm distributes tax/fees proportionally:

1. Each person's items are tracked separately
2. Items can be shared between specific people
3. Tax is distributed proportionally: `person_total = person_subtotal × (1 + (total_tax / bill_subtotal))`

This ensures everyone pays their fair share including proportional tax.
