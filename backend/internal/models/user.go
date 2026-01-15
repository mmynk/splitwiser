package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user account.
type User struct {
	// ID is the unique identifier for the user (UUID format).
	ID string

	// Email is the user's email address (unique).
	// Used for login and notifications.
	Email string

	// DisplayName is the display name shown in the UI.
	DisplayName string

	// PasswordHash is the bcrypt hash of the user's password.
	// Nullable to support other auth methods (passkeys, OAuth, etc.)
	PasswordHash string

	// CreatedAt is the Unix timestamp when the user account was created.
	CreatedAt int64

	// UpdatedAt is the Unix timestamp when the user account was last updated.
	UpdatedAt int64
}

// NewUser creates a new User with a generated UUID and timestamps.
func NewUser(email, displayName, passwordHash string) *User {
	now := time.Now().Unix()
	return &User{
		ID:           uuid.New().String(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
