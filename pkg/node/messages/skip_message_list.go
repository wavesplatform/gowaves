package messages

import (
	"sync/atomic"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SkipMessageList struct {
	list atomic.Value
}

func (l *SkipMessageList) List() proto.PeerMessageIDs {
	list := l.list.Load()
	if list == nil {
		return nil
	}
	return list.(proto.PeerMessageIDs)
}

func (l *SkipMessageList) SetList(list proto.PeerMessageIDs) {
	l.list.Store(list)
}
