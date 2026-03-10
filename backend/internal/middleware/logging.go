package middleware

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	rpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "splitwiser_requests_total",
		Help: "Total number of RPC requests by procedure.",
	}, []string{"procedure"})

	rpcErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "splitwiser_errors_total",
		Help: "Total number of RPC errors by procedure and connect error code.",
	}, []string{"procedure", "code"})
)

// LoggingInterceptor returns a Connect interceptor that logs every RPC call
// and increments Prometheus counters for request and error rates.
func LoggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure
			userID := GetUserID(ctx) // empty if pre-auth

			resp, err := next(ctx, req)

			duration := time.Since(start).Milliseconds()
			rpcRequestsTotal.WithLabelValues(procedure).Inc()

			if err != nil {
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					rpcErrorsTotal.WithLabelValues(procedure, connectErr.Code().String()).Inc()
					slog.Warn("RPC error",
						"procedure", procedure,
						"code", connectErr.Code(),
						"error", connectErr.Message(),
						"user_id", userID,
						"duration_ms", duration,
					)
				} else {
					rpcErrorsTotal.WithLabelValues(procedure, "unknown").Inc()
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
