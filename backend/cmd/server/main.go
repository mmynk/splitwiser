package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/mmynk/splitwiser/internal/service"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

const (
	port = 8080
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// Register Connect service
	path, handler := protoconnect.NewSplitServiceHandler(service.NewSplitService())
	mux.Handle(path, handler)

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
