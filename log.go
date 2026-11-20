package volcenginex

import "context"

// Logger is the small structured logging adapter used by the SDK.
// Implementations must be safe for concurrent use.
type Logger interface {
	Info(context.Context, string, ...any)
	Error(context.Context, string, ...any)
}

type discardLogger struct{}

func (discardLogger) Info(context.Context, string, ...any)  {}
func (discardLogger) Error(context.Context, string, ...any) {}

func loggerOrDiscard(l Logger) Logger {
	if l == nil {
		return discardLogger{}
	}
	return l
}
