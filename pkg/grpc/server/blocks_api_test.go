package server

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func headerFromState(t *testing.T, height proto.Height, st state.StateInfo) *g.BlockWithHeight {
	header, err := st.HeaderByHeight(height)
	assert.NoError(t, err)
	headerProto, err := header.HeaderToProtobuf(proto.MainNetScheme, height)
	assert.NoError(t, err)
	return headerProto
}

func blockFromState(t *testing.T, height proto.Height, st state.StateInfo) *g.BlockWithHeight {
	block, err := st.BlockByHeight(height)
	assert.NoError(t, err)
	blockProto, err := block.ToProtobuf(proto.MainNetScheme, height)
	assert.NoError(t, err)
	return blockProto
}

func TestGetBlock(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, utxpool.New(utxSize), sch)
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

	cl := g.NewBlocksApiClient(conn)

	// Prepare state.
	blockHeight := proto.Height(99)
	blocks, err := state.ReadMainnetBlocksToHeight(blockHeight)
	assert.NoError(t, err)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)
	// Retrieve expected block.
	correctBlockProto := blockFromState(t, blockHeight, st)
	noTransactionsProto := headerFromState(t, blockHeight, st)

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

func TestGetBlockRange(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, utxpool.New(utxSize), sch)
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

	cl := g.NewBlocksApiClient(conn)

	// Add some blocks.
	blockHeight := proto.Height(99)
	blocks, err := state.ReadMainnetBlocksToHeight(blockHeight)
	assert.NoError(t, err)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)

	// With transactions.
	startHeight := proto.Height(10)
	endHeight := proto.Height(50)
	req := &g.BlockRangeRequest{
		FromHeight:          uint32(startHeight),
		ToHeight:            uint32(endHeight),
		IncludeTransactions: true,
	}
	stream, err := cl.GetBlockRange(ctx, req)
	assert.NoError(t, err)
	for h := startHeight; h <= endHeight; h++ {
		block, err := stream.Recv()
		assert.NoError(t, err)
		correctBlock := blockFromState(t, h, st)
		assert.True(t, protobuf.Equal(correctBlock, block))
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// Without transactions.
	req.IncludeTransactions = false
	stream, err = cl.GetBlockRange(ctx, req)
	assert.NoError(t, err)
	for h := startHeight; h <= endHeight; h++ {
		block, err := stream.Recv()
		assert.NoError(t, err)
		correctBlock := headerFromState(t, h, st)
		assert.True(t, protobuf.Equal(correctBlock, block))
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// Enable filter.
	gen := crypto.MustPublicKeyFromBase58("ARqHSzWJjTmtx3eqoFSkR6d432v2q4s1jLrYEt8axVmd")
	genBytes := gen.Bytes()
	req.Filter = &g.BlockRangeRequest_Generator{Generator: genBytes}
	stream, err = cl.GetBlockRange(ctx, req)
	assert.NoError(t, err)
	for h := startHeight; h <= endHeight; h++ {
		correctBlock := headerFromState(t, h, st)
		if !bytes.Equal(correctBlock.Block.Header.Generator, genBytes) {
			continue
		}
		block, err := stream.Recv()
		assert.NoError(t, err)
		assert.True(t, protobuf.Equal(correctBlock, block))
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetCurrentHeight(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, utxpool.New(utxSize), sch)
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

	cl := g.NewBlocksApiClient(conn)

	res, err := cl.GetCurrentHeight(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), res.Value)

	// Add some blocks.
	blockHeight := proto.Height(99)
	blocks, err := state.ReadMainnetBlocksToHeight(blockHeight)
	assert.NoError(t, err)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)

	res, err = cl.GetCurrentHeight(ctx, &empty.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, uint32(blockHeight), res.Value)
}
