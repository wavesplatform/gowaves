package network

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type scoreSelector struct {
	m map[proto.Score][]peer.ID
}

func newScoreSelector() *scoreSelector {
	return &scoreSelector{}
}

func (s *scoreSelector) push(peer peer.ID, score *proto.Score) {

}

func (s *scoreSelector) pop() (peer.ID, *proto.Score) {
	return nil, nil
}
