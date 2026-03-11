package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mmynk/splitwiser/internal/models"
)

// SendFriendRequest persists a new friendship request.
// Returns an error if a request already exists in either direction.
func (s *SQLiteStore) SendFriendRequest(ctx context.Context, friendship *models.Friendship) error {
	if friendship.ID == "" {
		friendship.ID = uuid.New().String()
	}
	now := time.Now().Unix()
	if friendship.CreatedAt == 0 {
		friendship.CreatedAt = now
	}
	if friendship.UpdatedAt == 0 {
		friendship.UpdatedAt = now
	}

	// Check for an existing request in either direction before inserting.
	var existing string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM friendships
		WHERE (requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)`,
		friendship.RequesterID, friendship.AddresseeID,
		friendship.AddresseeID, friendship.RequesterID,
	).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing friendship: %w", err)
	}
	if err == nil {
		return fmt.Errorf("friendship request already exists between these users")
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO friendships (id, requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		friendship.ID, friendship.RequesterID, friendship.AddresseeID,
		string(friendship.Status), friendship.CreatedAt, friendship.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert friendship: %w", err)
	}

	return nil
}

// GetFriendship retrieves a friendship by ID.
func (s *SQLiteStore) GetFriendship(ctx context.Context, id string) (*models.Friendship, error) {
	f := &models.Friendship{}
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM friendships WHERE id = ?`,
		id,
	).Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &status, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("friendship not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get friendship: %w", err)
	}
	f.Status = models.FriendshipStatus(status)
	return f, nil
}

// UpdateFriendshipStatus updates the status of a friendship by ID.
func (s *SQLiteStore) UpdateFriendshipStatus(ctx context.Context, id string, status models.FriendshipStatus) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE friendships SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update friendship status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("friendship not found: %s", id)
	}
	return nil
}

// ListFriendships lists friendships for a user filtered by direction and status.
func (s *SQLiteStore) ListFriendships(ctx context.Context, userID string, incoming bool, status models.FriendshipStatus) ([]*models.Friendship, error) {
	var col string
	if incoming {
		col = "addressee_id"
	} else {
		col = "requester_id"
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM friendships WHERE `+col+` = ? AND status = ?
		ORDER BY created_at DESC`,
		userID, string(status),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list friendships: %w", err)
	}
	defer rows.Close()

	var friendships []*models.Friendship
	for rows.Next() {
		f := &models.Friendship{}
		var s string
		if err := rows.Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &s, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan friendship: %w", err)
		}
		f.Status = models.FriendshipStatus(s)
		friendships = append(friendships, f)
	}
	return friendships, rows.Err()
}

// DeleteFriendship removes a friendship by ID.
func (s *SQLiteStore) DeleteFriendship(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM friendships WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete friendship: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("friendship not found: %s", id)
	}
	return nil
}

// GetFriendshipBetween retrieves the friendship between two users in either direction.
// Returns a not-found error if no row exists.
func (s *SQLiteStore) GetFriendshipBetween(ctx context.Context, userIDA, userIDB string) (*models.Friendship, error) {
	f := &models.Friendship{}
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM friendships
		WHERE (requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)`,
		userIDA, userIDB, userIDB, userIDA,
	).Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &status, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("friendship not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get friendship: %w", err)
	}
	f.Status = models.FriendshipStatus(status)
	return f, nil
}

// AreFriends returns true if the two users have an accepted friendship in either direction.
func (s *SQLiteStore) AreFriends(ctx context.Context, userIDA, userIDB string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM friendships
		WHERE status = 'accepted'
		  AND ((requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?))`,
		userIDA, userIDB, userIDB, userIDA,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check friendship: %w", err)
	}
	return count > 0, nil
}

// SearchFriends finds accepted friends matching a partial display_name query.
func (s *SQLiteStore) SearchFriends(ctx context.Context, callerID string, query string) ([]*models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.display_name
		FROM users u
		JOIN friendships f ON (
			(f.requester_id = ? AND f.addressee_id = u.id) OR
			(f.addressee_id = ? AND f.requester_id = u.id)
		)
		WHERE f.status = 'accepted'
		  AND u.display_name LIKE ?
		LIMIT 10`,
		callerID, callerID, "%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search friends: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.DisplayName); err != nil {
			return nil, fmt.Errorf("failed to scan friend: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetFriends returns all accepted friends of a user as User objects.
func (s *SQLiteStore) GetFriends(ctx context.Context, userID string) ([]*models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.display_name
		FROM friendships f
		JOIN users u ON (f.requester_id = ? AND f.addressee_id = u.id)
		           OR  (f.addressee_id = ? AND f.requester_id = u.id)
		WHERE f.status = 'accepted'
		ORDER BY u.display_name`,
		userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get friends: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName); err != nil {
			return nil, fmt.Errorf("failed to scan friend: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
