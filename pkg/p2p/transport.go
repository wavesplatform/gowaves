package p2p

import (
	"context"
	"net"
	"time"
)

// DefaultTransport implements the default dialing strategy
var DefaultTransport = Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
}

// Transport allows one to reimplement the way we dial a peer
type Transport struct {
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
}
