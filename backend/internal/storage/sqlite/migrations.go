package sqlite

import "database/sql"

// migrations contains the SQL statements to set up the database schema.
// These run on startup to ensure tables exist.
// IMPORTANT: Groups tables must be created BEFORE bills table due to foreign key constraint.
const schema = `
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (group_id, name),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS bills (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    total REAL NOT NULL,
    subtotal REAL NOT NULL,
    created_at INTEGER NOT NULL,
    group_id TEXT,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    bill_id TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS item_assignments (
    item_id TEXT NOT NULL,
    participant TEXT NOT NULL,
    PRIMARY KEY (item_id, participant),
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS participants (
    bill_id TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (bill_id, name),
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_items_bill_id ON items(bill_id);
CREATE INDEX IF NOT EXISTS idx_item_assignments_item_id ON item_assignments(item_id);
CREATE INDEX IF NOT EXISTS idx_participants_bill_id ON participants(bill_id);
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_bills_group_id ON bills(group_id);
`

// runMigrations executes the schema setup.
func runMigrations(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
