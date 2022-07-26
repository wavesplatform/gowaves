package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetBaseTarget(t *testing.T) {
	dataDir := t.TempDir()
	params := defaultStateParams()
	params.StoreExtendedApiData = true
	st, err := state.NewState(dataDir, true, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	t.Cleanup(func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
	})

	cl := g.NewBlockchainApiClient(conn)

	res, err := cl.GetBaseTarget(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	// MainNet Genesis base target.
	assert.Equal(t, int64(153722867), res.BaseTarget)

	// This target is base target of block at height 3 on MainNet.
	newTarget := 171657201
	blocks, err := state.ReadMainnetBlocksToHeight(proto.Height(3))
	assert.NoError(t, err)
	_, err = st.AddDeserializedBlocks(blocks)
	assert.NoError(t, err)
	// Check new base target.
	res, err = cl.GetBaseTarget(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int64(newTarget), res.BaseTarget)
}

func TestGetCumulativeScore(t *testing.T) {
	dataDir := t.TempDir()
	params := defaultStateParams()
	st, err := state.NewState(dataDir, true, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	t.Cleanup(func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
	})

	cl := g.NewBlockchainApiClient(conn)

	res, err := cl.GetCumulativeScore(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	genesisTarget := uint64(153722867)
	result, err := state.CalculateScore(genesisTarget)
	assert.NoError(t, err)
	resultBytes, err := result.GobEncode()
	assert.NoError(t, err)
	assert.Equal(t, resultBytes, res.Score)
}
