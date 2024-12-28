package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neilotoole/slogt"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/networking"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	appName        = "wavesL"
	nonce          = uint64(0)
	networkTimeout = 3 * time.Second
	pingInterval   = 5 * time.Second
)

type NetClient struct {
	ctx  context.Context
	t    testing.TB
	impl Implementation
	n    *networking.Network
	c    *networking.Config
	h    *handler
	s    *networking.Session

	closing atomic.Bool
	closed  sync.Once
}

func NewNetClient(
	ctx context.Context, t testing.TB, impl Implementation, port string, peers []proto.PeerInfo,
) *NetClient {
	n := networking.NewNetwork()
	p := newProtocol(t, nil)
	h := newHandler(t, peers)

	f := slogt.Factory(func(w io.Writer) slog.Handler {
		opts := &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}
		return slog.NewTextHandler(w, opts)
	})
	log := slogt.New(t, f)

	slog.SetLogLoggerLevel(slog.LevelError)
	conf := networking.NewConfig(p, h).
		WithSlogHandler(log.Handler()).
		WithWriteTimeout(networkTimeout).
		WithKeepAliveInterval(pingInterval).
		WithSlogAttributes(slog.String("suite", t.Name()), slog.String("impl", impl.String()))

	conn, err := net.Dial("tcp", config.DefaultIP+":"+port)
	require.NoError(t, err, "failed to dial TCP to %s node", impl.String())

	s, err := n.NewSession(ctx, conn, conf)
	require.NoError(t, err, "failed to establish new session to %s node", impl.String())

	cli := &NetClient{ctx: ctx, t: t, impl: impl, n: n, c: conf, h: h, s: s}
	h.client = cli // Set client reference in handler.
	return cli
}

func (c *NetClient) SendHandshake() {
	handshake := &proto.Handshake{
		AppName:      appName,
		Version:      proto.ProtocolVersion(),
		NodeName:     "itest",
		NodeNonce:    nonce,
		DeclaredAddr: proto.HandshakeTCPAddr{},
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}
	buf := bytes.NewBuffer(nil)
	_, err := handshake.WriteTo(buf)
	require.NoError(c.t, err,
		"failed to marshal handshake to %s node at %q", c.impl.String(), c.s.RemoteAddr())
	_, err = c.s.Write(buf.Bytes())
	require.NoError(c.t, err,
		"failed to send handshake to %s node at %q", c.impl.String(), c.s.RemoteAddr())
}

func (c *NetClient) SendMessage(m proto.Message) {
	_, err := m.WriteTo(c.s)
	require.NoError(c.t, err, "failed to send message to %s node at %q", c.impl.String(), c.s.RemoteAddr())
}

func (c *NetClient) Close() {
	c.closed.Do(func() {
		if c.closing.CompareAndSwap(false, true) {
			c.t.Logf("Closing connection to %s node at %q", c.impl.String(), c.s.RemoteAddr().String())
		}
		err := c.s.Close()
		require.NoError(c.t, err, "failed to close session to %s node at %q", c.impl.String(), c.s.RemoteAddr())
		c.h.close()
	})
}

// SubscribeForMessages adds specified types to the message waiting queue.
// Once the awaited message received the corresponding type is removed from the queue.
func (c *NetClient) SubscribeForMessages(messageType ...reflect.Type) error {
	for _, mt := range messageType {
		if err := c.h.waitFor(mt); err != nil {
			return err
		}
	}
	return nil
}

// AwaitMessage waits for a message from the node for the specified timeout.
func (c *NetClient) AwaitMessage(messageType reflect.Type, timeout time.Duration) (proto.Message, error) {
	select {
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for message of type %q", messageType.String())
	case msg := <-c.h.receiveChan():
		if reflect.TypeOf(msg) != messageType {
			return nil, fmt.Errorf("unexpected message type %q, expecting %q",
				reflect.TypeOf(msg).String(), messageType.String())
		}
		return msg, nil
	}
}

// AwaitGetBlockMessage waits for a GetBlockMessage from the node for the specified timeout and
// returns the requested block ID.
func (c *NetClient) AwaitGetBlockMessage(timeout time.Duration) (proto.BlockID, error) {
	msg, err := c.AwaitMessage(reflect.TypeOf(&proto.GetBlockMessage{}), timeout)
	if err != nil {
		return proto.BlockID{}, err
	}
	getBlockMessage, ok := msg.(*proto.GetBlockMessage)
	if !ok {
		return proto.BlockID{}, fmt.Errorf("failed to cast message of type %q to GetBlockMessage",
			reflect.TypeOf(msg).String())
	}
	return getBlockMessage.BlockID, nil
}

// AwaitScoreMessage waits for a ScoreMessage from the node for the specified timeout and returns the received score.
func (c *NetClient) AwaitScoreMessage(timeout time.Duration) (*big.Int, error) {
	msg, err := c.AwaitMessage(reflect.TypeOf(&proto.ScoreMessage{}), timeout)
	if err != nil {
		return nil, err
	}
	scoreMessage, ok := msg.(*proto.ScoreMessage)
	if !ok {
		return nil, fmt.Errorf("failed to cast message of type %q to ScoreMessage", reflect.TypeOf(msg).String())
	}
	score := new(big.Int).SetBytes(scoreMessage.Score)
	return score, nil
}

