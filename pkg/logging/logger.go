package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"moul.io/zapfilter"
)

type config struct {
	filter zapfilter.FilterFunc
	opts   []zap.Option
	ec     zapcore.EncoderConfig
}

func newConfig(opts []Option) *config {
	f, err := zapfilter.ByLevels("")
	if err != nil {
		panic(fmt.Sprintf("Impossible error: %v", err))
	}
	c := &config{
		filter: f,
		ec:     zap.NewDevelopmentEncoderConfig(),
	}
	for _, o := range opts {
		o.apply(c)
	}
	return c
}

func (c *config) logger(level zapcore.Level) *zap.Logger {
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(c.ec), zapcore.Lock(os.Stdout), level)
	logger := zap.New(zapfilter.NewFilteringCore(core, c.filter))
	zap.ReplaceGlobals(logger.WithOptions(c.opts...))

	return logger
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func NetworkFilter(flag bool) Option {
	return optionFunc(func(c *config) {
		if !flag {
			c.filter = zapfilter.All(c.filter, zapfilter.Reverse(zapfilter.ByNamespaces(NetworkNamespace)))
		}
	})
}

func NetworkDataFilter(flag bool) Option {
	return optionFunc(func(c *config) {
		if !flag {
			c.filter = zapfilter.All(c.filter, zapfilter.Reverse(zapfilter.ByNamespaces(NetworkDataNamespace)))
		}
	})
}

func FSMFilter(flag bool) Option {
	return optionFunc(func(c *config) {
		if !flag {
			c.filter = zapfilter.All(c.filter, zapfilter.Reverse(zapfilter.ByNamespaces(FSMNamespace)))
		}
	})
}

func DevelopmentFlag(flag bool) Option {
	return optionFunc(func(c *config) {
		if flag {
			c.opts = append(c.opts, zap.AddCaller())
		}
	})
}

func SetupSimpleLogger(level zapcore.Level) *zap.Logger {
	return SetupLogger(level)
}

func SetupLogger(level zapcore.Level, opts ...Option) *zap.Logger {
	c := newConfig(opts)
	logger := c.logger(level)
	zap.ReplaceGlobals(logger)
	return logger
}
