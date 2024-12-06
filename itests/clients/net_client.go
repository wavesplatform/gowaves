package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"sync"
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
	s    *networking.Session

	closingLock sync.Mutex
	closingFlag bool
	closedLock  sync.Mutex
	closedFlag  bool
}

func NewNetClient(
	ctx context.Context, t testing.TB, impl Implementation, port string, peers []proto.PeerInfo,
) *NetClient {
	n := networking.NewNetwork()
	p := newProtocol(nil)
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
		WithLogger(log).
		WithWriteTimeout(networkTimeout).
		WithKeepAliveInterval(pingInterval).
		WithSlogAttribute(slog.String("suite", t.Name())).
		WithSlogAttribute(slog.String("impl", impl.String()))

	conn, err := net.Dial("tcp", config.DefaultIP+":"+port)
	require.NoError(t, err, "failed to dial TCP to %s node", impl.String())

	s, err := n.NewSession(ctx, conn, conf)
	require.NoError(t, err, "failed to establish new session to %s node", impl.String())

	cli := &NetClient{ctx: ctx, t: t, impl: impl, n: n, c: conf, s: s}
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
	b, err := m.MarshalBinary()
	require.NoError(c.t, err, "failed to marshal message to %s node at %q", c.impl.String(), c.s.RemoteAddr())
	_, err = c.s.Write(b)
	require.NoError(c.t, err, "failed to send message to %s node at %q", c.impl.String(), c.s.RemoteAddr())
}

func (c *NetClient) Close() {
	c.t.Logf("Trying to close connection to %s node at %q", c.impl.String(), c.s.RemoteAddr().String())

	c.closingLock.Lock()
	c.closingFlag = true
	c.closingLock.Unlock()

	c.closedLock.Lock()
	defer c.closedLock.Unlock()
	c.t.Logf("Closing connection to %s node at %q (%t)", c.impl.String(), c.s.RemoteAddr().String(), c.closedFlag)
	if c.closedFlag {
		return
	}
	_ = c.s.Close()
	c.closedFlag = true
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

func (c *NetClient) closing() bool {
	c.closingLock.Lock()
	defer c.closingLock.Unlock()
	return c.closingFlag
}

type protocol struct {
	dropLock sync.Mutex
	drop     map[proto.PeerMessageID]struct{}
}

func newProtocol(drop []proto.PeerMessageID) *protocol {
	m := make(map[proto.PeerMessageID]struct{})
	for _, id := range drop {
		m[id] = struct{}{}
	}
	return &protocol{drop: m}
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
}

func newHandler(t testing.TB, peers []proto.PeerInfo) *handler {
	return &handler{t: t, peers: peers}
}

func (h *handler) OnReceive(s *networking.Session, data []byte) {
	msg, err := proto.UnmarshalMessage(data)
	if err != nil { // Fail test on unmarshal error.
		h.t.Logf("Failed to unmarshal message from bytes: %q", base64.StdEncoding.EncodeToString(data))
		h.t.FailNow()
		return
	}
	switch msg.(type) { // Only reply with peers on GetPeersMessage.
	case *proto.GetPeersMessage:
		rpl := &proto.PeersMessage{Peers: h.peers}
		bts, mErr := rpl.MarshalBinary()
		if mErr != nil { // Fail test on marshal error.
			h.t.Logf("Failed to marshal peers message: %v", mErr)
			h.t.FailNow()
			return
		}
		if _, wErr := s.Write(bts); wErr != nil {
			h.t.Logf("Failed to send peers message: %v", wErr)
			h.t.FailNow()
			return
		}
	default:
	}
}

func (h *handler) OnHandshake(_ *networking.Session, _ networking.Handshake) {
	h.t.Logf("Connection to %s node at %q was established", h.client.impl.String(), h.client.s.RemoteAddr())
}

func (h *handler) OnClose(s *networking.Session) {
	h.t.Logf("Connection to %q was closed", s.RemoteAddr())
	if !h.client.closing() && h.client != nil {
		h.client.reconnect()
	}
}
