package node_client

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HttpClient struct {
	cli     *client.Client
	timeout time.Duration
}

func NewHttpClient(t *testing.T, port string) *HttpClient {
	c, err := client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + port + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	require.NoError(t, err, "couldn't create go node api client")
	return &HttpClient{
		cli: c,
		// actually, there's no need to use such timeout because above we've already set default context for http client
		timeout: 15 * time.Second,
	}
}

func (c *HttpClient) GetHeight(t *testing.T) *client.BlocksHeight {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	h, _, err := c.cli.Blocks.Height(ctx)
	require.NoError(t, err, "failed to get height from node")
	return h
}

func (c *HttpClient) StateHash(t *testing.T, height uint64) *proto.StateHash {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	stateHash, _, err := c.cli.Debug.StateHash(ctx, height)
	require.NoError(t, err, "failed to get stateHash from node")
	return stateHash
}

func (c *HttpClient) PrintMsg(t *testing.T, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, err := c.cli.Debug.PrintMsg(ctx, msg)
	require.NoError(t, err, "failed to send Msg to node")
}

func (c *HttpClient) TransactionInfo(t *testing.T, ID crypto.Digest) proto.Transaction {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	info, _, err := c.cli.Transactions.Info(ctx, ID)
	require.NoError(t, err, "failed to get TransactionInfo from node")
	return info
}

func (c *HttpClient) TransactionInfoRaw(id crypto.Digest) (proto.Transaction, *client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.cli.Transactions.Info(ctx, id)
}

func (c *HttpClient) TransactionBroadcast(transaction proto.Transaction) (*client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.cli.Transactions.Broadcast(ctx, transaction)
}

func (c *HttpClient) WavesBalance(t *testing.T, address proto.WavesAddress) *client.AddressesBalance {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	balance, _, err := c.cli.Addresses.Balance(ctx, address)
	require.NoError(t, err)
	return balance
}

func (c *HttpClient) AssetBalance(t *testing.T, address proto.WavesAddress, assetId crypto.Digest) *client.AssetsBalanceAndAsset {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	balance, _, err := c.cli.Assets.BalanceByAddressAndAsset(ctx, address, assetId)
	require.NoError(t, err)
	return balance
}

func (c *HttpClient) ConnectedPeers(t *testing.T) ([]*client.PeersConnectedRow, *client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	connectedPeers, resp, err := c.cli.Peers.Connected(ctx)
	return connectedPeers, resp, err
}
