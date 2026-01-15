package service

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/auth"
	"github.com/mmynk/splitwiser/internal/middleware"
	proto "github.com/mmynk/splitwiser/pkg/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthService implements the AuthService RPC interface.
type AuthService struct {
	authenticator auth.Authenticator
	jwtManager    *auth.JWTManager
	logger        *slog.Logger
}

// NewAuthService creates a new authentication service.
func NewAuthService(authenticator auth.Authenticator, jwtManager *auth.JWTManager, logger *slog.Logger) *AuthService {
	return &AuthService{
		authenticator: authenticator,
		jwtManager:    jwtManager,
		logger:        logger,
	}
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, req *connect.Request[proto.RegisterRequest]) (*connect.Response[proto.RegisterResponse], error) {
	s.logger.Info("Register request", "email", req.Msg.Email)

	// Validate input
	if req.Msg.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, auth.ErrInvalidCredentials)
	}
	if req.Msg.DisplayName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, auth.ErrInvalidCredentials)
	}

	// Register user
	user, err := s.authenticator.Register(ctx, req.Msg.Email, req.Msg.DisplayName, req.Msg.Password)
	if err != nil {
		s.logger.Error("Registration failed", "email", req.Msg.Email, "error", err)
		if err == auth.ErrEmailExists {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		if err == auth.ErrWeakPassword {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "user_id", user.ID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Build response
	response := &proto.RegisterResponse{
		User: &proto.User{
			Id:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   timestamppb.New(time.Unix(user.CreatedAt, 0)),
		},
		Token: token,
	}

	s.logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	return connect.NewResponse(response), nil
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, req *connect.Request[proto.LoginRequest]) (*connect.Response[proto.LoginResponse], error) {
	s.logger.Info("Login request", "email", req.Msg.Email)

	// Validate input
	if req.Msg.Email == "" || req.Msg.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, auth.ErrInvalidCredentials)
	}

	// Authenticate user
	user, err := s.authenticator.Authenticate(ctx, req.Msg.Email, req.Msg.Password)
	if err != nil {
		s.logger.Warn("Login failed", "email", req.Msg.Email, "error", err)
		return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrInvalidCredentials)
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "user_id", user.ID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Build response
	response := &proto.LoginResponse{
		User: &proto.User{
			Id:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   timestamppb.New(time.Unix(user.CreatedAt, 0)),
		},
		Token: token,
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)
	return connect.NewResponse(response), nil
}

// Logout invalidates the user's session (currently a no-op since JWTs are stateless).
func (s *AuthService) Logout(ctx context.Context, req *connect.Request[proto.LogoutRequest]) (*connect.Response[proto.LogoutResponse], error) {
	// With stateless JWTs, logout is handled client-side by discarding the token.
	// For stateful sessions or token blacklisting, implement here.
	s.logger.Info("Logout request")
	return connect.NewResponse(&proto.LogoutResponse{}), nil
}

// GetCurrentUser returns the currently authenticated user's information.
func (s *AuthService) GetCurrentUser(ctx context.Context, req *connect.Request[proto.GetCurrentUserRequest]) (*connect.Response[proto.GetCurrentUserResponse], error) {
	// Get user ID from context (set by auth middleware)
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrMissingToken)
	}

	// Fetch user from storage
	// Note: We need to add GetUserByID to the authenticator or access storage directly
	// For now, we can just return the basic info from the JWT claims
	email := middleware.GetEmail(ctx)

	s.logger.Info("GetCurrentUser request", "user_id", userID)

	// TODO: Fetch full user details from storage
	// For now, return what we have from JWT claims
	response := &proto.GetCurrentUserResponse{
		User: &proto.User{
			Id:    userID,
			Email: email,
			// DisplayName and CreatedAt would need to be fetched from DB
		},
	}

	return connect.NewResponse(response), nil
}
