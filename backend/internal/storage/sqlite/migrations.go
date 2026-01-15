package sqlite

import "database/sql"

// migrations contains the SQL statements to set up the database schema.
// These run on startup to ensure tables exist.
// IMPORTANT: Users table must be created first, then groups, then bills (for foreign key constraints).
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    creator_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS bills (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    total REAL NOT NULL,
    subtotal REAL NOT NULL,
    created_at INTEGER NOT NULL,
    creator_id TEXT NOT NULL,
    group_id TEXT,
    payer_id TEXT NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL,
    FOREIGN KEY (payer_id) REFERENCES users(id) ON DELETE RESTRICT
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
    user_id TEXT NOT NULL,
    PRIMARY KEY (item_id, user_id),
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS participants (
    bill_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    PRIMARY KEY (bill_id, user_id),
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_items_bill_id ON items(bill_id);
CREATE INDEX IF NOT EXISTS idx_item_assignments_item_id ON item_assignments(item_id);
CREATE INDEX IF NOT EXISTS idx_item_assignments_user_id ON item_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_participants_bill_id ON participants(bill_id);
CREATE INDEX IF NOT EXISTS idx_participants_user_id ON participants(user_id);
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_bills_group_id ON bills(group_id);
CREATE INDEX IF NOT EXISTS idx_bills_creator_id ON bills(creator_id);
CREATE INDEX IF NOT EXISTS idx_groups_creator_id ON groups(creator_id);
`

// runMigrations executes the schema setup.
func runMigrations(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
