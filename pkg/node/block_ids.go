package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockIDs []proto.BlockID

// reverse creates new blockIDs sequence with a reverse order.
func (ids blockIDs) reverse() blockIDs {
	l := len(ids)
	r := make(blockIDs, len(ids))
	for i := range ids {
		r[l-1-i] = ids[i]
	}
	return r
}
