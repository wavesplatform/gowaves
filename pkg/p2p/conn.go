package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const retryTimeout = 30

// ConnOption is a connection creation option
type ConnOption func(*Conn) error

var ErrUnknownMessage = errors.New("unknown message")

// Conn is a connection between two waves nodes
type Conn struct {
	conn    net.Conn
	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	network string
	addr    string
	bufConn *bufio.Reader

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

	c.conn = conn
	c.bufConn = bufio.NewReaderSize(c.conn, 65535)

	return nil
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) Close() {
	c.conn.Close()
}

func (c *Conn) ReadMessage() (interface{}, error) {
	buf, err := c.bufConn.Peek(9)
	if err != nil {
		return nil, err
	}

	switch msgType := buf[8]; msgType {
	case proto.ContentIDGetPeers:
		var gp proto.GetPeersMessage
		_, err := gp.ReadFrom(c.bufConn)
		if err != nil {
			return nil, err
		}
		return gp, nil
	case proto.ContentIDPeers:
		var p proto.PeersMessage
		_, err := p.ReadFrom(c.bufConn)
		if err != nil {
			return nil, err
		}
		return p, nil
	case proto.ContentIDScore:
		var s proto.ScoreMessage
		_, err := s.ReadFrom(c.bufConn)
		if err != nil {
			return nil, err
		}
		return s, nil
	case proto.ContentIDSignatures:
		var m proto.SignaturesMessage
		_, err := m.ReadFrom(c.bufConn)
		if err != nil {
			return nil, err
		}
		return m, nil
	case proto.ContentIDBlock:
		var m proto.BlockMessage
		_, err := m.ReadFrom(c.bufConn)
		if err != nil {
			return nil, err
		}
		return m, nil
	default:
		var packetLen [4]byte
		_, err := io.ReadFull(c.bufConn, packetLen[:])
		l := binary.BigEndian.Uint32(packetLen[:])
		arr := make([]byte, l)
		_, err = io.ReadFull(c.bufConn, arr)
		if err != nil {
			return nil, err
		}
		return nil, ErrUnknownMessage
	}
}

// ReadWithDeadline reads a message with a deadline
func (c *Conn) ReadWithDeadline(deadLine time.Time) (interface{}, error) {
	c.conn.SetReadDeadline(deadLine)
	defer c.conn.SetReadDeadline(time.Time{})

	msg, err := c.ReadMessage()

	return msg, err
}

func (c *Conn) SendMessage(m interface{}) error {
	var err error
	switch v := m.(type) {
	case proto.GetPeersMessage:
		_, err = v.WriteTo(c.conn)
	case proto.PeersMessage:
		_, err = v.WriteTo(c.conn)
	case proto.GetSignaturesMessage:
		_, err = v.WriteTo(c.conn)
	case proto.SignaturesMessage:
		_, err = v.WriteTo(c.conn)
	case proto.GetBlockMessage:
		_, err = v.WriteTo(c.conn)
	case proto.BlockMessage:
		_, err = v.WriteTo(c.conn)
	case proto.ScoreMessage:
		_, err = v.WriteTo(c.conn)
	case proto.TransactionMessage:
		_, err = v.WriteTo(c.conn)
	}

	return err
}

// WithRemote is an option for remote endpoint
func WithRemote(network, addr string) ConnOption {
	return func(c *Conn) error {
		c.network = network
		c.addr = addr
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

// WithContext is an option to add a context to the connection
func WithContext(ctx context.Context) ConnOption {
	return func(c *Conn) error {
		c.ctx, c.cancel = context.WithCancel(ctx)
		return nil
	}
}

func WithNetConn(conn net.Conn) ConnOption {
	return func(c *Conn) error {
		c.conn = conn
		c.bufConn = bufio.NewReaderSize(c.conn, 65535)
		return nil
	}
}

// NewConn creates a new connection
func NewConn(options ...ConnOption) (*Conn, error) {
	c := Conn{}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	for _, option := range options {
		if err := option(&c); err != nil {
			return nil, err
		}
	}
	if c.Transport == nil {
		c.Transport = &DefaultTransport
	}

	c.timeout = time.Second * 3
	return &c, nil
}
