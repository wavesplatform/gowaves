package network

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type server struct {
	conn      net.Conn
	l         net.Listener
	readBytes [][]byte
	mu        sync.Mutex
}

func (a *server) addReadBytes(b []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.readBytes = append(a.readBytes, b)
}

func (a *server) GetReadBytes() [][]byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.readBytes
}

func runServerAsync(listen string) *server {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		panic(err)
	}
	a := &server{
		l: l,
	}

	go a.listen(l)
	return a
}

func (a *server) listen(l net.Listener) {

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		a.conn = conn
		h := proto.Handshake{}
		_, err = h.ReadFrom(conn)
		if err != nil {
			panic(err)
		}
		h.NodeName = "server"
		_, err = h.WriteTo(conn)
		if err != nil {
			panic(err)
		}

		b := make([]byte, 1024)
		_, err = conn.Read(b)
		if err != nil {
			//fmt.Println(err)
			zap.S().Error(err)
			return
		}
		a.addReadBytes(b)
	}
}

func (a *server) Addr() net.Addr {
	return a.l.Addr()
}

func (a *server) stop() {
	_ = a.conn.SetDeadline(time.Now().Add(-1 * time.Second))
	_ = a.conn.Close()
}

func TestOutgoingPeer_SendMessage(t *testing.T) {
	server := runServerAsync("127.0.0.1:")
	defer server.stop()

	ctx := context.Background()

	parent := peer.Parent{
		MessageCh: make(chan peer.ProtoMessage, 10),
		InfoCh:    make(chan peer.InfoMessage, 10),
	}

	params := OutgoingPeerParams{
		Address:  server.Addr().String(),
		Parent:   parent,
		DeclAddr: proto.TCPAddr{},
	}
	go RunOutgoingPeer(ctx, params)

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("no message arrived in 100ms")
		return
	case m := <-parent.InfoCh:
		connected := m.Value.(*peer.Connected)
		connected.Peer.SendMessage(&proto.GetPeersMessage{})
	}
	// waiting 10ms for error messages, no errors should arrive
	select {
	case m := <-parent.InfoCh:
		t.Fatalf("got unexpected message %+v", m)
	case <-time.After(10 * time.Millisecond):
	}

	assert.Equal(t, 1, len(server.GetReadBytes()), "server should have exactly 1 message")
	getPeersM := proto.GetPeersMessage{}
	err := getPeersM.UnmarshalBinary(server.GetReadBytes()[0])
	require.NoError(t, err, "message should be of type proto.GetPeersMessage and unmarshal correctly")
}
