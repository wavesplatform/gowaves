package peer

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
	"time"
)

type server struct {
	conn        net.Conn
	l           net.Listener
	readedBytes [][]byte
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
		a.readedBytes = append(a.readedBytes, b)
	}
}

func (a *server) Addr() net.Addr {
	return a.l.Addr()
}

func (a *server) stop() {
	_ = a.conn.SetDeadline(time.Now().Add(-1 * time.Second))
	a.conn.Close()
}

func callback(b []byte, address string, resendTo chan ProtoMessage, pool conn.Pool) {
	panic("call callback")
}

func TestOutgoingPeer_SendMessage(t *testing.T) {
	server := runServerAsync("127.0.0.1:")
	defer server.stop()

	ctx := context.Background()

	parent := Parent{
		MessageCh: make(chan ProtoMessage, 10),
		InfoCh:    make(chan InfoMessage, 10),
	}

	params := OutgoingPeerParams{
		Ctx:                       ctx,
		Address:                   server.Addr().String(),
		Parent:                    parent,
		ReceiveFromRemoteCallback: callback,
		Pool:                      bytespool.NewBytesPool(10, 2*1024*1024),
		DeclAddr:                  proto.PeerInfo{},
		SpawnedPeers:              NoOpDeleter{},
	}
	go RunOutgoingPeer(params)

	select {
	case <-time.After(10 * time.Millisecond):
		t.Error("no message arrived in 100ms")
		return
	case m := <-parent.InfoCh:
		connected := m.Value.(*Connected)
		connected.Peer.SendMessage(&proto.GetPeersMessage{})
	}
	// waiting 10ms for error messages, no errors should arrive
	select {
	case m := <-parent.InfoCh:
		t.Fatalf("got unexpected message %+v", m)
	case <-time.After(10 * time.Millisecond):
	}

	assert.Equal(t, 1, len(server.readedBytes), "server should have exactly 1 message")
	getPeersM := proto.GetPeersMessage{}
	err := getPeersM.UnmarshalBinary(server.readedBytes[0])
	require.NoError(t, err, "message should be of type proto.GetPeersMessage and unmarshal correctly")
}
