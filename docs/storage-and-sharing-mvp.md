# Plan: Splitwiser - Shareable Bills MVP

## User Requirements
- **Primary goal**: MVP with shareable bill links (no accounts), extensible towards user accounts/groups later
- **Storage**: Considering SQLite vs BadgerDB
- **Privacy**: Default privacy (defer if complex initially)

## Current State Analysis

### Backend
- Proto already defines `CreateBill` and `GetBill` RPCs (currently unimplemented)
- Clean separation: calculator logic is pure, service layer ready for storage
- Basic models exist (`User`, `Group`) but minimal
- Service has commented placeholder for storage layer

### Frontend
- Single-page calculator - no persistence or bill history yet
- Only uses `CalculateSplit` RPC
- No URL routing or shareable links

## Storage Discussion: SQLite vs BadgerDB

### SQLite (Recommended ✅)
**Pros:**
- Relational structure perfect for bills (structured data with relationships)
- SQL queries for filtering, sorting, joining
- ACID transactions built-in
- **Pure Go support**: `modernc.org/sqlite` - **NO CGO required** ✅
  - Drop-in replacement for `mattn/go-sqlite3`
  - Cross-compiles easily for all platforms
  - No C compiler needed in build environment
- Can migrate to PostgreSQL later with minimal code changes
- Industry standard, well-understood
- Built-in indexes, foreign keys, constraints

**Cons:**
- Slightly heavier than key-value stores
- Write concurrency limits (fine for MVP scale)
- Pure Go version ~10% slower than CGO version (negligible for this use case)

**Best for:**
- Structured data (bills have items, participants, splits)
- Future queries like "get all bills by creator" or "bills containing participant X"
- Easy migration path to PostgreSQL when scaling
- **Teams that avoid CGO dependencies** (deployment, cross-compilation, Docker Alpine images)

### BadgerDB
**Pros:**
- Pure Go, no CGO required
- Fast key-value operations
- Embeddable like SQLite
- Good for high write throughput

**Cons:**
- Key-value store requires manual indexing
- Would need to build own query layer for "get bills by creator"
- Less intuitive for relational data (bills → items → participants)
- Harder migration path to relational DB later

**Best for:**
- Simple key-value lookups
- High-throughput write scenarios
- Cache-like use cases

### Recommendation: SQLite with `modernc.org/sqlite`

For Splitwiser's data model (bills with nested items and participants), **SQLite is the better choice** because:

1. **No CGO** ✅: `modernc.org/sqlite` is pure Go (user requirement satisfied!)
2. **Natural fit**: Bills are inherently relational (bill → items → participant assignments)
3. **Extensibility**: User's goal is to eventually support accounts/groups, which are relational
4. **Migration path**: Can swap SQLite for PostgreSQL later with minimal code changes
5. **Query flexibility**: Future features like "bill history" or "bills I'm in" are trivial with SQL
6. **Standard interface**: Uses `database/sql` - familiar to any Go developer

BadgerDB would require building an indexing/query layer manually, which is reinventing what SQLite already does well.

**TL;DR**: SQLite + `modernc.org/sqlite` gives you relational power without CGO hassles.

## Phase 1: Add Bill Persistence (MVP)

### Database Schema

```sql
CREATE TABLE bills (
    id TEXT PRIMARY KEY,           -- UUID
    title TEXT NOT NULL,
    total REAL NOT NULL,
    subtotal REAL NOT NULL,
    created_at INTEGER NOT NULL,   -- Unix timestamp
    -- Future: creator_id for auth
    -- Future: group_id for groups
);

CREATE TABLE items (
    id TEXT PRIMARY KEY,
    bill_id TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE
);

CREATE TABLE item_assignments (
    item_id TEXT NOT NULL,
    participant TEXT NOT NULL,     -- String for MVP (no user accounts yet)
    PRIMARY KEY (item_id, participant),
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
);

CREATE TABLE participants (
    bill_id TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (bill_id, name),
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE
);

-- Future indexes for performance
CREATE INDEX idx_bills_created_at ON bills(created_at);
```

### Backend Implementation

#### 1. Storage Layer (`backend/internal/storage/`)

Create abstraction for future flexibility:

```go
// storage/store.go
type Store interface {
    CreateBill(ctx context.Context, bill *models.Bill) error
    GetBill(ctx context.Context, billID string) (*models.Bill, error)
    // Future: ListBills, UpdateBill, DeleteBill
}

// storage/sqlite/sqlite.go
import (
    "database/sql"
    _ "modernc.org/sqlite" // Pure Go SQLite driver (no CGO!)
)

type SQLiteStore struct {
    db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    // Open database with pure Go driver
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, err
    }

    // Run migrations (create tables if not exist)
    if err := runMigrations(db); err != nil {
        return nil, err
    }

    return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) CreateBill(...)
func (s *SQLiteStore) GetBill(...)
```

**Key points:**
- Use `modernc.org/sqlite` driver (pure Go, no CGO)
- Driver name is `"sqlite"` not `"sqlite3"`
- Standard `database/sql` interface - easy to swap drivers later
- **Why abstraction?** Enables easy swap to PostgreSQL or other stores later without changing service layer

#### 2. Update Models (`backend/internal/models/`)

Enhance existing models to match database schema:

