package node_client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
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

func (c *GrpcClient) GetBalance(t *testing.T, address proto.WavesAddress) *g.BalanceResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	stream, err := g.NewAccountsApiClient(c.conn).GetBalances(ctx, &g.BalancesRequest{Address: address.Body()}, grpc.EmptyCallOption{})
	assert.NoError(t, err, "(grpc) failed to get stream")
	b, err := stream.Recv()
	assert.NoError(t, err, "(grpc) failed to get balance from node")
	return b
}
