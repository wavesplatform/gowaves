package messages

import (
	"sync/atomic"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SkipMessageList struct {
	list atomic.Value
}

func (l *SkipMessageList) List() []proto.PeerMessageID {
	list := l.list.Load()
	if list == nil {
		return nil
	}
	return list.([]proto.PeerMessageID)
}

func (l *SkipMessageList) ignore(ids ...proto.PeerMessageID) {
	l.list.Store(ids)
}

func (l *SkipMessageList) DisableEverything() {
	l.ignore(
		proto.ContentIDGetPeers,
		proto.ContentIDPeers,
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDScore,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDGetBlockIDs,
		proto.ContentIDBlockIDs,
	)
}

func (l *SkipMessageList) DisableForIdle() {
	l.ignore(
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDGetBlockIDs,
		proto.ContentIDBlockIDs,
	)
}

func (l *SkipMessageList) DisableForOperation() {
	l.ignore(
		proto.ContentIDSignatures,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDBlockIDs,
	)
}

func (l *SkipMessageList) DisableForOperationNG() {
	l.ignore(
		proto.ContentIDSignatures,
		proto.ContentIDCheckpoint,
		proto.ContentIDBlockIDs,
	)
}

func (l *SkipMessageList) DisableForSync() {
	l.ignore(
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBTransaction,
		proto.ContentIDPBMicroBlock,
	)
}
