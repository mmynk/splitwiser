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
		bill.Title = generateTitle(bill.Items, bill.Participants)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert bill
	var groupID interface{} = nil
	if bill.GroupID != "" {
		groupID = bill.GroupID
	}
	var payerID interface{} = nil
	if bill.PayerID != "" {
		payerID = bill.PayerID
	}
	_, err = tx.ExecContext(ctx,
		"INSERT INTO bills (id, title, total, subtotal, created_at, group_id, payer_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		bill.ID, bill.Title, bill.Total, bill.Subtotal, bill.CreatedAt, groupID, payerID,
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
		for _, participant := range item.Participants {
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
	var groupID sql.NullString
	var payerID sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, total, subtotal, created_at, group_id, payer_id FROM bills WHERE id = ?",
		billID,
	).Scan(&bill.ID, &bill.Title, &bill.Total, &bill.Subtotal, &bill.CreatedAt, &groupID, &payerID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bill not found: %s", billID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bill: %w", err)
	}
	if groupID.Valid {
		bill.GroupID = groupID.String
	}
	if payerID.Valid {
		bill.PayerID = payerID.String
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
			item.Participants = append(item.Participants, participant)
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

// UpdateBill updates an existing bill, replacing all items and participants.
func (s *SQLiteStore) UpdateBill(ctx context.Context, bill *models.Bill) error {
	if bill.ID == "" {
		return fmt.Errorf("bill ID is required for update")
	}

	// Check if bill exists
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM bills WHERE id = ?", bill.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("bill not found: %s", bill.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to check bill existence: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update bill fields
	var groupID interface{} = nil
	if bill.GroupID != "" {
		groupID = bill.GroupID
	}
	var payerID interface{} = nil
	if bill.PayerID != "" {
		payerID = bill.PayerID
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE bills SET title = ?, total = ?, subtotal = ?, group_id = ?, payer_id = ? WHERE id = ?",
		bill.Title, bill.Total, bill.Subtotal, groupID, payerID, bill.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update bill: %w", err)
	}

	// Delete existing items (cascades to item_assignments via FK)
	_, err = tx.ExecContext(ctx, "DELETE FROM items WHERE bill_id = ?", bill.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing items: %w", err)
	}

	// Delete existing participants
	_, err = tx.ExecContext(ctx, "DELETE FROM participants WHERE bill_id = ?", bill.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing participants: %w", err)
	}

	// Insert new participants
	for _, name := range bill.Participants {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO participants (bill_id, name) VALUES (?, ?)",
			bill.ID, name,
		)
		if err != nil {
			return fmt.Errorf("failed to insert participant: %w", err)
		}
	}

	// Insert new items and their assignments
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
		for _, participant := range item.Participants {
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

// ListBillsByGroup retrieves all bills associated with a group.
func (s *SQLiteStore) ListBillsByGroup(ctx context.Context, groupID string) ([]*models.Bill, error) {
	// Get all bills for the group
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, total, subtotal, created_at, group_id FROM bills WHERE group_id = ? ORDER BY created_at DESC",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list bills by group: %w", err)
	}
	defer rows.Close()

	var bills []*models.Bill
	for rows.Next() {
		bill := &models.Bill{}
		var groupIDStr sql.NullString
		if err := rows.Scan(&bill.ID, &bill.Title, &bill.Total, &bill.Subtotal, &bill.CreatedAt, &groupIDStr); err != nil {
			return nil, fmt.Errorf("failed to scan bill: %w", err)
		}
		if groupIDStr.Valid {
			bill.GroupID = groupIDStr.String
		}

		// Get participants for this bill
		participantRows, err := s.db.QueryContext(ctx,
			"SELECT name FROM participants WHERE bill_id = ? ORDER BY name",
			bill.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get participants: %w", err)
		}

		for participantRows.Next() {
			var name string
			if err := participantRows.Scan(&name); err != nil {
				participantRows.Close()
				return nil, fmt.Errorf("failed to scan participant: %w", err)
			}
			bill.Participants = append(bill.Participants, name)
		}
		participantRows.Close()
		if err := participantRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate participants: %w", err)
		}

		// Get items for this bill (we include them for completeness, though the API might not need them)
		itemRows, err := s.db.QueryContext(ctx,
			"SELECT id, description, amount FROM items WHERE bill_id = ?",
			bill.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get items: %w", err)
		}

		for itemRows.Next() {
			var item models.Item
			if err := itemRows.Scan(&item.ID, &item.Description, &item.Amount); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan item: %w", err)
			}

			// Get assignments for this item
			assignRows, err := s.db.QueryContext(ctx,
				"SELECT participant FROM item_assignments WHERE item_id = ? ORDER BY participant",
				item.ID,
			)
			if err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to get item assignments: %w", err)
			}

			for assignRows.Next() {
				var participant string
				if err := assignRows.Scan(&participant); err != nil {
					assignRows.Close()
					itemRows.Close()
					return nil, fmt.Errorf("failed to scan assignment: %w", err)
				}
				item.Participants = append(item.Participants, participant)
			}
			assignRows.Close()
			if err := assignRows.Err(); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to iterate assignments: %w", err)
			}

			bill.Items = append(bill.Items, item)
		}
		itemRows.Close()
		if err := itemRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate items: %w", err)
		}

		bills = append(bills, bill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bills: %w", err)
	}

	return bills, nil
}

