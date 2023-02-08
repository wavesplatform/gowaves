package node_client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type GrpcClient struct {
	conn *grpc.ClientConn
}

func NewGrpcClient(t *testing.T, port string) *GrpcClient {
	conn, err := grpc.Dial(d.Localhost+":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err, "failed to dial grpc")
	return &GrpcClient{conn: conn}
}

func (c *GrpcClient) GetHeight(t *testing.T) *client.BlocksHeight {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	h, err := g.NewBlocksApiClient(c.conn).GetCurrentHeight(ctx, &emptypb.Empty{}, grpc.EmptyCallOption{})
	assert.NoError(t, err, "(grpc) failed to get height from node")
	return &client.BlocksHeight{Height: uint64(h.Value)}
}

func (c *GrpcClient) GetWavesBalance(t *testing.T, address proto.WavesAddress) *g.BalanceResponse_WavesBalances {
	return getBalance(t, c.conn, &g.BalancesRequest{Address: address.Bytes(), Assets: [][]byte{nil}}).GetWaves()
}

func (c *GrpcClient) GetAssetBalance(t *testing.T, address proto.WavesAddress, id []byte) *waves.Amount {
	require.NotEmpty(t, id, "asset bytes must not be empty")
	return getBalance(t, c.conn, &g.BalancesRequest{Address: address.Bytes(), Assets: [][]byte{id}}).GetAsset()
}

func getBalance(t *testing.T, conn *grpc.ClientConn, req *g.BalancesRequest) *g.BalanceResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	stream, err := g.NewAccountsApiClient(conn).GetBalances(ctx, req, grpc.EmptyCallOption{})
	assert.NoError(t, err, "(grpc) failed to get stream")
	b, err := stream.Recv()
	assert.NoError(t, err, "(grpc) failed to get balance from node")
	return b
}

func (c *GrpcClient) GetAddressByAlias(t *testing.T, alias string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	addr, err := g.NewAccountsApiClient(c.conn).ResolveAlias(ctx, &wrapperspb.StringValue{Value: alias})
	assert.NoError(t, err)
	return addr.GetValue()
}
