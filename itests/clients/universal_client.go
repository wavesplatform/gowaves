package clients

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type NodeUniversalClient struct {
	Implementation Implementation
	HTTPClient     *HTTPClient
	GRPCClient     *GRPCClient
	Connection     *NetClient
}

func NewNodeUniversalClient(
	ctx context.Context, t *testing.T, impl Implementation, httpPort, grpcPort, netAddress string, peers []proto.PeerInfo,
) *NodeUniversalClient {
	return &NodeUniversalClient{
		Implementation: impl,
		HTTPClient:     NewHTTPClient(t, impl, httpPort),
		GRPCClient:     NewGRPCClient(t, impl, grpcPort),
		Connection:     NewNetClient(ctx, t, impl, netAddress, peers),
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

func (c *NodeUniversalClient) WaitForHeight(t testing.TB, height uint64, opts ...config.WaitOption) uint64 {
	var h uint64
	params := config.NewWaitParams(opts...)
	ctx, cancel := context.WithTimeout(params.Ctx, params.Timeout)
	defer cancel()
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for gCtx.Err() == nil {
			h = c.HTTPClient.GetHeight(t).Height
			if h >= height {
				return nil
			}
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-time.After(time.Second):
				// Sleep for a second before checking the height again.
			}
		}
		return gCtx.Err()
	})
	// Wait for the goroutines to finish.
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Failed to wait for height %d: %v", height, err)
	}
	return h
}

func (c *NodeUniversalClient) WaitForTransaction(t testing.TB, id crypto.Digest, opts ...config.WaitOption) {
	params := config.NewWaitParams(opts...)
	ctx, cancel := context.WithTimeout(params.Ctx, params.Timeout)
	defer cancel()
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return RetryCtx(gCtx, params.Timeout, func() error {
			_, _, err := c.HTTPClient.TransactionInfoRaw(id)
			return err
		})
	})
	// Wait for the goroutines to finish.
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Failed to wait for transaction %q: %v", id.String(), err)
	}
}
