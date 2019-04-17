package mock

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"testing"
)

func TestCompile(t *testing.T) {
	var _ peer.Peer = &Peer{}
}
