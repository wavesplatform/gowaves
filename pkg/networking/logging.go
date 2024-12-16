package networking

import (
	"context"
	"log/slog"
)

// TODO: Remove this file and the handler when the default [slog.DiscardHandler] will be introduced in
//   Go version 1.24. See https://go-review.googlesource.com/c/go/+/626486.

// discardingHandler is a logger that discards all log messages.
// It is used when no slog handler is provided in the [Config].
type discardingHandler struct{}

func (h discardingHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (h discardingHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardingHandler) WithAttrs([]slog.Attr) slog.Handler        { return h }
func (h discardingHandler) WithGroup(string) slog.Handler             { return h }
