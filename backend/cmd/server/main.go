package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/mmynk/splitwiser/internal/service"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

const (
	port = 8080
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Get paths from env or use defaults
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

	mux := http.NewServeMux()

	// Register Connect services
	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(service.NewSplitService(store))
	mux.Handle(splitPath, splitHandler)

	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(service.NewGroupService(store))
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

	// Add logging and CORS middleware
	loggedHandler := loggingMiddleware(corsMiddleware(mux))

	// Wrap with h2c for HTTP/2 without TLS (required for Connect)
	h2cHandler := h2c.NewHandler(loggedHandler, &http2.Server{})

	addr := fmt.Sprintf(":%d", port)
	slog.Info("Connect server starting", "address", addr, "url", fmt.Sprintf("http://localhost%s", addr))
	if err := http.ListenAndServe(addr, h2cHandler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		slog.Info("Request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		next.ServeHTTP(w, r)

		slog.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// corsMiddleware adds CORS headers for browser access
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms")
		w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
