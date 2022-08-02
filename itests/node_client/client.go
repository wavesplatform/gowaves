package node_client

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	goNodeClient    *Client
	scalaNodeClient *Client
)

type Client struct {
	cli *client.Client
}

func GoNodeClient(t *testing.T) *Client {
	if goNodeClient != nil {
		return goNodeClient
	}
	GoNodeClientRaw, err := client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + d.GoNodeRESTApiPort + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	assert.NoError(t, err, "couldn't create go node api client")
	goNodeClient = &Client{cli: GoNodeClientRaw}
	return goNodeClient
}

func ScalaNodeClient(t *testing.T) *Client {
	if scalaNodeClient != nil {
		return scalaNodeClient
	}
	ScalaNodeClientRaw, err := client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + d.ScalaNodeRESTApiPort + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	assert.NoError(t, err, "couldn't create scala node api client")
	scalaNodeClient = &Client{cli: ScalaNodeClientRaw}
	return scalaNodeClient
}

func (c *Client) GetHeight(t *testing.T, ctx context.Context) *client.BlocksHeight {
	h, _, err := c.cli.Blocks.Height(ctx)
	assert.NoError(t, err, "failed to get height from node")
	return h
}

func (c *Client) StateHash(t *testing.T, ctx context.Context, height uint64) *proto.StateHash {
	stateHash, _, err := c.cli.Debug.StateHash(ctx, height)
	assert.NoError(t, err, "failed to get stateHash from node")
	return stateHash
}

func (c *Client) PrintMsg(t *testing.T, ctx context.Context, msg string) {
	_, err := c.cli.Debug.PrintMsg(ctx, msg)
	assert.NoError(t, err, "failed to send Msg to node")
}

func (c *Client) TransactionInfo(t *testing.T, ctx context.Context, ID crypto.Digest) proto.Transaction {
	info, _, err := c.cli.Transactions.Info(ctx, ID)
	assert.NoError(t, err, "failed to get TransactionInfo from node")
	return info
}

func (c *Client) TransactionInfoRaw(ctx context.Context, ID crypto.Digest) (proto.Transaction, *client.Response, error) {
	return c.cli.Transactions.Info(ctx, ID)
}
