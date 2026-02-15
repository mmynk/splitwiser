package middleware

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/auth"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserIDKey is the context key for storing the authenticated user ID.
	UserIDKey contextKey = "user_id"
	// EmailKey is the context key for storing the authenticated user's email.
	EmailKey contextKey = "email"
)

// GetUserID extracts the user ID from the context.
// Returns empty string if not found.
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

// GetEmail extracts the user email from the context.
// Returns empty string if not found.
func GetEmail(ctx context.Context) string {
	email, _ := ctx.Value(EmailKey).(string)
	return email
}

// RequireAuth returns a middleware that validates JWT tokens and requires authentication.
// It extracts the token from the Authorization header, validates it, and adds
// the user ID and email to the request context.
func RequireAuth(jwtManager *auth.JWTManager) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract Authorization header
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrMissingToken)
			}

			// Parse Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrInvalidToken)
			}
			tokenString := parts[1]

			// Validate token
			claims, err := jwtManager.Validate(tokenString)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			// Add user info to context
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)

			// Call the next handler with enriched context
			return next(ctx, req)
		}
	}
}

// OptionalAuth returns a middleware that validates JWT tokens if present, but allows
// requests without authentication. Useful for endpoints that have different behavior
// for authenticated vs unauthenticated users.
func OptionalAuth(jwtManager *auth.JWTManager) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract Authorization header
			authHeader := req.Header().Get("Authorization")
			if authHeader != "" {
				// Parse Bearer token
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString := parts[1]

					// Validate token (ignore errors - optional auth)
					claims, err := jwtManager.Validate(tokenString)
					if err == nil {
						// Add user info to context only if valid
						ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
						ctx = context.WithValue(ctx, EmailKey, claims.Email)
					}
				}
			}

			// Call the next handler (with or without user context)
			return next(ctx, req)
		}
	}
}

