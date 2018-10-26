package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const retryTimeout = 30

// ConnOption is a connection creation option
type ConnOption func(*Conn) error

// Conn is a connection between two waves nodes
type Conn struct {
	net.Conn
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	network  string
	addr     string
	version  proto.Version
	ingress  chan<- interface{}
	outgress chan interface{}

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

func (c *Conn) reader() {
	bufConn := bufio.NewReader(c.Conn)
	defer c.wg.Done()

LOOP:
	for {
		buf, err := bufConn.Peek(9)
		if err != nil {
			zap.S().Error("error while reading from connection: ", err)
			break LOOP
		}

		switch msgType := buf[8]; msgType {
		case proto.ContentIDGetPeers:
			var gp proto.GetPeersMessage
			_, err := gp.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("error while receiving GetPeersMessage: ", err)
				break LOOP
			}
			c.ingress <- gp
		case proto.ContentIDPeers:
			var p proto.PeersMessage
			_, err := p.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("failed to read Peers message: ", err)
				break LOOP
			}
			var b []byte
			b, e := json.Marshal(p)
			if e != nil {
				return
			}
			js := string(b)
			zap.S().Info("Got peers", js)
			c.ingress <- p
		case proto.ContentIDScore:
			var s proto.ScoreMessage
			_, err := s.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("failed to read Score message: ", err)
				break LOOP
			}
		case proto.ContentIDSignatures:
			var m proto.SignaturesMessage
			_, err := m.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("failed to read Signatures message:", err)
				break LOOP
			}
			c.ingress <- m
		default:
			l := binary.BigEndian.Uint32(buf[:4])
			arr := make([]byte, l)
			_, err := io.ReadFull(bufConn, arr)
			if err != nil {
				zap.S().Error("failed to read default message: ", err)
				break LOOP
			}
			break
		}
	}
}

func (c *Conn) sendMessage(m interface{}) error {
	var err error
	switch v := m.(type) {
	case proto.GetPeersMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.PeersMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.GetSignaturesMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.SignaturesMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.GetBlockMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.BlockMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.ScoreMessage:
		_, err = v.WriteTo(c.Conn)
	case proto.TransactionMessage:
		_, err = v.WriteTo(c.Conn)
	}

	return err
}

func (c *Conn) writer() {
	defer c.wg.Done()
LOOP:
	for {
		select {
		case m := <-c.outgress:
			if err := c.sendMessage(m); err != nil {
				break LOOP
			}
		case <-c.ctx.Done():
			break LOOP
		}
	}
}

func (c *Conn) loop() {
	defer c.wg.Done()
	select {
	case <-c.ctx.Done():
	}
	c.Conn.Close()
}

// Run runs the connection
func (c *Conn) Run() {
	c.wg.Add(3)
	go c.reader()
	go c.writer()
	go c.loop()
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

// WithContext is an option to add a context to the connection
func WithContext(ctx context.Context) ConnOption {
	return func(c *Conn) error {
		c.ctx, c.cancel = context.WithCancel(ctx)
		return nil
	}
}

func WithIngress(ch chan<- interface{}) ConnOption {
	return func(c *Conn) error {
		c.ingress = ch
		return nil
	}
}
func (c *Conn) Send() chan<- interface{} {
	return c.outgress
}

func (c *Conn) Close() {
	c.cancel()
	c.wg.Wait()
}

// NewConn creates a new connection
func NewConn(options ...ConnOption) (*Conn, error) {
	c := Conn{}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.ingress = make(chan interface{}, 1024)
	c.outgress = make(chan interface{}, 1024)

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
