package clients

import (
	"context"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type NodeUniversalClient struct {
	Implementation Implementation
	HTTPClient     *HTTPClient
	GRPCClient     *GRPCClient
	Connection     *NetClient
}

func NewNodeUniversalClient(
	ctx context.Context, t *testing.T, impl Implementation, httpPort, grpcPort, netPort string, peers []proto.PeerInfo,
) *NodeUniversalClient {
	return &NodeUniversalClient{
		Implementation: impl,
		HTTPClient:     NewHTTPClient(t, impl, httpPort),
		GRPCClient:     NewGRPCClient(t, impl, grpcPort),
		Connection:     NewNetClient(ctx, t, impl, netPort, peers),
	}
}

func (c *NodeUniversalClient) SendStartMessage(t *testing.T) {
	c.HTTPClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
}

func (c *NodeUniversalClient) SendEndMessage(t *testing.T) {
	c.HTTPClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
}

func (c *NodeUniversalClient) Handshake() {
	c.Connection.SendHandshake()
}

func (c *NodeUniversalClient) Close(t testing.TB) {
	c.GRPCClient.Close(t)
	c.Connection.Close()
}
