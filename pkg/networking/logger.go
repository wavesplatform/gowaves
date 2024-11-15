package networking

import (
	"context"
)

const Namespace = "NET"

type Logger interface {
	// Debug logs a message at the debug level.
	Debug(msg string, args ...any)

	// DebugContext logs a message at the debug level with access to the context's values
	DebugContext(ctx context.Context, msg string, args ...any)

	// Info logs a message at the info level.
	Info(msg string, args ...any)

	// InfoContext logs a message at the info level with access to the context's values
	InfoContext(ctx context.Context, msg string, args ...any)

	// Warn logs a message at the warn level.
	Warn(msg string, args ...any)

	// WarnContext logs a message at the warn level with access to the context's values
	WarnContext(ctx context.Context, msg string, args ...any)

	// Error logs a message at the error level.
	Error(msg string, args ...any)

	// ErrorContext logs a message at the error level with access to the context's values
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type wrappingLogger struct {
	logger     Logger
	attributes []any
}

func (l *wrappingLogger) Debug(msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.Debug(msg, args...)
}

func (l *wrappingLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.DebugContext(ctx, msg, args...)
}

func (l *wrappingLogger) Info(msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.Info(msg, args...)
}

func (l *wrappingLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *wrappingLogger) Warn(msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.Warn(msg, args...)
}

func (l *wrappingLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.WarnContext(ctx, msg, args...)
}

func (l *wrappingLogger) Error(msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.Error(msg, args...)
}

func (l *wrappingLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	args = append(args, l.attributes...)
	l.logger.ErrorContext(ctx, msg, args...)
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}

func (noopLogger) DebugContext(context.Context, string, ...any) {}

func (noopLogger) Info(string, ...any) {}

func (noopLogger) InfoContext(context.Context, string, ...any) {}

func (noopLogger) Warn(string, ...any) {}

func (noopLogger) WarnContext(context.Context, string, ...any) {}

func (noopLogger) Error(string, ...any) {}

func (noopLogger) ErrorContext(context.Context, string, ...any) {}
