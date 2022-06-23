package mock

import (
	"testing"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

func TestCompile(t *testing.T) {
	var _ peer.Peer = &Peer{}
}
