package networking

import (
	"log/slog"
	"time"
)

const (
	defaultKeepAliveInterval      = 1 * time.Minute
	defaultConnectionWriteTimeout = 15 * time.Second
)

// Config allows to set some parameters of the [Conn] or it's underlying connection.
type Config struct {
	slogHandler            slog.Handler
	protocol               Protocol
	handler                Handler
	keepAlive              bool
	keepAliveInterval      time.Duration
	connectionWriteTimeout time.Duration
	attributes             []any
}

// NewConfig creates a new Config and sets required Protocol and Handler parameters.
// Other parameters are set to their default values.
func NewConfig(p Protocol, h Handler) *Config {
	return &Config{
		protocol:               p,
		handler:                h,
		keepAlive:              true,
		keepAliveInterval:      defaultKeepAliveInterval,
		connectionWriteTimeout: defaultConnectionWriteTimeout,
		attributes:             nil,
	}
}

// WithSlogHandler sets the slog handler.
func (c *Config) WithSlogHandler(handler slog.Handler) *Config {
	c.slogHandler = handler
	return c
}

// WithWriteTimeout sets connection write timeout attribute to the Config.
func (c *Config) WithWriteTimeout(timeout time.Duration) *Config {
	c.connectionWriteTimeout = timeout
	return c
}

// WithSlogAttribute adds an attribute to the slice of attributes.
func (c *Config) WithSlogAttribute(attr slog.Attr) *Config {
	c.attributes = append(c.attributes, attr)
	return c
}

// WithSlogAttributes adds given attributes to the slice of attributes.
func (c *Config) WithSlogAttributes(attrs ...slog.Attr) *Config {
	for _, attr := range attrs {
		c.attributes = append(c.attributes, attr)
	}
	return c
}

func (c *Config) WithKeepAliveDisabled() *Config {
	c.keepAlive = false
	return c
}

func (c *Config) WithKeepAliveInterval(interval time.Duration) *Config {
	c.keepAliveInterval = interval
	return c
}
