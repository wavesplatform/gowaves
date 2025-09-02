package logging

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func traceJSON(t testing.TB, st []errors.Frame) []any {
	r := make([]any, len(st))
	for i, f := range st {
		fb, err := f.MarshalText()
		require.NoError(t, err)
		r[i] = string(fb)
	}
	return r
}

func TestError(t *testing.T) {
	e1 := stderrors.New("standard error")
	e2 := fmt.Errorf("wrapped error: %w", e1)
	e3 := errors.New("pkg errors error")
	st3, ok := e3.(stackTracer)
	require.True(t, ok)
	e4 := errors.Wrapf(e3, "wrapped pkg error")
	st4, ok := e4.(stackTracer)
	require.True(t, ok)
	for i, test := range []struct {
		err      error
		expected map[string]any
		dev      bool
	}{
		{nil, map[string]any{
			"test": "attribute",
		}, false},
		{nil, map[string]any{
			"test": "attribute",
		}, true},
		{e1, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "standard error"},
		}, false},
		{e1, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "standard error"},
		}, true},
		{e2, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "wrapped error: standard error"},
		}, false},
		{e2, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "wrapped error: standard error"},
		}, true},
		{e3, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "pkg errors error"},
		}, false},
		{e3, map[string]any{
			"test": "attribute",
			"error": map[string]any{
				"message": "pkg errors error",
				"trace":   traceJSON(t, st3.StackTrace()),
			},
		}, true},
		{e4, map[string]any{
			"test":  "attribute",
			"error": map[string]any{"message": "wrapped pkg error: pkg errors error"},
		}, false},
		{e4, map[string]any{
			"test": "attribute",
			"error": map[string]any{
				"message": "wrapped pkg error: pkg errors error",
				"trace":   traceJSON(t, st4.StackTrace()),
			},
		}, true},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			var buf bytes.Buffer
			var h = newHandler(LoggerJSON, slog.LevelDebug, &buf)
			if test.dev {
				h = newHandler(LoggerJSONDev, slog.LevelDebug, &buf)
			}
			logger := slog.New(h)
			logger.Error("Test error", slog.String("test", "attribute"), Error(test.err))
			var actual map[string]any
			str := buf.String()
			t.Log(str)
			err := json.Unmarshal([]byte(str), &actual)
			require.NoError(t, err)
			for k, v := range test.expected {
				assert.Contains(t, actual, k, "missing key %q in log output", k)
				if v != "" {
					assert.Equal(t, v, actual[k], "unexpected value for key %q", k)
				}
			}
		})
	}
}
