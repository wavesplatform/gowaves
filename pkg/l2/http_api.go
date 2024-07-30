package l2

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ybbus/jsonrpc/v3"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HttpAPIClient struct {
	rpcClient jsonrpc.RPCClient
}

type HttpAPIOpts struct {
	Address string
	Port    string
}

func NewHttpAPIClient(opts HttpAPIOpts) *HttpAPIClient {
	rpcClient := jsonrpc.NewClient("http://" + opts.Address + ":" + opts.Port)
	return &HttpAPIClient{
		rpcClient: rpcClient,
	}
}

func (c *HttpAPIClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByNumber", BigInt{Int: number}, false)
	if err != nil {
		return nil, fmt.Errorf("failed getting block by number %s: %v", number.String(), err)
	}
	return &response, nil
}

func (c *HttpAPIClient) GetLastExecutionBlock(ctx context.Context) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, fmt.Errorf("failed getting latest block: %v", err)
	}
	return &response, nil
}

func (c *HttpAPIClient) GetBlockByHash(ctx context.Context, hash proto.EthereumHash) (*EcBlock, error) {
	response := EcBlock{}
	err := c.rpcClient.CallFor(ctx, &response, "eth_getBlockByHash", hash, false)
	if err != nil {
		return nil, fmt.Errorf("failed getting latest block by hash %s: %v", hash.String(), err)
	}
	return &response, nil
}
