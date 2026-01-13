# Agent Instructions

This file provides guidance to AI coding assistants when working with code in this repository.

## Development Philosophy

**Priorities:**
1. Move fast
2. Do things right

**Non-priorities:**
- Backward compatibility (early development, not a concern)

Feel free to tear down and recreate as needed. Clean code over legacy support.

## Project Overview

Splitwiser is a free and open source alternative to Splitwise. It improves upon Splitwise with granular item-level splitting and automatic tax/fee distribution.

**Core USP**: Users can add individual items from a receipt and assign them to different people, with the system automatically calculating proportional tax, tips, and fees based on each person's subtotal.

## Architecture

### Backend (Go + Connect RPC)
- Connect RPC API with Protocol Buffers for type-safe communication
- Serves over HTTP/2 (h2c) on port 8080 with browser-native support
- Core calculator logic for bill splitting algorithm
- Structured logging with slog for request/response tracing
- Handles bill management and split calculations

### Frontend (Plain HTML/JS/CSS)
- Static files served by Go backend
- Located in `frontend/static/`
- Simple, focused UI for receipt entry and split calculation
- Calls Connect RPC API via fetch

### Protocol Buffers
- API contract defined in `proto/splitwiser.proto`
- Generates Go and TypeScript client/server code
- Core services: `SplitService` for calculations and bill management

## Development Commands

### Setup
```bash
make install              # Install all dependencies + generate proto code
make proto               # Generate protobuf code for Go and TypeScript
```

### Backend
```bash
make backend-build       # Build server binary to backend/bin/server
make backend-run         # Run backend server (port :8080)
make backend-test        # Run all tests

# Manual commands:
cd backend && go test ./...
cd backend && go test ./internal/calculator -run TestCalculateSplit -v
cd backend && ./bin/server
```

### Frontend
Frontend is plain HTML/JS/CSS served by the Go backend. No build step needed.
Files are in `frontend/static/`.

## Testing Guidelines

**IMPORTANT**: Always add integration tests for new API endpoints. Do NOT use curl-based manual tests.

- **Service-level integration tests**: `backend/internal/service/split_service_test.go`
  - Tests full RPC request/response cycle with real HTTP server
  - Uses `setupTestServer()` helper for consistent test setup
  - Each test gets isolated temp SQLite database

- **Storage-level unit tests**: `backend/internal/storage/sqlite/sqlite_test.go`
  - Tests database operations directly

- **Calculator unit tests**: `backend/internal/calculator/split_test.go`
  - Tests core splitting algorithm

When adding a new RPC endpoint:
1. Add service-level integration test in `split_service_test.go`
2. Add storage-level test in `sqlite_test.go` if new storage method
3. Run `make backend-test` to verify all tests pass

### Manual API Testing (Reference Only)
The curl examples below are for quick manual testing reference. **Always prefer writing integration tests.**

```bash
# Using curl (Connect supports standard HTTP):
curl -X POST http://localhost:8080/splitwiser.v1.SplitService/CalculateSplit \
  -H "Content-Type: application/json" \
  -d '{
    "items": [{"description": "Pizza", "amount": 20, "participants": ["Alice", "Bob"]}],
    "total": 33,
    "subtotal": 30,
    "participants": ["Alice", "Bob"]
  }'

# Or use grpcurl (Connect is gRPC-compatible):
grpcurl -plaintext -d '{"items":[{"description":"Pizza","amount":20,"participants":["Alice","Bob"]}],"total":33,"subtotal":30,"participants":["Alice","Bob"]}' localhost:8080 splitwiser.v1.SplitService/CalculateSplit
```

## Key Splitting Algorithm

The splitting logic works as follows:

1. **No Items Mode**: When no items are specified, split total equally among all participants
2. **Individual Items**: Each person's items are tracked separately with specific assignments
3. **Shared Items**: Items can be split among a subset of participants (not just all-or-one)
4. **Proportional Tax**: Tax/fees distributed based on each person's subtotal ratio

**Core Formula**: `person_total = person_subtotal × (1 + (total_tax / bill_subtotal))`

**Equal Split Formula** (when no items): Each person pays `bill_total / number_of_participants`

Implementation: `backend/internal/calculator/split.go`

## Critical Implementation Notes

### Tax/Fee Calculation
- Must handle edge case where subtotal is zero (return error)
- Proportional distribution based on pre-tax subtotals
- Rounding: accumulate errors to person with largest share (TODO)

### Data Model (proto/splitwiser.proto)
- **Bill**: Contains items, total, subtotal, participants, created timestamp
- **Item**: Description, amount, list of assigned participant IDs
- **PersonSplit**: Calculated subtotal, tax, and total for one person
- Items can be assigned to multiple people (amount splits equally)

### Known Issues from Original Script
The Python script (`../scripts/splitwiser.py`) has bugs that should NOT be replicated:
- Line 29-33: Exclusion logic incorrectly modifies subtotal
- No validation that `true_subtotal` matches calculated subtotal
- Missing validation for negative amounts or zero participants

### Proto Generation
After modifying `proto/splitwiser.proto`:
1. Run `make proto` to regenerate Go code with Connect
2. Update service implementations in `backend/internal/service/`
3. Frontend uses plain fetch (no TypeScript generation needed)

## Code Organization

```
backend/
├── cmd/server/          # Server entry point (main.go with Connect HTTP server)
├── internal/
│   ├── calculator/      # Core splitting algorithm (pure functions)
│   ├── service/         # Connect service implementations
│   ├── storage/         # SQLite storage layer
│   └── models/          # Domain models
├── pkg/
│   └── proto/          # Generated protobuf code
│       └── protoconnect/ # Generated Connect service code
├── data/               # SQLite database (bills.db)

frontend/
└── static/             # Plain HTML/JS/CSS (served by backend)
    ├── index.html      # Bill creation form
    ├── bill.html       # Bill view/edit page
    ├── app.js          # Bill creation form logic
    ├── bill.js         # Bill view/edit logic
    ├── form-components.js  # Shared form components (participants, items)
    └── styles.css      # Styling

proto/
└── splitwiser.proto    # API contract (source of truth)
```
