package middleware

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// LoggingInterceptor returns a Connect interceptor that logs every RPC call.
// It logs the procedure name, user ID, duration, and any error codes/messages.
func LoggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure
			userID := GetUserID(ctx) // empty if pre-auth

			resp, err := next(ctx, req)

			duration := time.Since(start).Milliseconds()
			if err != nil {
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					slog.Warn("RPC error",
						"procedure", procedure,
						"code", connectErr.Code(),
						"error", connectErr.Message(),
						"user_id", userID,
						"duration_ms", duration,
					)
				} else {
					slog.Error("RPC error",
						"procedure", procedure,
						"error", err,
						"user_id", userID,
						"duration_ms", duration,
					)
				}
			} else {
				slog.Info("RPC ok",
					"procedure", procedure,
					"user_id", userID,
					"duration_ms", duration,
				)
			}

			return resp, err
		}
	}
}
