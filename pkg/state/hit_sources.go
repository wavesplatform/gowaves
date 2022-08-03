package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const hitSourceSize = 32

type hitSources struct {
	hs *historyStorage
}

func newHitSources(hs *historyStorage) *hitSources {
	return &hitSources{hs: hs}
}

func (hss *hitSources) appendBlockHitSource(block *proto.Block, blockHeight uint64, hs []byte) error {
	if len(hs) != hitSourceSize {
		return errInvalidDataSize
	}
	key := hitSourceKey{height: blockHeight}
	return hss.hs.addNewEntry(hitSource, key.bytes(), hs, block.BlockID())
}

func (hss *hitSources) hitSource(height uint64) ([]byte, error) {
	key := hitSourceKey{height: height}
	hs, err := hss.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	if len(hs) != hitSourceSize {
		return nil, errInvalidDataSize
	}
	return hs, nil
}

func (hss *hitSources) newestHitSource(height uint64) ([]byte, error) {
	key := hitSourceKey{height: height}
	hs, err := hss.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	if len(hs) != hitSourceSize {
		return nil, errInvalidDataSize
	}
	return hs, nil
}
