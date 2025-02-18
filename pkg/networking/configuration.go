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
type Config[HS Handshake, H Header] struct {
	slogHandler            slog.Handler
	protocol               Protocol[HS, H]
	handler                Handler[HS, H]
	keepAlive              bool
	keepAliveInterval      time.Duration
	connectionWriteTimeout time.Duration
	attributes             []any
}

// NewConfig creates a new Config and sets required Protocol and Handler parameters.
// Other parameters are set to their default values.
func NewConfig[HS Handshake, H Header](p Protocol[HS, H], h Handler[HS, H]) *Config[HS, H] {
	return &Config[HS, H]{
		protocol:               p,
		handler:                h,
		keepAlive:              true,
		keepAliveInterval:      defaultKeepAliveInterval,
		connectionWriteTimeout: defaultConnectionWriteTimeout,
		attributes:             nil,
	}
}

// WithSlogHandler sets the slog handler.
func (c *Config[HS, H]) WithSlogHandler(handler slog.Handler) *Config[HS, H] {
	c.slogHandler = handler
	return c
}

// WithWriteTimeout sets connection write timeout attribute to the Config.
func (c *Config[HS, H]) WithWriteTimeout(timeout time.Duration) *Config[HS, H] {
	c.connectionWriteTimeout = timeout
	return c
}

// WithSlogAttribute adds an attribute to the slice of attributes.
func (c *Config[HS, H]) WithSlogAttribute(attr slog.Attr) *Config[HS, H] {
	c.attributes = append(c.attributes, attr)
	return c
}

// WithSlogAttributes adds given attributes to the slice of attributes.
func (c *Config[HS, H]) WithSlogAttributes(attrs ...slog.Attr) *Config[HS, H] {
	for _, attr := range attrs {
		c.attributes = append(c.attributes, attr)
	}
	return c
}

func (c *Config[HS, H]) WithKeepAliveDisabled() *Config[HS, H] {
	c.keepAlive = false
	return c
}

func (c *Config[HS, H]) WithKeepAliveInterval(interval time.Duration) *Config[HS, H] {
	c.keepAliveInterval = interval
	return c
}
