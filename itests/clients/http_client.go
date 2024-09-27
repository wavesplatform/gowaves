package clients

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HTTPClient struct {
	impl    Implementation
	cli     *client.Client
	timeout time.Duration
}

func NewHTTPClient(t *testing.T, impl Implementation, port string) *HTTPClient {
	c, err := client.NewClient(client.Options{
		BaseUrl: "http://" + config.DefaultIP + ":" + port + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  d.DefaultAPIKey,
		ChainID: 'L', // I tried to use constant `utilities.TestChainID`, but after all decided that a little duplication is better in this case.
	})
	require.NoError(t, err, "couldn't create %s node HTTP API client", impl.String())
	return &HTTPClient{
		impl: impl,
		cli:  c,
		// actually, there's no need to use such timeout because above we've already set default context for http client
		timeout: 15 * time.Second,
	}
}

func (c *HTTPClient) GetHeight(t *testing.T) *client.BlocksHeight {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	h, _, err := c.cli.Blocks.Height(ctx)
	require.NoError(t, err, "failed to get height from %s node", c.impl.String())
	return h
}

func (c *HTTPClient) StateHash(t *testing.T, height uint64) *proto.StateHash {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	stateHash, _, err := c.cli.Debug.StateHash(ctx, height)
	require.NoError(t, err, "failed to get stateHash from %s node", c.impl.String())
	return stateHash
}

func (c *HTTPClient) PrintMsg(t *testing.T, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, err := c.cli.Debug.PrintMsg(ctx, msg)
	require.NoError(t, err, "failed to send Msg to %s node", c.impl.String())
}

func (c *HTTPClient) GetAssetDetails(assetID crypto.Digest) (*client.AssetsDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	details, _, err := c.cli.Assets.Details(ctx, assetID)
	return details, err
}

func (c *HTTPClient) TransactionInfo(t *testing.T, id crypto.Digest) proto.Transaction {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	info, _, err := c.cli.Transactions.Info(ctx, id)
	require.NoError(t, err, "failed to get TransactionInfo from %s node", c.impl.String())
	return info
}

func (c *HTTPClient) TransactionInfoRaw(id crypto.Digest) (proto.Transaction, *client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.cli.Transactions.Info(ctx, id)
}

func (c *HTTPClient) TransactionBroadcast(transaction proto.Transaction) (*client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.cli.Transactions.Broadcast(ctx, transaction)
}

func (c *HTTPClient) WavesBalance(t *testing.T, address proto.WavesAddress) *client.AddressesBalance {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	balance, _, err := c.cli.Addresses.Balance(ctx, address)
	require.NoError(t, err, "failed to get waves balance from %s node", c.impl.String())
	return balance
}

func (c *HTTPClient) AssetBalance(
	t *testing.T, address proto.WavesAddress, assetID crypto.Digest,
) *client.AssetsBalanceAndAsset {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	balance, _, err := c.cli.Assets.BalanceByAddressAndAsset(ctx, address, assetID)
	require.NoError(t, err, "failed to get asset balance from %s node", c.impl.String())
	return balance
}

func (c *HTTPClient) ConnectedPeers() ([]*client.PeersConnectedRow, *client.Response, error) {
	return c.ConnectedPeersCtx(context.Background())
}

func (c *HTTPClient) ConnectedPeersCtx(ctx context.Context) ([]*client.PeersConnectedRow, *client.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	connectedPeers, resp, err := c.cli.Peers.Connected(ctx)
	return connectedPeers, resp, err
}

func (c *HTTPClient) BlockHeader(t *testing.T, height proto.Height) *client.Headers {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	header, _, err := c.cli.Blocks.HeadersAt(ctx, height)
	require.NoError(t, err, "failed to get block header from %s node", c.impl.String())
	return header
}

func (c *HTTPClient) Rewards(t *testing.T) *client.RewardInfo {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	rewardInfo, _, err := c.cli.Blockchain.Rewards(ctx)
	require.NoError(t, err, "failed to get rewards from %s node", c.impl.String())
	return rewardInfo
}

func (c *HTTPClient) RewardsAtHeight(t *testing.T, height proto.Height) *client.RewardInfo {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	rewardInfo, _, err := c.cli.Blockchain.RewardsAtHeight(ctx, height)
	require.NoError(t, err, "failed to get rewards from %s node", c.impl.String())
	return rewardInfo
}

func (c *HTTPClient) RollbackToHeight(t *testing.T, height uint64, returnTxToUtx bool) *proto.BlockID {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	blockID, _, err := c.cli.Debug.RollbackToHeight(ctx, height, returnTxToUtx)
	require.NoError(t, err, "failed to rollback to height on %s node", c.impl.String())
	return blockID
}
