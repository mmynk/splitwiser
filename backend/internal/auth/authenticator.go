package auth

import (
	"context"

	"github.com/mmynk/splitwiser/internal/models"
)

// Authenticator defines the interface for authentication implementations.
// This abstraction allows swapping between different auth methods (password, passkeys, OAuth, etc.)
// without changing the service layer code.
type Authenticator interface {
	// Register creates a new user account with the given email and credential.
	// The credential format depends on the implementation (e.g., password, OAuth token, etc.)
	// Returns the created user or an error if registration fails.
	Register(ctx context.Context, email, displayName, credential string) (*models.User, error)

	// Authenticate verifies the user's credentials and returns the user if successful.
	// Returns an error if authentication fails.
	Authenticate(ctx context.Context, email, credential string) (*models.User, error)

	// ValidateCredential checks if the credential meets the implementation's requirements.
	// For passwords: check length, complexity, etc.
	// For other methods: validate format, etc.
	ValidateCredential(credential string) error
}
