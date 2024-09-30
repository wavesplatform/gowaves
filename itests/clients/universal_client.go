package clients

import (
	"testing"
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

func (c *NodeUniversalClient) SendStartMessage(t *testing.T) {
	c.HTTPClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
}

func (c *NodeUniversalClient) SendEndMessage(t *testing.T) {
	c.HTTPClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
}
