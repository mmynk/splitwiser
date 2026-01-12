// Package sqlite provides a SQLite-backed implementation of the storage.Store interface.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGO)

	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage"
)

// Ensure SQLiteStore implements storage.Store
var _ storage.Store = (*SQLiteStore)(nil)

// SQLiteStore implements storage.Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New creates a new SQLiteStore with the given database path.
// It creates the parent directories and runs migrations automatically.
func New(dbPath string) (*SQLiteStore, error) {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with pure Go driver
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateBill persists a new bill to the database.
func (s *SQLiteStore) CreateBill(ctx context.Context, bill *models.Bill) error {
	// Generate IDs if not set
	if bill.ID == "" {
		bill.ID = uuid.New().String()
	}
	if bill.CreatedAt == 0 {
		bill.CreatedAt = time.Now().Unix()
	}
	if bill.Title == "" {
		bill.Title = generateTitle(bill.Participants)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert bill
	_, err = tx.ExecContext(ctx,
		"INSERT INTO bills (id, title, total, subtotal, created_at) VALUES (?, ?, ?, ?, ?)",
		bill.ID, bill.Title, bill.Total, bill.Subtotal, bill.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert bill: %w", err)
	}

	// Insert participants
	for _, name := range bill.Participants {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO participants (bill_id, name) VALUES (?, ?)",
			bill.ID, name,
		)
		if err != nil {
			return fmt.Errorf("failed to insert participant: %w", err)
		}
	}

	// Insert items and their assignments
	for i := range bill.Items {
		item := &bill.Items[i]
		if item.ID == "" {
			item.ID = uuid.New().String()
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO items (id, bill_id, description, amount) VALUES (?, ?, ?, ?)",
			item.ID, bill.ID, item.Description, item.Amount,
		)
		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}

		// Insert item assignments
		for _, participant := range item.AssignedTo {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO item_assignments (item_id, participant) VALUES (?, ?)",
				item.ID, participant,
			)
			if err != nil {
				return fmt.Errorf("failed to insert item assignment: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetBill retrieves a bill by ID, including all items and participants.
func (s *SQLiteStore) GetBill(ctx context.Context, billID string) (*models.Bill, error) {
	// Get bill
	bill := &models.Bill{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, total, subtotal, created_at FROM bills WHERE id = ?",
		billID,
	).Scan(&bill.ID, &bill.Title, &bill.Total, &bill.Subtotal, &bill.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bill not found: %s", billID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bill: %w", err)
	}

	// Get participants
	rows, err := s.db.QueryContext(ctx,
		"SELECT name FROM participants WHERE bill_id = ? ORDER BY name",
		billID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		bill.Participants = append(bill.Participants, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate participants: %w", err)
	}

	// Get items with their assignments
	itemRows, err := s.db.QueryContext(ctx,
		"SELECT id, description, amount FROM items WHERE bill_id = ?",
		billID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item models.Item
		if err := itemRows.Scan(&item.ID, &item.Description, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Get assignments for this item
		assignRows, err := s.db.QueryContext(ctx,
			"SELECT participant FROM item_assignments WHERE item_id = ? ORDER BY participant",
			item.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get item assignments: %w", err)
		}

		for assignRows.Next() {
			var participant string
			if err := assignRows.Scan(&participant); err != nil {
				assignRows.Close()
				return nil, fmt.Errorf("failed to scan assignment: %w", err)
			}
			item.AssignedTo = append(item.AssignedTo, participant)
		}
		assignRows.Close()
		if err := assignRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate assignments: %w", err)
		}

		bill.Items = append(bill.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate items: %w", err)
	}

	return bill, nil
}

// generateTitle creates an auto-generated title from participants.
func generateTitle(participants []string) string {
	if len(participants) == 0 {
		return fmt.Sprintf("Bill - %s", time.Now().Format("Jan 2, 2006"))
	}
	if len(participants) <= 3 {
		return fmt.Sprintf("Split with %s", strings.Join(participants, ", "))
	}
	return fmt.Sprintf("Split with %s and %d others",
		strings.Join(participants[:2], ", "),
		len(participants)-2,
	)
}
