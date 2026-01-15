package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mmynk/splitwiser/internal/models"
)

// CreateUser inserts a new user into the database.
func (s *SQLiteStore) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.DisplayName,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by their email address.
func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetUsersByIDs retrieves multiple users by their IDs.
// Returns a map of user ID to User object.
// Users that don't exist are omitted from the result.
func (s *SQLiteStore) GetUsersByIDs(ctx context.Context, ids []string) (map[string]*models.User, error) {
	if len(ids) == 0 {
		return make(map[string]*models.User), nil
	}

	// Build the IN clause with placeholders
	query := `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users
		WHERE id IN (?` + repeatPlaceholder(len(ids)-1) + `)`

	// Convert ids to []interface{} for ExecContext
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}
	defer rows.Close()

	users := make(map[string]*models.User)
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.DisplayName,
			&user.PasswordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users[user.ID] = user
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// repeatPlaceholder returns a string of ", ?" repeated n times.
// Used for building IN clauses with multiple placeholders.
func repeatPlaceholder(n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += ", ?"
	}
	return result
}
