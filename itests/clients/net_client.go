package clients

import (
	"bytes"
	"context"
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

	"github.com/wavesplatform/gowaves/pkg/execution"
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
	ctx    context.Context
	cancel context.CancelFunc
	t      testing.TB
	impl   Implementation
	n      *networking.Network
	c      *networking.Config
	h      *handler
	s      *networking.Session
	tg     *execution.TaskGroup
	closed atomic.Bool
}

// NewNetClient creates a new NetClient instance that connects to the specified Waves node.
// It establishes a TCP connection to given address.
// Parameter peers is a list of peer information that the client will use to respond to GetPeersMessage requests.
func NewNetClient(
	ctx context.Context, t testing.TB, impl Implementation, address string, peers []proto.PeerInfo,
) *NetClient {
	ctx, cancel := context.WithCancel(ctx)
	n := networking.NewNetwork()
	p := newProtocol(t, nil)
	h := newHandler(t, impl, peers)

	f := slogt.Factory(func(w io.Writer) slog.Handler {
		opts := &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}
		return slog.NewTextHandler(w, opts)
	})
	log := slogt.New(t, f)

	slog.SetLogLoggerLevel(slog.LevelError)
	conf := networking.NewConfig().
		WithProtocol(p).
		WithHandler(h).
		WithSlogHandler(log.Handler()).
		WithWriteTimeout(networkTimeout).
		WithKeepAliveInterval(pingInterval).
		WithSlogAttributes(slog.String("suite", t.Name()), slog.String("impl", impl.String()))

	conn, err := net.Dial("tcp", address)
	require.NoErrorf(t, err, "failed to dial TCP to %s node at %q", impl.String(), address)

	s, err := n.NewSession(ctx, conn, conf)
	require.NoErrorf(t, err, "failed to establish new session to %s node", impl.String())

	nc := &NetClient{
		ctx:    ctx,
		cancel: cancel,
		t:      t,
		impl:   impl,
		n:      n,
		c:      conf,
		h:      h,
		s:      s,
		tg:     execution.NewTaskGroup(suppressContextCancellationError),
	}
	nc.tg.Run(nc.watch)

	return nc
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
	if err != nil {
		//TODO: It is possible now to detect if the peer closed the connection during the write.
		// We can use this to check for expected disconnects, for example,
		// when we send a malformed transactions to a node.
		c.t.Logf("[%s] Failed to send message of type %T to %s node at %q: %v",
			time.Now().Format(time.RFC3339Nano), m, c.impl.String(), c.s.RemoteAddr(), err)
	}
}

func (c *NetClient) Close() {
	if c.closed.CompareAndSwap(false, true) {
		c.t.Logf("[%s] Closing NetClient to %s node at %q",
			time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr())
		c.cancel()
		err := c.s.Close()
		require.NoErrorf(c.t, err, "failed to close session to %s node at %q", c.impl.String(), c.s.RemoteAddr())
		c.h.close()
		c.t.Logf("[%s] Waiting to watch loop to finish", time.Now().Format(time.RFC3339Nano))
		if wErr := c.tg.Wait(); wErr != nil {
			c.t.Logf("[%s] Watch loop of NetClient to %s node at %q finished with error: %v",
				time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr(), wErr)
		}
		c.t.Logf("[%s] Closed NetClient to %s node at %q",
			time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr())
	}
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
	return mbr.TotalBlockSig, nil
}

func (c *NetClient) watch() error {
	for {
		select {
		case <-c.ctx.Done():
			c.t.Logf("[%s] Context done, stopping watch loop", time.Now().Format(time.RFC3339Nano))
			return c.ctx.Err()
		case _, ok := <-c.h.reconnectChan():
			c.t.Logf("[%s] Got a reconnect event", time.Now().Format(time.RFC3339Nano))
			if !ok {
				c.t.Logf("[%s] Reconnect channel was closed, exiting watch loop", time.Now().Format(time.RFC3339Nano))
				return nil // Channel was closed, exit the watch loop.
			}
			c.reconnect()
		case <-c.h.closeChan():
			c.t.Logf("[%s] Got a close event", time.Now().Format(time.RFC3339Nano))
			return nil
		}
	}
}

