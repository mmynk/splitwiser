package models

// User represents a registered user account.
//
// NOTE: This model is for FUTURE use when authentication is added.
// The MVP does not use user accounts - participants are identified by name strings only.
//
// Future features:
//   - User login/registration
//   - Bill history per user
//   - User can belong to recurring groups
//   - Payment tracking between users
type User struct {
	// ID is the unique identifier for the user (UUID format).
	ID string

	// Name is the display name of the user.
	Name string

	// Email is the user's email address (unique).
	// Used for login and notifications.
	Email string

	// CreatedAt is the Unix timestamp when the user account was created.
	CreatedAt int64

	// Future fields to consider:
	// - PasswordHash string (for auth)
	// - Phone string (for SMS notifications)
	// - Avatar string (profile picture URL)
	// - Preferences map[string]string (user settings)
}
