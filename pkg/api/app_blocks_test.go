package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestApp_BlocksFirst(t *testing.T) {
	g := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: crypto.MustSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"),
		},
	}

	s := node.NewMockStateManager(g)
	app, _ := NewApp("api-key", s, nil, nil, nil)
	first, _ := app.BlocksFirst()
	require.EqualValues(t, 1, first.Height)
}
func TestApp_BlocksLast(t *testing.T) {
	g := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: crypto.MustSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"),
		},
	}

	s := node.NewMockStateManager(g)
	app, _ := NewApp("api-key", s, nil, nil, nil)
	first, _ := app.BlocksLast()
	require.EqualValues(t, 1, first.Height)
}
