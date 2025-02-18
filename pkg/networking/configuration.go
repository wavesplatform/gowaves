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
type Config[HS Handshake] struct {
	slogHandler            slog.Handler
	protocol               Protocol[HS]
	handler                Handler[HS]
	keepAlive              bool
	keepAliveInterval      time.Duration
	connectionWriteTimeout time.Duration
	attributes             []any
}

// NewConfig creates a new Config and sets required Protocol and Handler parameters.
// Other parameters are set to their default values.
func NewConfig[HS Handshake](p Protocol[HS], h Handler[HS]) *Config[HS] {
	return &Config[HS]{
		protocol:               p,
		handler:                h,
		keepAlive:              true,
		keepAliveInterval:      defaultKeepAliveInterval,
		connectionWriteTimeout: defaultConnectionWriteTimeout,
		attributes:             nil,
	}
}

// WithSlogHandler sets the slog handler.
func (c *Config[HS]) WithSlogHandler(handler slog.Handler) *Config[HS] {
	c.slogHandler = handler
	return c
}

// WithWriteTimeout sets connection write timeout attribute to the Config.
func (c *Config[HS]) WithWriteTimeout(timeout time.Duration) *Config[HS] {
	c.connectionWriteTimeout = timeout
	return c
}

// WithSlogAttribute adds an attribute to the slice of attributes.
func (c *Config[HS]) WithSlogAttribute(attr slog.Attr) *Config[HS] {
	c.attributes = append(c.attributes, attr)
	return c
}

// WithSlogAttributes adds given attributes to the slice of attributes.
func (c *Config[HS]) WithSlogAttributes(attrs ...slog.Attr) *Config[HS] {
	for _, attr := range attrs {
		c.attributes = append(c.attributes, attr)
	}
	return c
}

func (c *Config[HS]) WithKeepAliveDisabled() *Config[HS] {
	c.keepAlive = false
	return c
}

func (c *Config[HS]) WithKeepAliveInterval(interval time.Duration) *Config[HS] {
	c.keepAliveInterval = interval
	return c
}
