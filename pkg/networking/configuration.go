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

// NewConfig creates a new Config and sets default keepAliveInterval and connectionWriteTimeout.
// KeepAlive is enabled by default.
// Protocol and Handler should be set explicitly.
func NewConfig() *Config {
	return &Config{
		keepAlive:              true,
		keepAliveInterval:      defaultKeepAliveInterval,
		connectionWriteTimeout: defaultConnectionWriteTimeout,
		attributes:             nil,
	}
}

// WithProtocol sets the protocol.
func (c *Config) WithProtocol(p Protocol) *Config {
	c.protocol = p
	return c
}

// WithHandler sets the handler.
func (c *Config) WithHandler(h Handler) *Config {
	c.handler = h
	return c
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
