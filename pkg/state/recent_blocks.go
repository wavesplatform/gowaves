package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type recentBlocks struct {
	ids []crypto.Signature
}

func newRecentBlocks() (*recentBlocks, error) {
	return &recentBlocks{}, nil
}

func (rb *recentBlocks) addBlockID(blockID crypto.Signature) error {
	if len(rb.ids) < rollbackMaxBlocks {
		rb.ids = append(rb.ids, blockID)
	} else {
		rb.ids = rb.ids[1:]
		rb.ids = append(rb.ids, blockID)
	}
	return nil
}

func (rb *recentBlocks) blockIsRecent(blockID crypto.Signature, bottomLimit int) (bool, error) {
	if bottomLimit > len(rb.ids) {
		return false, errors.New("bottomLimit is too large")
	}
	recent := rb.ids[len(rb.ids)-bottomLimit:]
	for _, id := range recent {
		if id == blockID {
			return true, nil
		}
	}
	return false, nil
}

func (rb *recentBlocks) isEmpty() bool {
	return rb.ids == nil
}

func (rb *recentBlocks) reset() {
	rb.ids = nil
}
