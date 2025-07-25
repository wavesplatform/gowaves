package logging

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dpotapov/slogpfx"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
)

const NamespaceKey = "namespace"

// DefaultHandler creates a new slog handler with the specified parameters.
func DefaultHandler(params Parameters) slog.Handler {
	return NewHandler(params.Type, params.Level)
}

// NewHandler creates a new slog handler based on the specified logger type and level.
func NewHandler(loggerType LoggerType, level slog.Level) (h slog.Handler) {
	return newHandler(loggerType, level, os.Stdout, true)
}

func newHandler(loggerType LoggerType, level slog.Level, w io.Writer, trace bool) (h slog.Handler) {
	defer func() { // TODO: temporary workaround
		h = newTraceHandler(h, trace)
	}()
	switch loggerType {
	case LoggerText:
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	case LoggerJSON:
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	case LoggerPretty:
		type fd interface{ Fd() uintptr }
		_ = fd(os.Stdout) // Ensure that os.Stdout implements fd interface
		colorize := false
		if f, ok := w.(fd); ok {
			colorize = isatty.IsTerminal(f.Fd())
		}
		return buildPrettyHandler(w, level, colorize)
	case LoggerPrettyNoColor:
		return buildPrettyHandler(w, level, false)
	default:
		panic(fmt.Sprintf("unsupported logger type %d", loggerType))
	}
}

func buildPrettyHandler(w io.Writer, level slog.Level, colorize bool) slog.Handler {
	tintHandler := tint.NewHandler(w, &tint.Options{
		Level:      level,
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
		NoColor:    !colorize,
	})
	formatter := slogpfx.DefaultPrefixFormatter
	if colorize {
		formatter = slogpfx.ColorizePrefix(formatter)
	}
	prefixed := slogpfx.NewHandler(tintHandler, &slogpfx.HandlerOptions{
		PrefixKeys:      []string{NamespaceKey},
		PrefixFormatter: formatter,
	})
	return prefixed
}

type attrVisitorHandler struct {
	slog.Handler
	attrVisitor func(a slog.Attr) bool
}

func (h *attrVisitorHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Handler.Enabled(ctx, r.Level) {
		return nil // skip handling if the level is not enabled
	}
	if h.attrVisitor != nil {
		r.Attrs(h.attrVisitor)
	}
	return h.Handler.Handle(ctx, r)
}

func newTraceHandler(h slog.Handler, trace bool) slog.Handler {
	// Wraps the handler to add a trace attribute if the error implements stackTracer.
	return &attrVisitorHandler{
		Handler: h,
		attrVisitor: func(a slog.Attr) bool {
			if a.Key != errorKey || a.Value.Kind() != slog.KindLogValuer {
				return true // continue processing other attributes
			}
			if elv, ok := a.Value.Any().(errorLogValuer); ok && elv.opts != nil {
				elv.opts.trace = trace
			}
			return true // continue processing other attributes
		},
	}
}

// textMarshaler is a helper function that formats a value as a slog.Attr
// and checks if the value implements encoding.TextMarshaler.
func textMarshaler(key string, value encoding.TextMarshaler) slog.Attr { return slog.Any(key, value) }

type typenamePrinter struct{ v any }

func (t typenamePrinter) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%T", t.v), nil
}

// Type returns a slog.Attr that contains the type name of the value.
func Type(value any) slog.Attr {
	const key = "type"
	return textMarshaler(key, typenamePrinter{v: value})
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type errTextMarshaler struct {
	err error
}

func (e errTextMarshaler) MarshalText() ([]byte, error) {
	return []byte(e.err.Error()), nil
}

type errorLogValuerOpts struct {
	trace bool
}

type errorLogValuer struct {
	err  error
	opts *errorLogValuerOpts
}

func (e errorLogValuer) LogValue() slog.Value {
	if e.err == nil {
		return slog.Value{} // Return an empty value if the error is nil
	}
	const (
		msgKey   = "message"
		traceKey = "trace"
	)
	attrs := [2]slog.Attr{
		slog.Any(msgKey, errTextMarshaler{e.err}),
	}
	if e.opts != nil && e.opts.trace {
		if st, ok := e.err.(stackTracer); ok {
			attrs[1] = slog.Any(traceKey, st.StackTrace())
		}
	}
	return slog.GroupValue(attrs[:]...)
}

const errorKey = "error"

func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	// check if the errorLogValuer interface is implemented
	var lvErr slog.LogValuer = errorLogValuer{
		err:  err,
		opts: new(errorLogValuerOpts),
	}
	return slog.Any(errorKey, lvErr)
}

func ErrorTrace(err error) slog.Attr {
	const key = "trace"
	if err == nil {
		return slog.Any("", nil)
	}
	if st, ok := err.(stackTracer); ok {
		return slog.Any(key, fmt.Sprintf("%+v", st.StackTrace()))
	}
	return slog.Any("", nil)
}
