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
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

const NamespaceKey = "namespace"

const errorKey = "error"

// DefaultHandler creates a new slog handler with the specified parameters.
func DefaultHandler(params Parameters) slog.Handler {
	return NewHandler(params.Type, params.Level)
}

// NewHandler creates a new slog handler based on the specified logger type and level.
func NewHandler(loggerType LoggerType, level slog.Level) slog.Handler {
	return newHandler(loggerType, level, os.Stdout)
}

func newHandler(loggerType LoggerType, level slog.Level, w io.Writer) slog.Handler {
	switch loggerType {
	case LoggerText:
		return newTraceHandler(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}), false)
	case LoggerJSON:
		return newTraceHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}), false)
	case LoggerPretty:
		return newTraceHandler(buildPrettyHandler(w, level, isColorized(w), false), false)
	case LoggerPrettyNoColor:
		return newTraceHandler(buildPrettyHandler(w, level, false, false), false)
	case LoggerTextDev:
		return newTraceHandler(slog.NewTextHandler(w, &slog.HandlerOptions{AddSource: true, Level: level}), true)
	case LoggerJSONDev:
		return newTraceHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true, Level: level}), true)
	case LoggerPrettyDev:
		return newTraceHandler(buildPrettyHandler(w, level, isColorized(w), true), true)
	case LoggerPrettyNoColorDev:
		return newTraceHandler(buildPrettyHandler(w, level, false, true), true)
	default:
		panic(fmt.Sprintf("unsupported logger type %d", loggerType))
	}
}

func isColorized(w io.Writer) bool {
	// Check if the writer is a terminal and supports colorization.
	type fd interface{ Fd() uintptr }
	if f, ok := w.(fd); ok {
		return isatty.IsTerminal(f.Fd())
	}
	return false
}

func buildPrettyHandler(w io.Writer, level slog.Level, colorize, addSource bool) slog.Handler {
	tintHandler := tint.NewHandler(w, &tint.Options{
		Level:      level,
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
		NoColor:    !colorize,
		AddSource:  addSource,
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
	if !h.Enabled(ctx, r.Level) {
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
		textMarshaler(msgKey, errTextMarshaler{e.err}),
	}
	if e.opts != nil && e.opts.trace {
		if st, ok := e.err.(stackTracer); ok {
			attrs[1] = slog.Any(traceKey, st.StackTrace())
		}
	}
	return slog.GroupValue(attrs[:]...)
}

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

type scheme = byte

type txIDGetter interface {
	GetID(scheme scheme) (id []byte, err error)
}

type txIDSlogValuer struct {
	t      txIDGetter
	scheme scheme
}

func (v txIDSlogValuer) LogValue() slog.Value {
	id, err := v.t.GetID(v.scheme)
	if err != nil {
		return slog.GroupValue(slog.Group("txGetID", Error(err)))
	}
	return slog.StringValue(base58.Encode(id))
}

func TxID(t txIDGetter, scheme scheme) slog.Attr {
	var val slog.LogValuer = txIDSlogValuer{t: t, scheme: scheme}
	return slog.Any("txID", val)
}
