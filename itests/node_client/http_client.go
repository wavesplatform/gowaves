package node_client

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HttpClient struct {
	cli *client.Client
}

func NewHttpClient(t *testing.T, port string) *HttpClient {
	c, err := client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + port + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	assert.NoError(t, err, "couldn't create go node api client")
	return &HttpClient{cli: c}
}

func (c *HttpClient) GetHeight(t *testing.T) *client.BlocksHeight {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	h, _, err := c.cli.Blocks.Height(ctx)
	assert.NoError(t, err, "failed to get height from node")
	return h
}

func (c *HttpClient) StateHash(t *testing.T, height uint64) *proto.StateHash {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	stateHash, _, err := c.cli.Debug.StateHash(ctx, height)
	assert.NoError(t, err, "failed to get stateHash from node")
	return stateHash
}

func (c *HttpClient) PrintMsg(t *testing.T, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_, err := c.cli.Debug.PrintMsg(ctx, msg)
	assert.NoError(t, err, "failed to send Msg to node")
}

func (c *HttpClient) TransactionInfo(t *testing.T, ID crypto.Digest) proto.Transaction {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	info, _, err := c.cli.Transactions.Info(ctx, ID)
	assert.NoError(t, err, "failed to get TransactionInfo from node")
	return info
}

func (c *HttpClient) TransactionInfoRaw(ID crypto.Digest) (proto.Transaction, *client.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	return c.cli.Transactions.Info(ctx, ID)
}
