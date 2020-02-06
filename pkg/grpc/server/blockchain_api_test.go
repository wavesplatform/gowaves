package server

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetBaseTarget(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	params.StoreExtendedApiData = true
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	cl := g.NewBlockchainApiClient(conn)

	res, err := cl.GetBaseTarget(ctx, &empty.Empty{})
	assert.NoError(t, err)
	// MainNet Genesis base target.
	assert.Equal(t, int64(153722867), res.BaseTarget)

	// This target is base target of block at height 3 on MainNet.
	newTarget := 171657201
	blocks, err := state.ReadMainnetBlocksToHeight(proto.Height(3))
	assert.NoError(t, err)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)
	// Check new base target.
	res, err = cl.GetBaseTarget(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int64(newTarget), res.BaseTarget)
}

func TestGetCumulativeScore(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	cl := g.NewBlockchainApiClient(conn)

	res, err := cl.GetCumulativeScore(ctx, &empty.Empty{})
	assert.NoError(t, err)
	genesisTarget := uint64(153722867)
	result, err := state.CalculateScore(genesisTarget)
	assert.NoError(t, err)
	resultBytes, err := result.GobEncode()
	assert.NoError(t, err)
	assert.Equal(t, resultBytes, res.Score)
}