```go
type Bill struct {
    ID          string
    Title       string
    Items       []Item
    Total       float64
    Subtotal    float64
    Participants []string
    CreatedAt   int64
}

type Item struct {
    ID          string
    Description string
    Amount      float64
    AssignedTo  []string
}
```

#### 3. Update Service (`backend/internal/service/split_service.go`)

- Uncomment storage field, inject `Store` in constructor
- Implement `CreateBill`: validate, generate UUID, store, return split calculation
- Implement `GetBill`: retrieve from storage, return with cached split results

#### 4. Update Server (`backend/cmd/server/main.go`)

- Initialize SQLite store on startup
- Pass store to service constructor
- Handle migrations (create tables if not exist)

### Frontend Implementation

#### 1. Add Routing

Use Next.js routing:
- `/` - Current calculator (create new bill)
- `/bill/[id]` - View existing bill (shareable link)

#### 2. Update Home Page (`index.tsx`)

After successful `CalculateSplit`:
1. Call `CreateBill` RPC with results
2. Get `bill_id` in response
3. Redirect to `/bill/[id]`

#### 3. Create Bill View Page (`pages/bill/[id].tsx`)

- Fetch bill using `GetBill` RPC
- Display bill details (read-only)
- Show split results
- Shareable URL in address bar
- "Create New Bill" button

### Privacy Consideration

**MVP approach**: Bills are unlisted but not truly private
- Anyone with URL can view (like Google Docs "anyone with link")
- No authentication required
- No bill listing page (can't discover other bills)
- Good enough for MVP with friends

**Future**: Add optional password protection or auth-based privacy

## Phase 2: Extensibility for Future Auth

Design storage layer to support future user accounts with minimal changes:

### Database Changes (Future)
```sql
ALTER TABLE bills ADD COLUMN creator_id TEXT;
ALTER TABLE bills ADD COLUMN group_id TEXT;

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE,
    name TEXT,
    created_at INTEGER
);
```

### Backend Changes (Future)
- Add auth middleware (JWT or session)
- Update `CreateBill` to capture `creator_id` from auth context
- Add `ListBills(userID)` to fetch user's bill history
- Update models to include `Creator` field

### Why This Works
- Bill IDs are already UUIDs (secure, unguessable)
- Participant names stored as strings (can migrate to user IDs later)
- Storage interface allows swapping SQLite → PostgreSQL
- Service layer just needs auth context injected

## Critical Files

### To Create
- `backend/internal/storage/store.go` - Storage interface
- `backend/internal/storage/sqlite/sqlite.go` - SQLite implementation
- `backend/internal/storage/sqlite/migrations.go` - Schema setup
- `frontend/src/pages/bill/[id].tsx` - Bill view page

### To Modify
- `backend/internal/models/split.go` - Add Bill model fields
- `backend/internal/service/split_service.go` - Implement CreateBill/GetBill
- `backend/cmd/server/main.go` - Initialize storage
- `frontend/src/pages/index.tsx` - Add CreateBill call and redirect

## Testing Strategy

### Manual Testing Flow
1. Start backend: `make backend-run`
2. Start frontend: `make frontend-dev`
3. Create a bill with items and participants
4. Click "Calculate & Save" → redirects to `/bill/{id}`
5. Copy URL and open in incognito/different browser → bill loads
6. Restart backend → bill persists

### Automated Tests
- `backend/internal/storage/sqlite/sqlite_test.go` - Test CRUD operations
- Update `backend/internal/service/split_service_test.go` - Mock storage layer
- Add integration test: create bill → retrieve bill → verify data

## Decisions

1. **Bill title**: Auto-generate titles (items are optional)
   - **MVP**: Simple format like `"Bill - {date}"` or `"Split with {participants}"`
   - **Future idea** 💡: Generate fun names like Claude's session names ("effervescent-marinating-lark")
   - Could be related to group names: "jubilant-pizza-crew" or "snuggling-brunch-buddies"
   - Note: Keep list of adjective-noun-noun combinations for future implementation

2. **Bill expiry**: Keep forever (no auto-deletion)
   - Simple implementation, no cleanup jobs needed
   - Users can reference old splits
   - Can add manual deletion feature later if needed

3. **Database location**: `./data/bills.db`
   - Create `data/` directory if not exists on startup
   - Easy for Docker volume mounting
   - Clear location for deployments
   - Add to `.gitignore`

## Future Ideas (Parking Lot)

These are ideas to revisit after MVP is working:

1. **Fun auto-generated bill names** (like Claude session names)
   - Pattern: `{adjective}-{adjective}-{noun}` related to groups/food/money
   - Examples: "jubilant-pizza-crew", "generous-brunch-squad", "splitting-taco-friends"
   - Could use group context when groups are added
   - Implementation: word lists + random selection with seed from bill ID

2. **Progressive enhancement path**
   - MVP: Bills are unlisted but accessible via URL (like Google Docs "anyone with link")
   - Phase 2: Optional password protection per bill
   - Phase 3: User accounts with private bill history
   - Phase 4: Recurring groups with default participants

3. **Mobile optimization**
   - Receipt photo upload + OCR to extract items
   - Mobile-first UI improvements
   - PWA support for offline usage

## Implementation Plan

If approved, implementation order:
1. Create storage layer with SQLite
2. Implement CreateBill/GetBill RPCs with auto-generated titles
3. Add frontend routing and bill view page
4. Update home page to save and redirect
5. Add `data/` to `.gitignore`
6. Test end-to-end flow
7. Document API and shareable link format
