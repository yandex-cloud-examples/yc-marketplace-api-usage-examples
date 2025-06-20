package logging

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// LoggerKey is the context key for the logger.
	LoggerKey = contextKey("logger")
	// RequestIDKey is the context key for the request ID string within the logger.
	RequestIDKey = contextKey("requestID")
)

// LoggingMiddleware injects a logger with a request ID into the context.
// It uses chi's middleware.GetReqID to retrieve the request ID.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context()) // Use chi's GetReqID
			ctxLogger := logger.With(slog.String(string(RequestIDKey), reqID))
			ctx := context.WithValue(r.Context(), LoggerKey, ctxLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLoggerFromContext retrieves the logger from the context.
// If no logger is found, it returns the default slog logger.
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	// Fallback to a default logger if not found.
	return slog.Default()
}
