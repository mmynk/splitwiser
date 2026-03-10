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

// nullString returns a sql.NullString for a string value, treating empty string as NULL.
func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
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
	_, err = tx.ExecContext(ctx,
		"INSERT INTO bills (id, title, total, subtotal, created_at, group_id, payer_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		bill.ID, bill.Title, bill.Total, bill.Subtotal, bill.CreatedAt,
		nullString(bill.GroupID), nullString(bill.PayerID),
	)
	if err != nil {
		return fmt.Errorf("failed to insert bill: %w", err)
	}

	// Insert participants
	for _, p := range bill.Participants {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO participants (bill_id, name, user_id) VALUES (?, ?, ?)",
			bill.ID, p.DisplayName, nullString(p.UserID),
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

		// Insert item assignments (display names)
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
		"SELECT name, user_id FROM participants WHERE bill_id = ? ORDER BY name",
		billID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var userID sql.NullString
		if err := rows.Scan(&name, &userID); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		p := models.BillParticipant{DisplayName: name}
		if userID.Valid {
			p.UserID = userID.String
		}
		bill.Participants = append(bill.Participants, p)
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

	_, err = tx.ExecContext(ctx,
		"UPDATE bills SET title = ?, total = ?, subtotal = ?, group_id = ?, payer_id = ? WHERE id = ?",
		bill.Title, bill.Total, bill.Subtotal, nullString(bill.GroupID), nullString(bill.PayerID), bill.ID,
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
	for _, p := range bill.Participants {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO participants (bill_id, name, user_id) VALUES (?, ?, ?)",
			bill.ID, p.DisplayName, nullString(p.UserID),
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

// DeleteBill removes a bill and its associated data (items, participants, assignments).
func (s *SQLiteStore) DeleteBill(ctx context.Context, billID string) error {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM bills WHERE id = ?", billID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("bill not found: %s", billID)
	}
	if err != nil {
		return fmt.Errorf("failed to check bill existence: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM bills WHERE id = ?", billID)
	if err != nil {
		return fmt.Errorf("failed to delete bill: %w", err)
	}

	return nil
}

// ListBillsByGroup retrieves all bills associated with a group.
func (s *SQLiteStore) ListBillsByGroup(ctx context.Context, groupID string) ([]*models.Bill, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, total, subtotal, payer_id, created_at, group_id FROM bills WHERE group_id = ? ORDER BY created_at DESC",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list bills by group: %w", err)
	}
	defer rows.Close()

	var bills []*models.Bill
	for rows.Next() {
		bill := &models.Bill{}
		var payerIDStr sql.NullString
		var groupIDStr sql.NullString
		if err := rows.Scan(&bill.ID, &bill.Title, &bill.Total, &bill.Subtotal, &payerIDStr, &bill.CreatedAt, &groupIDStr); err != nil {
			return nil, fmt.Errorf("failed to scan bill: %w", err)
		}
		if payerIDStr.Valid {
			bill.PayerID = payerIDStr.String
		}
		if groupIDStr.Valid {
			bill.GroupID = groupIDStr.String
		}

		bill.Participants, err = s.getParticipants(ctx, bill.ID)
		if err != nil {
			return nil, err
		}

		bill.Items, err = s.getItemsWithAssignments(ctx, bill.ID)
		if err != nil {
			return nil, err
		}

		bills = append(bills, bill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bills: %w", err)
	}

	return bills, nil
}

// ListBillsByParticipant retrieves all bills where the given user_id is linked as a participant.
func (s *SQLiteStore) ListBillsByParticipant(ctx context.Context, userID string) ([]*models.Bill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.total, b.subtotal, b.payer_id, b.group_id, b.created_at
		FROM bills b
		INNER JOIN participants p ON b.id = p.bill_id
		WHERE p.user_id = ?
		ORDER BY b.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list bills by participant: %w", err)
	}
	defer rows.Close()

	var bills []*models.Bill
	for rows.Next() {
		bill := &models.Bill{}
		var payerID sql.NullString
		var groupID sql.NullString
		if err := rows.Scan(&bill.ID, &bill.Title, &bill.Total, &bill.Subtotal, &payerID, &groupID, &bill.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bill: %w", err)
		}
		if payerID.Valid {
			bill.PayerID = payerID.String
		}
		if groupID.Valid {
			bill.GroupID = groupID.String
		}

		bill.Participants, err = s.getParticipants(ctx, bill.ID)
		if err != nil {
			return nil, err
		}

		bills = append(bills, bill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bills: %w", err)
	}

	return bills, nil
}

// getParticipants is a helper that fetches participants for a bill.
func (s *SQLiteStore) getParticipants(ctx context.Context, billID string) ([]models.BillParticipant, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name, user_id FROM participants WHERE bill_id = ? ORDER BY name",
		billID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	defer rows.Close()

	var participants []models.BillParticipant
	for rows.Next() {
		var name string
		var userID sql.NullString
		if err := rows.Scan(&name, &userID); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		p := models.BillParticipant{DisplayName: name}
		if userID.Valid {
			p.UserID = userID.String
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// getItemsWithAssignments is a helper that fetches items and their participant assignments.
func (s *SQLiteStore) getItemsWithAssignments(ctx context.Context, billID string) ([]models.Item, error) {
	itemRows, err := s.db.QueryContext(ctx,
		"SELECT id, description, amount FROM items WHERE bill_id = ?",
		billID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer itemRows.Close()

	var items []models.Item
	for itemRows.Next() {
		var item models.Item
		if err := itemRows.Scan(&item.ID, &item.Description, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

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

		items = append(items, item)
	}
	return items, itemRows.Err()
}

// Stats holds aggregate counts for observability metrics.
type Stats struct {
	Users  int64
	Bills  int64
	Groups int64
}

// GetStats returns aggregate counts of users, bills, and groups.
// It runs a single query with three subselects, so it's cheap to call on each scrape.
func (s *SQLiteStore) GetStats(ctx context.Context) (Stats, error) {
	var stats Stats
	row := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM bills),
			(SELECT COUNT(*) FROM groups)
	`)
	if err := row.Scan(&stats.Users, &stats.Bills, &stats.Groups); err != nil {
		return Stats{}, fmt.Errorf("failed to get stats: %w", err)
	}
	return stats, nil
}

// generateTitle creates an auto-generated title using hybrid "Items - Participants" format.
func generateTitle(items []models.Item, participants []models.BillParticipant) string {
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
			itemsStr = fmt.Sprintf("%s, %s & %d more", items[0].Description, items[1].Description, len(items)-2)
		}
	}

	participantsStr := ""
	names := make([]string, len(participants))
	for i, p := range participants {
		names[i] = p.DisplayName
	}
	if len(names) <= 3 {
		participantsStr = strings.Join(names, ", ")
	} else {
		participantsStr = fmt.Sprintf("%s & %d others", strings.Join(names[:2], ", "), len(names)-2)
	}

	if itemsStr != "" && participantsStr != "" {
		return fmt.Sprintf("%s - %s", itemsStr, participantsStr)
	} else if itemsStr != "" {
		return itemsStr
	} else if participantsStr != "" {
		return fmt.Sprintf("Split with %s", participantsStr)
	}
	return fmt.Sprintf("Bill - %s", time.Now().Format("Jan 2, 2006"))
}

// CreateGroup persists a new group to the database.
func (s *SQLiteStore) CreateGroup(ctx context.Context, group *models.Group) error {
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

	_, err = tx.ExecContext(ctx,
		"INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)",
		group.ID, group.Name, group.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert group: %w", err)
	}

	for _, m := range group.Members {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO group_members (group_id, name, user_id) VALUES (?, ?, ?)",
			group.ID, m.DisplayName, nullString(m.UserID),
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

	group.Members, err = s.getGroupMembers(ctx, groupID)
	return group, err
}

// ListGroupsByUser retrieves all groups where the given user_id is a member.
func (s *SQLiteStore) ListGroupsByUser(ctx context.Context, userID string) ([]*models.Group, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT g.id, g.name, g.created_at
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ?
		ORDER BY g.created_at DESC`,
		userID,
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

	for _, group := range groups {
		group.Members, err = s.getGroupMembers(ctx, group.ID)
		if err != nil {
			return nil, err
		}
	}

	return groups, nil
}

// UpdateGroup updates an existing group, replacing all members.
func (s *SQLiteStore) UpdateGroup(ctx context.Context, group *models.Group) error {
	if group.ID == "" {
		return fmt.Errorf("group ID is required for update")
	}

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

	_, err = tx.ExecContext(ctx,
		"UPDATE groups SET name = ? WHERE id = ?",
		group.Name, group.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM group_members WHERE group_id = ?", group.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing members: %w", err)
	}

	for _, m := range group.Members {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO group_members (group_id, name, user_id) VALUES (?, ?, ?)",
			group.ID, m.DisplayName, nullString(m.UserID),
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

// AddGroupMembers adds members (by display name only) to a group idempotently.
// Deprecated: use AddGroupMembersWithIDs for members with optional user IDs.
func (s *SQLiteStore) AddGroupMembers(ctx context.Context, groupID string, memberIDs []string) error {
	members := make([]models.GroupMember, len(memberIDs))
	for i, name := range memberIDs {
		members[i] = models.GroupMember{DisplayName: name}
	}
	return s.AddGroupMembersWithIDs(ctx, groupID, members)
}

// AddGroupMembersWithIDs adds members (with optional user IDs) to a group idempotently.
func (s *SQLiteStore) AddGroupMembersWithIDs(ctx context.Context, groupID string, members []models.GroupMember) error {
	if len(members) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, m := range members {
		_, err = tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO group_members (group_id, name, user_id) VALUES (?, ?, ?)",
			groupID, m.DisplayName, nullString(m.UserID),
		)
		if err != nil {
			return fmt.Errorf("failed to add group member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteGroup removes a group by ID.
func (s *SQLiteStore) DeleteGroup(ctx context.Context, groupID string) error {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM groups WHERE id = ?", groupID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("group not found: %s", groupID)
	}
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", groupID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// getGroupMembers is a helper that fetches members for a group.
func (s *SQLiteStore) getGroupMembers(ctx context.Context, groupID string) ([]models.GroupMember, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name, user_id FROM group_members WHERE group_id = ? ORDER BY name",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var name string
		var userID sql.NullString
		if err := rows.Scan(&name, &userID); err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		m := models.GroupMember{DisplayName: name}
		if userID.Valid {
			m.UserID = userID.String
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