func (c *NetClient) reconnect() {
	c.t.Logf("[%s] Reconnecting to %s node at %q",
		time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr().String())
	if c.closed.Load() {
		return
	}
	if clErr := c.s.Close(); clErr != nil {
		c.t.Logf("[%s] Failed to close session to %s node at %q: %v",
			time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr().String(), clErr)
	}
	c.t.Logf("[%s] Reconnecting to %q", time.Now().Format(time.RFC3339Nano), c.s.RemoteAddr().String())
	conn, err := net.Dial("tcp", c.s.RemoteAddr().String())
	require.NoErrorf(c.t, err, "failed to dial TCP to %s node at %q",
		c.impl.String(), c.s.RemoteAddr().String())

	s, err := c.n.NewSession(c.ctx, conn, c.c)
	require.NoErrorf(c.t, err, "failed to re-establish the session to %s node", c.impl.String())
	c.s = s
	c.t.Logf("[%s] Reconnected to node %s at %q",
		time.Now().Format(time.RFC3339Nano), c.impl.String(), c.s.RemoteAddr().String())
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

func (p *protocol) IsAcceptableHandshake(_ *networking.Session, h networking.Handshake) bool {
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

func (p *protocol) IsAcceptableMessage(_ *networking.Session, h networking.Header) bool {
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
	peers       []proto.PeerInfo
	t           testing.TB
	impl        Implementation
	queue       []reflect.Type
	reconnectCh chan struct{}      // Channel to signal that the session should be reconnected.
	closeCh     chan struct{}      // Channel to signal that the session should be closed.
	ch          chan proto.Message // Channel to notify about received messages.
}

func newHandler(t testing.TB, impl Implementation, peers []proto.PeerInfo) *handler {
	return &handler{
		t:           t,
		impl:        impl,
		peers:       peers,
		reconnectCh: make(chan struct{}),
		closeCh:     make(chan struct{}),
		ch:          make(chan proto.Message, 1),
	}
}

func (h *handler) OnReceive(s networking.EndpointWriter, r io.Reader) {
	msg, _, err := proto.ReadMessageFrom(r)
	if err != nil {
		h.t.Logf("Failed to read message from %q: %v", s.RemoteAddr(), err)
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

func (h *handler) OnHandshake(s networking.EndpointWriter, _ networking.Handshake) {
	h.t.Logf("Connection to %s node at %q was established", h.impl.String(), s.RemoteAddrPort())
}

func (h *handler) OnHandshakeFailed(s networking.EndpointWriter, _ networking.Handshake) {
	h.t.Logf("Handshake with %q node at %q failed", h.impl.String(), s.RemoteAddrPort())
}

func (h *handler) OnClose(s networking.EndpointWriter) {
	h.t.Logf("[%s] Connection to %s node at %q was closed", time.Now().Format(time.RFC3339Nano),
		h.impl.String(), s.RemoteAddrPort())
	select { // Signal that the session was closed and should be reconnected.
	case h.reconnectCh <- struct{}{}:
	default:
	}
}

func (h *handler) OnFailure(s networking.EndpointWriter, err error) {
	h.t.Logf("[%s] Connection to %q failed: %v", time.Now().Format(time.RFC3339Nano), s.RemoteAddr(), err)
	select { // Signal that the session failed and should be closed.
	case h.closeCh <- struct{}{}:
	default:
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

func (h *handler) reconnectChan() <-chan struct{} {
	return h.reconnectCh
}

func (h *handler) closeChan() <-chan struct{} {
	return h.closeCh
}

func (h *handler) close() {
	h.t.Log("Closing handler")
	close(h.closeCh)
	h.closeCh = nil
	close(h.reconnectCh)
	h.reconnectCh = nil
	close(h.ch)
	h.ch = nil
	h.t.Log("Closed handler channels")
}

func suppressContextCancellationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