// generateTitle creates an auto-generated title using hybrid "Items - Participants" format.
func generateTitle(items []models.Item, participants []string) string {
	// Strategy: "Items - Participants"

	itemsStr := ""
	if len(items) > 0 {
		if len(items) == 1 {
			itemsStr = items[0].Description
		} else if len(items) <= 3 {
			descriptions := make([]string, len(items))
			for i, item := range items {
				descriptions[i] = item.Description
			}
			itemsStr = strings.Join(descriptions, ", ")
		} else {
			descriptions := make([]string, 2)
			descriptions[0] = items[0].Description
			descriptions[1] = items[1].Description
			itemsStr = fmt.Sprintf("%s & %d more", strings.Join(descriptions, ", "), len(items)-2)
		}
	}

	participantsStr := ""
	if len(participants) <= 3 {
		participantsStr = strings.Join(participants, ", ")
	} else {
		participantsStr = fmt.Sprintf("%s & %d others",
			strings.Join(participants[:2], ", "),
			len(participants)-2)
	}

	// Combine
	if itemsStr != "" && participantsStr != "" {
		return fmt.Sprintf("%s - %s", itemsStr, participantsStr)
	} else if itemsStr != "" {
		return itemsStr
	} else if participantsStr != "" {
		return fmt.Sprintf("Split with %s", participantsStr)
	} else {
		return fmt.Sprintf("Bill - %s", time.Now().Format("Jan 2, 2006"))
	}
}

// CreateGroup persists a new group to the database.
func (s *SQLiteStore) CreateGroup(ctx context.Context, group *models.Group) error {
	// Generate IDs if not set
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	if group.CreatedAt == 0 {
		group.CreatedAt = time.Now().Unix()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert group
	_, err = tx.ExecContext(ctx,
		"INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)",
		group.ID, group.Name, group.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert group: %w", err)
	}

	// Insert members
	for _, name := range group.Members {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO group_members (group_id, name) VALUES (?, ?)",
			group.ID, name,
		)
		if err != nil {
			return fmt.Errorf("failed to insert group member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetGroup retrieves a group by ID, including all members.
func (s *SQLiteStore) GetGroup(ctx context.Context, groupID string) (*models.Group, error) {
	// Get group
	group := &models.Group{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, created_at FROM groups WHERE id = ?",
		groupID,
	).Scan(&group.ID, &group.Name, &group.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Get members
	rows, err := s.db.QueryContext(ctx,
		"SELECT name FROM group_members WHERE group_id = ? ORDER BY name",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		group.Members = append(group.Members, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate group members: %w", err)
	}

	return group, nil
}

// ListGroups retrieves all groups with their members.
func (s *SQLiteStore) ListGroups(ctx context.Context) ([]*models.Group, error) {
	// Get all groups
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, created_at FROM groups ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		group := &models.Group{}
		if err := rows.Scan(&group.ID, &group.Name, &group.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate groups: %w", err)
	}

	// Get members for each group
	for _, group := range groups {
		memberRows, err := s.db.QueryContext(ctx,
			"SELECT name FROM group_members WHERE group_id = ? ORDER BY name",
			group.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get group members: %w", err)
		}

		for memberRows.Next() {
			var name string
			if err := memberRows.Scan(&name); err != nil {
				memberRows.Close()
				return nil, fmt.Errorf("failed to scan group member: %w", err)
			}
			group.Members = append(group.Members, name)
		}
		memberRows.Close()
		if err := memberRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate group members: %w", err)
		}
	}

	return groups, nil
}

// UpdateGroup updates an existing group, replacing all members.
func (s *SQLiteStore) UpdateGroup(ctx context.Context, group *models.Group) error {
	if group.ID == "" {
		return fmt.Errorf("group ID is required for update")
	}

	// Check if group exists
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM groups WHERE id = ?", group.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("group not found: %s", group.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update group fields
	_, err = tx.ExecContext(ctx,
		"UPDATE groups SET name = ? WHERE id = ?",
		group.Name, group.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	// Delete existing members
	_, err = tx.ExecContext(ctx, "DELETE FROM group_members WHERE group_id = ?", group.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing members: %w", err)
	}

	// Insert new members
	for _, name := range group.Members {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO group_members (group_id, name) VALUES (?, ?)",
			group.ID, name,
		)
		if err != nil {
			return fmt.Errorf("failed to insert group member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteGroup removes a group by ID.
// Bills associated with the group will have their group_id set to NULL (via ON DELETE SET NULL).
func (s *SQLiteStore) DeleteGroup(ctx context.Context, groupID string) error {
	// Check if group exists
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM groups WHERE id = ?", groupID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("group not found: %s", groupID)
	}
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	// Delete group (cascades to group_members, sets bills.group_id to NULL)
	_, err = s.db.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", groupID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}
