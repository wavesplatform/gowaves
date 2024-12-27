package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
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
	log := slogt.New(t)
	conf := networking.NewConfig(p, h).
		WithSlogHandler(log.Handler()).
		WithWriteTimeout(networkTimeout).
		WithKeepAliveInterval(pingInterval).
		WithSlogAttributes(slog.String("suite", t.Name()), slog.String("impl", impl.String()))

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
	})
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
}

func newHandler(t testing.TB, peers []proto.PeerInfo) *handler {
	return &handler{t: t, peers: peers}
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
		h.t.Logf("Received GetPeersMessage from %q", s.RemoteAddr())
		rpl := &proto.PeersMessage{Peers: h.peers}
		if _, sErr := rpl.WriteTo(s); sErr != nil {
			h.t.Logf("Failed to send peers message: %v", sErr)
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
	if !h.client.closing.Load() && h.client != nil {
		h.client.reconnect()
	}
}
