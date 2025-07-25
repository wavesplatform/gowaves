package logging

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/neilotoole/slogt"
	"github.com/pkg/errors"
	slogmock "github.com/samber/slog-mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	log := slogt.New(t, slogt.JSON())

	e1 := stderrors.New("standard error")
	e2 := fmt.Errorf("wrapped error: %w", e1)
	e3 := errors.New("pkg errors error")
	st3, ok := e3.(stackTracer)
	require.True(t, ok)
	e4 := errors.Wrapf(e3, "wrapped pkg error")
	st4, ok := e4.(stackTracer)
	require.True(t, ok)
	for i, test := range []struct {
		err   error
		attrs map[string]slog.Attr
	}{
		{nil, map[string]slog.Attr{
			"test": slog.String("test", "attribute"),
			"":     slog.Any("", nil),
		}},
		{e1, map[string]slog.Attr{
			"test":  slog.String("test", "attribute"),
			"error": slog.String("error", "standard error"),
			"":      slog.Any("", nil),
		}},
		{e2, map[string]slog.Attr{
			"test":  slog.String("test", "attribute"),
			"error": slog.String("error", "wrapped error: standard error"),
			"":      slog.Any("", nil),
		}},
		{e3, map[string]slog.Attr{
			"test":  slog.String("test", "attribute"),
			"error": slog.String("error", "pkg errors error"),
			"trace": slog.Any("trace", fmt.Sprintf("%+v", st3.StackTrace())),
		}},
		{e4, map[string]slog.Attr{
			"test":  slog.String("test", "attribute"),
			"error": slog.String("error", "wrapped pkg error: pkg errors error"),
			"trace": slog.Any("trace", fmt.Sprintf("%+v", st4.StackTrace())),
		}},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			mh := slogmock.Option{
				Enabled: func(_ context.Context, _ slog.Level) bool {
					return true
				},
				Handle: func(_ context.Context, record slog.Record) error {
					assert.Equal(t, slog.LevelError, record.Level, "expected error level")
					assert.Equal(t, "Test error", record.Message, "expected error msg")
					record.Attrs(func(attr slog.Attr) bool {
						exp, aok := test.attrs[attr.Key]
						assert.True(t, aok, "unexpected attribute key: %s", attr.Key)
						assert.Equal(t, exp.Value.Any(), attr.Value.Any(),
							"unexpected attribute value for key: %s", attr.Key)
						return true
					})
					return nil
				},
			}.NewMockHandler()
			logger := slog.New(mh)
			logger.Error("Test error",
				slog.String("test", "attribute"), Error(test.err), ErrorTrace(test.err))
			log.Info("Test error",
				slog.String("test", "attribute"), Error(test.err), ErrorTrace(test.err))
		})
	}
}
