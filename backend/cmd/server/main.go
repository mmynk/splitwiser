package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/mmynk/splitwiser/internal/auth"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/service"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	"github.com/mmynk/splitwiser/pkg/logging"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

const (
	jwtTokenDuration = 24 * time.Hour // Tokens valid for 24 hours
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// Setup colored structured logging (level from LOG_LEVEL env, default INFO)
	logging.Setup()
	logger := slog.Default()

	// Read configuration from environment
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-do-not-use-in-production")
	if jwtSecret == "dev-secret-do-not-use-in-production" {
		slog.Warn("JWT_SECRET not set - using insecure default. Set JWT_SECRET for production.")
	}

	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		slog.Error("Invalid PORT value", "port", portStr, "error", err)
		os.Exit(1)
	}

	corsOrigin := getEnv("CORS_ORIGIN", "*")
	if corsOrigin == "*" {
		slog.Warn("CORS_ORIGIN is set to wildcard '*'. Set CORS_ORIGIN to your domain for production.")
	}

	tlsCertFile := getEnv("TLS_CERT_FILE", "")
	tlsKeyFile := getEnv("TLS_KEY_FILE", "")

	dbPath := getEnv("DB_PATH", "./data/bills.db")
	staticPath := getEnv("STATIC_PATH", "../frontend/static")

	// Initialize SQLite storage
	store, err := sqlite.New(dbPath)
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("Storage initialized", "database", dbPath)

	// Initialize authentication components
	jwtManager := auth.NewJWTManager(jwtSecret, jwtTokenDuration)
	passwordAuth := auth.NewPasswordAuthenticator(store)

	// Create auth middleware
	authMiddleware := middleware.RequireAuth(jwtManager)

	// Create logging interceptor (runs before auth to capture all errors)
	loggingInterceptor := middleware.LoggingInterceptor()

	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register AuthService (logging only, no auth required)
	authPath, authHandler := protoconnect.NewAuthServiceHandler(
		service.NewAuthService(passwordAuth, jwtManager, logger),
		connect.WithInterceptors(loggingInterceptor),
	)
	mux.Handle(authPath, authHandler)

	// Register protected services with logging + auth middleware
	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(
		service.NewSplitService(store),
		connect.WithInterceptors(loggingInterceptor, authMiddleware),
	)
	mux.Handle(splitPath, splitHandler)

	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(
		service.NewGroupService(store),
		connect.WithInterceptors(loggingInterceptor, authMiddleware),
	)
	mux.Handle(groupPath, groupHandler)

	// Serve static files from frontend/static
	staticDir, err := filepath.Abs(staticPath)
	if err != nil {
		slog.Error("Failed to resolve static path", "error", err)
		os.Exit(1)
	}
	slog.Info("Serving static files", "path", staticDir)

	// Handle all non-API routes with static file server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an API request (Connect RPC)
		if strings.HasPrefix(r.URL.Path, "/splitwiser.v1.") {
			http.NotFound(w, r)
			return
		}

		// Serve static files
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}

		filePath := filepath.Join(staticDir, filepath.Clean(urlPath))

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// For SPA-like behavior, serve index.html for unknown paths
			// But for bill.html, we use query params so this isn't needed
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}

		http.ServeFile(w, r, filePath)
	})

	// Add CORS middleware
	handler := corsMiddleware(mux, corsOrigin)

	addr := fmt.Sprintf(":%d", port)

	// TLS mode: both cert and key must be set (or neither)
	if tlsCertFile != "" && tlsKeyFile != "" {
		// TLS negotiates HTTP/2 natively via ALPN — no h2c wrapper needed
		server := &http.Server{
			Addr:    addr,
			Handler: handler,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		slog.Info("Connect server starting with TLS", "address", addr, "url", fmt.Sprintf("https://localhost%s", addr))
		if err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	} else if tlsCertFile != "" || tlsKeyFile != "" {
		slog.Error("Both TLS_CERT_FILE and TLS_KEY_FILE must be set (or neither)")
		os.Exit(1)
	} else {
		// No TLS — use h2c for HTTP/2 without TLS (local dev)
		h2cHandler := h2c.NewHandler(handler, &http2.Server{})
		slog.Info("Connect server starting", "address", addr, "url", fmt.Sprintf("http://localhost%s", addr))
		if err := http.ListenAndServe(addr, h2cHandler); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}
}

// corsMiddleware adds CORS headers for browser access
func corsMiddleware(next http.Handler, allowOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
