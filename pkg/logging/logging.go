package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dpotapov/slogpfx"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const NamespaceKey = "namespace"

// DefaultHandler creates a new slog handler with the specified parameters.
func DefaultHandler(params Parameters) slog.Handler {
	return NewHandler(params.Type, params.Level)
}

// NewHandler creates a new slog handler based on the specified logger type and level.
func NewHandler(loggerType LoggerType, level slog.Level) slog.Handler {
	switch loggerType {
	case LoggerText:
		return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	case LoggerJSON:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	case LoggerPretty:
		w := os.Stdout
		colorize := isatty.IsTerminal(w.Fd())
		return buildPrettyHandler(w, level, colorize)
	case LoggerPrettyNoColor:
		return buildPrettyHandler(os.Stdout, level, false)
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
