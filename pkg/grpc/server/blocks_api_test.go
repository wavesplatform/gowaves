package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetBlock(t *testing.T) {
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

	cl := g.NewBlocksApiClient(conn)
	server, err := NewServer(st)
	assert.NoError(t, err)
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			t.Error("server.Run failed")
		}
	}()

	time.Sleep(5 * time.Second)
	// Prepare state.
	blockHeight := proto.Height(99)
	blocks := state.ReadMainnetBlocksToHeight(t, blockHeight)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)
	// Retrieve expected block.
	correctBlock, err := st.BlockByHeight(blockHeight)
	assert.NoError(t, err)
	correctBlockProto, err := correctBlock.ToProtobuf(proto.MainNetScheme, blockHeight)
	assert.NoError(t, err)
	noTransactionsProto, err := correctBlock.ToProtobuf(proto.MainNetScheme, blockHeight)
	assert.NoError(t, err)
	noTransactionsProto.Block.Transactions = nil

	sig := crypto.MustSignatureFromBase58("VaviVcQWhEz2idFT9P5YQebai2CtDrUrbqmkZNSUsKS1mNpSyg8NAyHnmrY32Cgv1oSfPdTWXqZTExNz33Edtmv")
	parent := crypto.MustSignatureFromBase58("2uN9rN94LSARneoTChNzVrDUuU9sT5CVvCtcFuRzpEtxZZAFGkCQPJiNjBJPSLo47tfXFZmgu1UdSfFeUzD9rZYX")

	// By block ID.
	req := &g.BlockRequest{Request: &g.BlockRequest_BlockId{BlockId: sig.Bytes()}, IncludeTransactions: true}
	res, err := cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctBlockProto, res))
	// Without transactions.
	req = &g.BlockRequest{Request: &g.BlockRequest_BlockId{BlockId: sig.Bytes()}, IncludeTransactions: false}
	res, err = cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(noTransactionsProto, res))

	// By height.
	req = &g.BlockRequest{Request: &g.BlockRequest_Height{Height: int32(blockHeight)}, IncludeTransactions: true}
	res, err = cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctBlockProto, res))
	// Without transactions.
	req = &g.BlockRequest{Request: &g.BlockRequest_Height{Height: int32(blockHeight)}, IncludeTransactions: false}
	res, err = cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(noTransactionsProto, res))

	// By reference.
	req = &g.BlockRequest{Request: &g.BlockRequest_Reference{Reference: parent.Bytes()}, IncludeTransactions: true}
	res, err = cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctBlockProto, res))
	// Without transactions.
	req = &g.BlockRequest{Request: &g.BlockRequest_Reference{Reference: parent.Bytes()}, IncludeTransactions: false}
	res, err = cl.GetBlock(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(noTransactionsProto, res))
}

func TestGetCurrentHeight(t *testing.T) {
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

	cl := g.NewBlocksApiClient(conn)
	server, err := NewServer(st)
	assert.NoError(t, err)
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			t.Error("server.Run failed")
		}
	}()

	time.Sleep(5 * time.Second)
	res, err := cl.GetCurrentHeight(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), res.Value)

	// Add some blocks.
	blockHeight := proto.Height(99)
	blocks := state.ReadMainnetBlocksToHeight(t, blockHeight)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)

	res, err = cl.GetCurrentHeight(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, uint32(blockHeight), res.Value)
}
