package p2p

import (
	"context"
	"net"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const retryTimeout = 30

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

func WithRemote(network, addr string) func(*Conn) {
	return func(c *Conn) {
		c.network = network
		c.addr = addr
	}
}

func WithVersion(v proto.Version) func(*Conn) {
	return func(c *Conn) {
		c.version = v
	}
}

func WithTransport(t *Transport) func(*Conn) {
	return func(c *Conn) {
		c.Transport = t
	}
}

// NewConn creates a new connection
func NewConn(options ...func(*Conn)) (*Conn, error) {
	c := Conn{}
	for _, option := range options {
		option(&c)
	}
	if c.Transport == nil {
		c.Transport = &DefaultTransport
	}

	return &c, nil
}
