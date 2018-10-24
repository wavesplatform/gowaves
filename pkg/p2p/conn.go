package p2p

import (
	"context"
	"net"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const retryTimeout = 30

// ConnOption is a connection creation option
type ConnOption func(*Conn) error

// Conn is a connection between two waves nodes
type Conn struct {
	net.Conn
	network string
	addr    string
	version proto.Version

	Transport *Transport
}

// Dial is a wrapper around DialContext
func (c *Conn) Dial(network, addr string) error {
	return c.DialContext(context.Background(), network, addr)
}

// DialContext dials a remote andpoint
func (c *Conn) DialContext(ctx context.Context, network, addr string) error {
	conn, err := c.Transport.DialContext(ctx, network, addr)
	if err != nil {
		return err
	}

	c.Conn = conn

	return nil
}

// WithRemote is an option for remote endpoint
func WithRemote(network, addr string) ConnOption {
	return func(c *Conn) error {
		c.network = network
		c.addr = addr
		return nil
	}
}

// WithVersion is an option for versioning a connection
func WithVersion(v proto.Version) ConnOption {
	return func(c *Conn) error {
		c.version = v
		return nil
	}
}

// WithTransport is an option for custom transport of the connection
func WithTransport(t *Transport) ConnOption {
	return func(c *Conn) error {
		c.Transport = t
		return nil
	}
}

// NewConn creates a new connection
func NewConn(options ...ConnOption) (*Conn, error) {
	c := Conn{}
	for _, option := range options {
		if err := option(&c); err != nil {
			return nil, err
		}
	}
	if c.Transport == nil {
		c.Transport = &DefaultTransport
	}

	return &c, nil
}
