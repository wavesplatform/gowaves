package l2

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ybbus/jsonrpc/v3"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HTTPAPIClient struct {
	rpcClient jsonrpc.RPCClient
}

type HTTPAPIOpts struct {
	Address string
	Port    string
}

func NewHTTPAPIClient(opts HTTPAPIOpts) *HTTPAPIClient {
	rpcClient := jsonrpc.NewClient("http://" + opts.Address + ":" + opts.Port)
	return &HTTPAPIClient{
		rpcClient: rpcClient,
	}
}

func (c *HTTPAPIClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByNumber", BigInt{Int: number}, false)
	if err != nil {
		return nil, fmt.Errorf("failed getting block by number %s: %w", number.String(), err)
	}
	return &response, nil
}

func (c *HTTPAPIClient) GetLastExecutionBlock(ctx context.Context) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, fmt.Errorf("failed getting latest block: %w", err)
	}
	return &response, nil
}

func (c *HTTPAPIClient) GetBlockByHash(ctx context.Context, hash proto.EthereumHash) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByHash", hash, false)
	if err != nil {
		return nil, fmt.Errorf("failed getting block by hash %s: %w", hash.String(), err)
	}
	return &response, nil
}
