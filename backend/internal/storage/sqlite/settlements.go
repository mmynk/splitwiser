package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mmynk/splitwiser/internal/models"
)

// CreateSettlement persists a new settlement to the database.
func (s *SQLiteStore) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	if settlement.ID == "" {
		settlement.ID = uuid.New().String()
	}
	if settlement.CreatedAt == 0 {
		settlement.CreatedAt = time.Now().Unix()
	}

	var groupID interface{}
	if settlement.GroupID != nil {
		groupID = *settlement.GroupID
	}

	var note interface{}
	if settlement.Note != "" {
		note = settlement.Note
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settlements (id, group_id, from_user_id, to_user_id, amount, created_at, created_by, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		settlement.ID, groupID, settlement.FromUserID, settlement.ToUserID,
		settlement.Amount, settlement.CreatedAt, settlement.CreatedBy, note,
	)
	if err != nil {
		return fmt.Errorf("failed to insert settlement: %w", err)
	}

	return nil
}

// GetSettlement retrieves a settlement by ID.
func (s *SQLiteStore) GetSettlement(ctx context.Context, settlementID string) (*models.Settlement, error) {
	settlement := &models.Settlement{}
	var groupID sql.NullString
	var note sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, group_id, from_user_id, to_user_id, amount, created_at, created_by, note
		 FROM settlements WHERE id = ?`,
		settlementID,
	).Scan(&settlement.ID, &groupID, &settlement.FromUserID, &settlement.ToUserID,
		&settlement.Amount, &settlement.CreatedAt, &settlement.CreatedBy, &note)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	if groupID.Valid {
		settlement.GroupID = &groupID.String
	}
	if note.Valid {
		settlement.Note = note.String
	}

	return settlement, nil
}

// ListSettlementsByGroup retrieves all settlements for a group.
func (s *SQLiteStore) ListSettlementsByGroup(ctx context.Context, groupID string) ([]*models.Settlement, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, group_id, from_user_id, to_user_id, amount, created_at, created_by, note
		 FROM settlements WHERE group_id = ? ORDER BY created_at DESC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements by group: %w", err)
	}
	defer rows.Close()

	return scanSettlements(rows)
}

// ListDirectSettlementsByUser retrieves all settlements with no group (cross-group settle ups)
// involving the given display name as either payer or payee.
func (s *SQLiteStore) ListDirectSettlementsByUser(ctx context.Context, displayName string) ([]*models.Settlement, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, group_id, from_user_id, to_user_id, amount, created_at, created_by, note
		 FROM settlements
		 WHERE group_id IS NULL AND (from_user_id = ? OR to_user_id = ?)
		 ORDER BY created_at DESC`,
		displayName, displayName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list direct settlements: %w", err)
	}
	defer rows.Close()

	return scanSettlements(rows)
}

// DeleteSettlement removes a settlement by ID.
func (s *SQLiteStore) DeleteSettlement(ctx context.Context, settlementID string) error {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM settlements WHERE id = ?", settlementID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("settlement not found: %s", settlementID)
	}
	if err != nil {
		return fmt.Errorf("failed to check settlement existence: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM settlements WHERE id = ?", settlementID)
	if err != nil {
		return fmt.Errorf("failed to delete settlement: %w", err)
	}

	return nil
}

func scanSettlements(rows *sql.Rows) ([]*models.Settlement, error) {
	var settlements []*models.Settlement
	for rows.Next() {
		settlement := &models.Settlement{}
		var groupID sql.NullString
		var note sql.NullString

		if err := rows.Scan(&settlement.ID, &groupID, &settlement.FromUserID, &settlement.ToUserID,
			&settlement.Amount, &settlement.CreatedAt, &settlement.CreatedBy, &note); err != nil {
			return nil, fmt.Errorf("failed to scan settlement: %w", err)
		}

		if groupID.Valid {
			settlement.GroupID = &groupID.String
		}
		if note.Valid {
			settlement.Note = note.String
		}

		settlements = append(settlements, settlement)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate settlements: %w", err)
	}

	return settlements, nil
}
