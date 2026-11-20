package trace

import (
	"context"
	"uuid"
)

type requestIDKey struct{}

// NewRequestID returns a cryptographically random request ID.
func NewRequestID() string { return uuid.New().String() }

// WithRequestID attaches an SDK request ID to a context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID extracts the SDK request ID from a context.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