// AwaitMicroblockRequest waits for a MicroBlockRequestMessage from the node for the specified timeout and
// returns the received block ID.
func (c *NetClient) AwaitMicroblockRequest(timeout time.Duration) (proto.BlockID, error) {
	msg, err := c.AwaitMessage(reflect.TypeOf(&proto.MicroBlockRequestMessage{}), timeout)
	if err != nil {
		return proto.BlockID{}, err
	}
	mbr, ok := msg.(*proto.MicroBlockRequestMessage)
	if !ok {
		return proto.BlockID{}, fmt.Errorf("failed to cast message of type %q to MicroBlockRequestMessage",
			reflect.TypeOf(msg).String())
	}
	r, err := proto.NewBlockIDFromBytes(mbr.TotalBlockSig)
	if err != nil {
		return proto.BlockID{}, err
	}
	return r, nil
}

func (c *NetClient) reconnect() {
	c.t.Logf("Reconnecting to %q", c.s.RemoteAddr().String())
	conn, err := net.Dial("tcp", c.s.RemoteAddr().String())
	require.NoError(c.t, err, "failed to dial TCP to %s node", c.impl.String())

	s, err := c.n.NewSession(c.ctx, conn, c.c)
	require.NoError(c.t, err, "failed to re-establish the session to %s node", c.impl.String())
	c.s = s

	c.SendHandshake()
}

type protocol struct {
	t        testing.TB
	dropLock sync.Mutex
	drop     map[proto.PeerMessageID]struct{}
}

func newProtocol(t testing.TB, drop []proto.PeerMessageID) *protocol {
	m := make(map[proto.PeerMessageID]struct{})
	for _, id := range drop {
		m[id] = struct{}{}
	}
	return &protocol{t: t, drop: m}
}

func (p *protocol) EmptyHandshake() networking.Handshake {
	return &proto.Handshake{}
}

func (p *protocol) EmptyHeader() networking.Header {
	return &proto.Header{}
}

func (p *protocol) Ping() ([]byte, error) {
	msg := &proto.GetPeersMessage{}
	return msg.MarshalBinary()
}

func (p *protocol) IsAcceptableHandshake(h networking.Handshake) bool {
	hs, ok := h.(*proto.Handshake)
	if !ok {
		return false
	}
	// Reject nodes with incorrect network bytes, unsupported protocol versions,
	// or a zero nonce (indicating a self-connection).
	if hs.AppName != appName || hs.Version.Cmp(proto.ProtocolVersion()) < 0 || hs.NodeNonce == 0 {
		p.t.Logf("Unacceptable handshake:")
		if hs.AppName != appName {
			p.t.Logf("\tinvalid application name %q, expected %q", hs.AppName, appName)
		}
		if hs.Version.Cmp(proto.ProtocolVersion()) < 0 {
			p.t.Logf("\tinvalid application version %q should be equal or more than %q",
				hs.Version, proto.ProtocolVersion())
		}
		if hs.NodeNonce == 0 {
			p.t.Logf("\tinvalid node nonce %d", hs.NodeNonce)
		}
		return false
	}
	return true
}

func (p *protocol) IsAcceptableMessage(h networking.Header) bool {
	hdr, ok := h.(*proto.Header)
	if !ok {
		return false
	}
	p.dropLock.Lock()
	defer p.dropLock.Unlock()
	_, ok = p.drop[hdr.ContentID]
	return !ok
}

type handler struct {
	peers  []proto.PeerInfo
	t      testing.TB
	client *NetClient
	queue  []reflect.Type
	ch     chan proto.Message
}

func newHandler(t testing.TB, peers []proto.PeerInfo) *handler {
	ch := make(chan proto.Message, 1)
	return &handler{t: t, peers: peers, ch: ch}
}

func (h *handler) OnReceive(s *networking.Session, r io.Reader) {
	data, err := io.ReadAll(r)
	if err != nil {
		h.t.Logf("Failed to read message from %q: %v", s.RemoteAddr(), err)
		h.t.FailNow()
		return
	}
	msg, err := proto.UnmarshalMessage(data)
	if err != nil { // Fail test on unmarshal error.
		h.t.Logf("Failed to unmarshal message from bytes: %q", base64.StdEncoding.EncodeToString(data))
		h.t.FailNow()
		return
	}
	switch msg.(type) { // Only reply with peers on GetPeersMessage.
	case *proto.GetPeersMessage:
		rpl := &proto.PeersMessage{Peers: h.peers}
		if _, sErr := rpl.WriteTo(s); sErr != nil {
			h.t.Logf("Failed to send peers message: %v", sErr)
			h.t.FailNow()
			return
		}
	default:
		if len(h.queue) == 0 { // No messages to wait for.
			return
		}
		et := h.queue[0]
		if reflect.TypeOf(msg) == et {
			h.t.Logf("Received expected message of type %q", reflect.TypeOf(msg).String())
			h.queue = h.queue[1:] // Pop the expected type.
			h.ch <- msg
		}
	}
}

func (h *handler) OnHandshake(_ *networking.Session, _ networking.Handshake) {
	h.t.Logf("Connection to %s node at %q was established", h.client.impl.String(), h.client.s.RemoteAddr())
}

func (h *handler) OnClose(s *networking.Session) {
	h.t.Logf("Connection to %q was closed", s.RemoteAddr())
	if !h.client.closing.Load() && h.client != nil {
		h.client.reconnect()
	}
}

func (h *handler) waitFor(messageType reflect.Type) error {
	if messageType == nil {
		return errors.New("nil message type")
	}
	if messageType == reflect.TypeOf(proto.GetPeersMessage{}) {
		return errors.New("cannot wait for GetPeersMessage")
	}
	h.queue = append(h.queue, messageType)
	return nil
}

func (h *handler) receiveChan() <-chan proto.Message {
	return h.ch
}

func (h *handler) close() {
	if h.ch != nil {
		close(h.ch)
		h.ch = nil
	}
}
