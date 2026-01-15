package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/mmynk/splitwiser/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrEmailExists        = errors.New("email already registered")
)

// UserStorage defines the interface for user persistence operations.
// This allows the authenticator to be independent of the storage implementation.
type UserStorage interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
}

// PasswordAuthenticator implements password-based authentication using bcrypt.
type PasswordAuthenticator struct {
	storage UserStorage
}

// NewPasswordAuthenticator creates a new password-based authenticator.
func NewPasswordAuthenticator(storage UserStorage) *PasswordAuthenticator {
	return &PasswordAuthenticator{
		storage: storage,
	}
}

// ValidateCredential checks if the password meets minimum requirements.
func (a *PasswordAuthenticator) ValidateCredential(credential string) error {
	if len(credential) < 8 {
		return ErrWeakPassword
	}
	return nil
}

// Register creates a new user account with a hashed password.
func (a *PasswordAuthenticator) Register(ctx context.Context, email, displayName, credential string) (*models.User, error) {
	// Validate password strength
	if err := a.ValidateCredential(credential); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, err := a.storage.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailExists
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(credential), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user model
	user := models.NewUser(email, displayName, string(hashedPassword))

	// Save to storage
	if err := a.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Authenticate verifies the email and password, returning the user if valid.
func (a *PasswordAuthenticator) Authenticate(ctx context.Context, email, credential string) (*models.User, error) {
	// Get user by email
	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(credential)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
