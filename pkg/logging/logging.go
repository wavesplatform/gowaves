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
	switch params.Type {
	case LoggerText:
		return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: params.Level})
	case LoggerJSON:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: params.Level})
	case LoggerPretty:
		w := os.Stdout
		colorize := isatty.IsTerminal(w.Fd())
		return buildPrettyHandler(w, params.Level, colorize)
	case LoggerPrettyNoColor:
		return buildPrettyHandler(os.Stdout, params.Level, false)
	default:
		panic(fmt.Sprintf("unsupported logger type %d", params.Type))
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
