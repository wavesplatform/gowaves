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
	params := defaultStateParams()
	params.StoreExtendedApiData = true
	st := newTestState(t, true, params, settings.MainNetSettings)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	err := server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

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
	params := defaultStateParams()
	st := newTestState(t, true, params, settings.MainNetSettings)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	err := server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

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
