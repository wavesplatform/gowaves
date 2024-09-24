package clients

import (
	"testing"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type NodeUniversalClient struct {
	Implementation Implementation
	HTTPClient     *HTTPClient
	GRPCClient     *GRPCClient
}

func NewNodeUniversalClient(t *testing.T, impl Implementation, httpPort string, grpcPort string) *NodeUniversalClient {
	return &NodeUniversalClient{
		Implementation: impl,
		HTTPClient:     NewHTTPClient(t, impl, httpPort),
		GRPCClient:     NewGRPCClient(t, impl, grpcPort),
	}
}

func (c *NodeUniversalClient) WaitForHeight(t *testing.T, height proto.Height) proto.Height {
	var h proto.Height
	for {
		h = c.HTTPClient.GetHeight(t).Height
		if h >= height {
			break
		}
		time.Sleep(time.Second * 1)
	}
	return h
}
