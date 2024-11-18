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
	logger                 *slog.Logger
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
		logger:                 slog.Default(),
		protocol:               p,
		handler:                h,
		keepAlive:              true,
		keepAliveInterval:      defaultKeepAliveInterval,
		connectionWriteTimeout: defaultConnectionWriteTimeout,
		attributes:             nil,
	}
}

// WithLogger sets the logger.
func (c *Config) WithLogger(logger *slog.Logger) *Config {
	c.logger = logger
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

func (c *Config) WithKeepAliveDisabled() *Config {
	c.keepAlive = false
	return c
}
