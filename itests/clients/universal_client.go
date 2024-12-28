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
