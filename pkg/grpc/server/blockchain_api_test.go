package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"google.golang.org/grpc"
)

func connect(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	assert.NoError(t, err, "grpc.Dial() failed")
	return conn
}

func TestGetBaseTarget(t *testing.T) {
	grpcTestAddr := fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	st, err := state.NewState(dataDir, state.DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	cl := g.NewBlockchainApiClient(conn)
	server := NewServer(st)
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			t.Error("server.Run failed")
		}
	}()

	time.Sleep(5 * time.Second)
	res, err := cl.GetBaseTarget(ctx, &empty.Empty{})
	assert.NoError(t, err)
	// MainNet Genesis base target.
	assert.Equal(t, int64(153722867), res.BaseTarget)

	// This target is base target of block at height 3 on MainNet.
	newTarget := 171657201
	blocks := state.ReadMainnetBlocksToHeight(t, proto.Height(3))
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)
	// Check new base target.
	res, err = cl.GetBaseTarget(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int64(newTarget), res.BaseTarget)
}

func TestGetCumulativeScore(t *testing.T) {
	grpcTestAddr := fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	st, err := state.NewState(dataDir, state.DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	cl := g.NewBlockchainApiClient(conn)
	server := NewServer(st)
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			t.Error("server.Run failed")
		}
	}()

	time.Sleep(5 * time.Second)
	res, err := cl.GetCumulativeScore(ctx, &empty.Empty{})
	assert.NoError(t, err)
	genesisTarget := uint64(153722867)
	result, err := state.CalculateScore(genesisTarget)
	assert.NoError(t, err)
	resultBytes, err := result.GobEncode()
	assert.NoError(t, err)
	assert.Equal(t, resultBytes, res.Score)
}
