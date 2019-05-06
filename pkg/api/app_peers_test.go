package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type mockNode struct {
	state state.State
}

func (a mockNode) State() state.State {
	return a.state
}

func (a mockNode) PeerManager() node.PeerManager {
	panic("implement")
}
func (a mockNode) SpawnOutgoingConnection(ctx context.Context, addr proto.TCPAddr) error {
	panic("implement")
}

func TestApp_PeersAll(t *testing.T) {
	s := &node.MockStateManager{
		Peers_: []proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")},
	}

	app, err := NewApp("key", mockNode{s})
	require.NoError(t, err)

	rs2, err := app.PeersAll()
	require.NoError(t, err)
	require.Len(t, rs2.Peers, 1)
}
