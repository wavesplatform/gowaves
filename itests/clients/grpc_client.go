package clients

import (
	"context"
	"io"
	"math"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const defaultTimeout = 30 * time.Second

type balanceAtHeight struct {
	impl    Implementation
	balance int64
	height  uint64
}

type GRPCClient struct {
	impl    Implementation
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewGRPCClient(t *testing.T, impl Implementation, port string) *GRPCClient {
	conn, err := grpc.NewClient(config.DefaultIP+":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoErrorf(t, err, "failed to dial GRPC to %s", impl.String())
	return &GRPCClient{impl: impl, conn: conn, timeout: defaultTimeout}
}

func (c *GRPCClient) GetFeatureActivationStatusInfo(t *testing.T, h int32) *g.ActivationStatusResponse {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	response, err := g.NewBlockchainApiClient(c.conn).GetActivationStatus(ctx, &g.ActivationStatusRequest{Height: h})
	require.NoErrorf(t, err, "[GRPC] failed to get feature activation status from %s node", c.impl.String())
	return response
}

func (c *GRPCClient) GetHeight(t *testing.T) *client.BlocksHeight {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	h, err := g.NewBlocksApiClient(c.conn).GetCurrentHeight(ctx, &emptypb.Empty{}, grpc.EmptyCallOption{})
	assert.NoErrorf(t, err, "[GRPC] failed to get height from %s node", c.impl.String())
	return &client.BlocksHeight{Height: uint64(h.Value)}
}

func (c *GRPCClient) GetBlock(t *testing.T, height uint64) *g.BlockWithHeight {
	if height > math.MaxInt32 {
		require.FailNow(t, "height is too large to be casted to int32")
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	block, err := g.NewBlocksApiClient(c.conn).GetBlock(ctx,
		&g.BlockRequest{Request: &g.BlockRequest_Height{Height: int32(height)}, IncludeTransactions: true})
	assert.NoErrorf(t, err, "[GRPC] failed to get block from %s node", c.impl.String())
	return block
}

func (c *GRPCClient) GetWavesBalance(t *testing.T, address proto.WavesAddress) *g.BalanceResponse_WavesBalances {
	return c.getBalance(t, &g.BalancesRequest{Address: address.Bytes(), Assets: [][]byte{nil}}).GetWaves()
}

func (c *GRPCClient) GetAssetBalance(t *testing.T, address proto.WavesAddress, id []byte) *waves.Amount {
	require.NotEmptyf(t, id, "asset bytes must not be empty than calling %s node", c.impl.String())
	return c.getBalance(t, &g.BalancesRequest{Address: address.Bytes(), Assets: [][]byte{id}}).GetAsset()
}

func (c *GRPCClient) GetAddressByAlias(t *testing.T, alias string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	addr, err := g.NewAccountsApiClient(c.conn).ResolveAlias(ctx, &wrapperspb.StringValue{Value: alias})
	assert.NoErrorf(t, err, "failed to get address by alias from %s node", c.impl.String())
	return addr.GetValue()
}

func (c *GRPCClient) GetAssetsInfo(t *testing.T, id []byte) *g.AssetInfoResponse {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	assetInfo, err := g.NewAssetsApiClient(c.conn).GetInfo(ctx, &g.AssetRequest{AssetId: id})
	assert.NoErrorf(t, err, "failed to get asset info from %s node", c.impl.String())
	return assetInfo
}

func (c *GRPCClient) Close(t testing.TB) {
	err := c.conn.Close()
	assert.NoErrorf(t, err, "failed to close GRPC connection to %s node", c.impl.String())
}

func (c *GRPCClient) getBalance(t *testing.T, req *g.BalancesRequest) *g.BalanceResponse {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	stream, err := g.NewAccountsApiClient(c.conn).GetBalances(ctx, req, grpc.EmptyCallOption{})
	assert.NoErrorf(t, err, "[GRPC] failed to get stream from %s node", c.impl.String())
	b, err := stream.Recv()
	assert.NoErrorf(t, err, "[GRPC] failed to get balance from %s node", c.impl.String())
	return b
}

func (c *GRPCClient) syncedWavesAvailableBalance(
	ctx context.Context, address proto.WavesAddress,
) (balanceAtHeight, error) {
	beforeRsp, err := g.NewBlocksApiClient(c.conn).GetCurrentHeight(ctx, &emptypb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return balanceAtHeight{}, errors.Wrapf(err,
			"syncedWavesAvailableBalance: failed to get initial height from %s node", c.impl.String())
	}
	before := uint64(beforeRsp.Value)

	req := &g.BalancesRequest{Address: address.Bytes(), Assets: [][]byte{nil}}
	stream, err := g.NewAccountsApiClient(c.conn).GetBalances(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return balanceAtHeight{}, errors.Wrapf(err,
			"syncedWavesAvailableBalance: failed to get balance stream from %s node", c.impl.String())
	}
	balanceRsp, err := stream.Recv()
	if err != nil {
		return balanceAtHeight{}, errors.Wrapf(err,
			"syncedWavesAvailableBalance: failed to get balance from %s node", c.impl.String())
	}
	available := balanceRsp.GetWaves().Available

	afterRsp, err := g.NewBlocksApiClient(c.conn).GetCurrentHeight(ctx, &emptypb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return balanceAtHeight{}, errors.Wrapf(err,
			"syncedWavesAvailableBalance: failed to get height from %s node", c.impl.String())
	}
	after := uint64(afterRsp.Value)

	if before != after {
		return balanceAtHeight{}, errors.Errorf(
			"syncedWavesAvailableBalance: height changed during balance check on %s node", c.impl.String())
	}
	return balanceAtHeight{impl: c.impl, balance: available, height: after}, nil
}

// GetDataEntryByKey return data entries for account by key.
func (c *GRPCClient) GetDataEntryByKey(t *testing.T, address proto.WavesAddress, key string) *waves.DataEntry {
	ctx, cancel := context.WithTimeout(t.Context(), c.timeout)
	defer cancel()
	dr := g.DataRequest{
		Address: address.Bytes(),
		Key:     key,
	}
	stream, err := g.NewAccountsApiClient(c.conn).GetDataEntries(ctx, &dr, grpc.EmptyCallOption{})
	assert.NoErrorf(t, err, "failed to get data entries from %s node with error: %s", c.impl.String(), err)
	d, err := stream.Recv()
	assert.NoErrorf(t, err, "failed to get data entry from %s node with error: %s", c.impl.String(), err)
	return d.GetEntry()
}

// GetDataEntries returns all data entries for account.
func (c *GRPCClient) GetDataEntries(t *testing.T, address proto.WavesAddress) []*waves.DataEntry {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	de := make([]*waves.DataEntry, 0)
	defer cancel()
	dr := g.DataRequest{
		Address: address.Bytes(),
	}
	stream, err := g.NewAccountsApiClient(c.conn).GetDataEntries(ctx, &dr, grpc.EmptyCallOption{})
	assert.NoErrorf(t, err, "failed to get data entries from %s node with error: %s", c.impl.String(), err)
	for {
		d, errStrm := stream.Recv()
		if errors.Is(errStrm, io.EOF) {
			break
		}
		require.NoErrorf(t, errStrm, "failed to get data entry from %s node with error: %s",
			c.impl.String(), errStrm)
		de = append(de, d.GetEntry())
	}
	return de
}

func (c *GRPCClient) GetTransactionsStatuses(t *testing.T, txIDs []crypto.Digest) []*g.TransactionStatus {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	var grpcStatuses []*g.TransactionStatus
	defer cancel()

	ids := make([][]byte, len(txIDs))
	for i, digest := range txIDs {
		ids[i] = digest.Bytes()
	}

	tx := g.TransactionsByIdRequest{
		TransactionIds: ids,
	}
	stream, err := g.NewTransactionsApiClient(c.conn).GetStatuses(ctx, &tx, grpc.EmptyCallOption{})
	assert.NoErrorf(t, err, "failed to get transaction statuses from %s node with error: %s",
		c.impl.String(), err)
	for {
		tr, errStrm := stream.Recv()
		if errors.Is(errStrm, io.EOF) {
			break
		}
		require.NoErrorf(t, errStrm, "failed to get transaction status from %s node with error: %s",
			c.impl.String(), errStrm)
		grpcStatuses = append(grpcStatuses, tr)
	}
	return grpcStatuses
}
