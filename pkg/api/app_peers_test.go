package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestApp_PeersAll(t *testing.T) {
	s := &node.MockStateManager{
		Peers_: []proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")},
	}

	app, err := NewApp("key", s, nil, nil, nil, nil)
	require.NoError(t, err)

	rs2, err := app.PeersAll()
	require.NoError(t, err)
	require.Len(t, rs2.Peers, 1)
}
