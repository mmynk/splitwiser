package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/auth"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// setupAuthTestServer creates a test server with a real SQLite DB and JWT auth.
// The AuthService is registered with OptionalAuth so Register/Login work without
// a token, while GetCurrentUser works when a valid Bearer token is provided.
func setupAuthTestServer(t *testing.T) (protoconnect.AuthServiceClient, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-auth-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	store, err := sqlite.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create store: %v", err)
	}

	jwtManager := auth.NewJWTManager("test-secret-key-for-tests", 24*time.Hour)
	passwordAuth := auth.NewPasswordAuthenticator(store)
	authSvc := NewAuthService(passwordAuth, jwtManager, slog.Default())

	authPath, authHandler := protoconnect.NewAuthServiceHandler(
		authSvc,
		connect.WithInterceptors(middleware.OptionalAuth(jwtManager)),
	)

	mux := http.NewServeMux()
	mux.Handle(authPath, authHandler)
	server := httptest.NewServer(mux)

	client := protoconnect.NewAuthServiceClient(http.DefaultClient, server.URL)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return client, cleanup
}

func TestGetCurrentUser_ReturnsFullUserDetails(t *testing.T) {
	client, cleanup := setupAuthTestServer(t)
	defer cleanup()

	// Register a user to get a token
	registerResp, err := client.Register(context.Background(), connect.NewRequest(&pb.RegisterRequest{
		Email:       "test@example.com",
		DisplayName: "Test User",
		Password:    "password123",
	}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	token := registerResp.Msg.Token

	// Call GetCurrentUser with the Bearer token
	req := connect.NewRequest(&pb.GetCurrentUserRequest{})
	req.Header().Set("Authorization", "Bearer "+token)

	resp, err := client.GetCurrentUser(context.Background(), req)
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}

	user := resp.Msg.User
	if user.Email != "test@example.com" {
		t.Errorf("expected email %q, got %q", "test@example.com", user.Email)
	}
	if user.DisplayName != "Test User" {
		t.Errorf("expected display_name %q, got %q", "Test User", user.DisplayName)
	}
	if user.CreatedAt == nil {
		t.Error("expected created_at to be set, got nil")
	}
	if user.Id == "" {
		t.Error("expected user ID to be set")
	}
}

func TestGetCurrentUser_RequiresAuth(t *testing.T) {
	client, cleanup := setupAuthTestServer(t)
	defer cleanup()

	// Call GetCurrentUser without any token
	_, err := client.GetCurrentUser(context.Background(), connect.NewRequest(&pb.GetCurrentUserRequest{}))
	if err == nil {
		t.Fatal("expected error when calling GetCurrentUser without auth, got nil")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated, got %v", connectErr.Code())
	}
}
