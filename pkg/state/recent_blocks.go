package state

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type recentBlocks struct {
	rwLock sync.RWMutex

	stableHeight, newestHeight uint64
	rangeSize                  int
	// IDs of recent blocks in DB.
	stableIds []crypto.Signature
	isStable  map[crypto.Signature]uint64
	// IDs of recent blocks which have not been flushed to DB yet.
	newestIds []crypto.Signature
	isNewest  map[crypto.Signature]uint64

	rw *blockReadWriter
}

func newRecentBlocks(rangeSize int, rw *blockReadWriter) (*recentBlocks, error) {
	return &recentBlocks{
		rangeSize: rangeSize,
		isStable:  make(map[crypto.Signature]uint64),
		isNewest:  make(map[crypto.Signature]uint64),
		rw:        rw,
	}, nil
}

func (rb *recentBlocks) isEmpty() bool {
	rb.rwLock.RLock()
	defer rb.rwLock.RUnlock()
	if rb.stableIds == nil && rb.rw != nil {
		return true
	}
	return false
}

func (rb *recentBlocks) height() (uint64, error) {
	if rb.isEmpty() {
		if err := rb.fillRecentIds(); err != nil {
			return 0, err
		}
	}
	rb.rwLock.RLock()
	defer rb.rwLock.RUnlock()
	return rb.stableHeight, nil
}

// Add to the list of newest IDs.
func (rb *recentBlocks) addNewBlockID(blockID crypto.Signature) error {
	if rb.isEmpty() {
		if err := rb.fillRecentIds(); err != nil {
			return err
		}
	}
	rb.rwLock.Lock()
	defer rb.rwLock.Unlock()
	if rb.newestHeight == 0 {
		rb.newestHeight = rb.stableHeight
	}
	rb.isNewest[blockID] = rb.newestHeight
	rb.newestIds = append(rb.newestIds, blockID)
	rb.newestHeight++
	return nil
}

func (rb *recentBlocks) addBlockIDImpl(blockID crypto.Signature) error {
	if len(rb.stableIds) < rb.rangeSize {
		rb.isStable[blockID] = rb.stableHeight
		rb.stableIds = append(rb.stableIds, blockID)
	} else {
		rb.isStable[blockID] = rb.stableHeight
		delete(rb.isStable, rb.stableIds[0])
		rb.stableIds = rb.stableIds[1:]
		rb.stableIds = append(rb.stableIds, blockID)
	}
	rb.stableHeight++
	return nil
}

// Add directly to the list of stable IDs.
func (rb *recentBlocks) addBlockID(blockID crypto.Signature) error {
	rb.rwLock.Lock()
	defer rb.rwLock.Unlock()
	return rb.addBlockIDImpl(blockID)
}

func (rb *recentBlocks) fillRecentIds() error {
	rb.rwLock.Lock()
	defer rb.rwLock.Unlock()
	height, err := rb.rw.currentHeight()
	if err != nil {
		return err
	}
	start := uint64(1)
	if height > uint64(rb.rangeSize) {
		start = height - uint64(rb.rangeSize)
	}
	rb.stableHeight = start
	for h := start; h <= height; h++ {
		id, err := rb.rw.blockIDByHeight(h)
		if err != nil {
			return err
		}
		if err := rb.addBlockIDImpl(id); err != nil {
			return err
		}
	}
	return nil
}

func (rb *recentBlocks) newBlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if rb.isEmpty() {
		if err := rb.fillRecentIds(); err != nil {
			return 0, err
		}
	}
	rb.rwLock.RLock()
	defer rb.rwLock.RUnlock()
	height, ok := rb.isNewest[blockID]
	if !ok {
		height, ok = rb.isStable[blockID]
		if !ok {
			return 0, nil
		}
		return height, nil
	}
	return height, nil
}

func (rb *recentBlocks) blockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if rb.isEmpty() {
		if err := rb.fillRecentIds(); err != nil {
			return 0, err
		}
	}
	rb.rwLock.RLock()
	defer rb.rwLock.RUnlock()
	stableHeight, ok := rb.isStable[blockID]
	if !ok {
		return 0, nil
	}
	return stableHeight, nil
}

func (rb *recentBlocks) reset() {
	rb.rwLock.Lock()
	defer rb.rwLock.Unlock()
	rb.stableIds = nil
	rb.isStable = make(map[crypto.Signature]uint64)
	rb.newestHeight = 0
	rb.stableHeight = 0
	rb.newestIds = nil
	rb.isNewest = make(map[crypto.Signature]uint64)
}

func (rb *recentBlocks) removeOutdated(ids []crypto.Signature) {
	for _, id := range ids {
		delete(rb.isStable, id)
	}
}

func (rb *recentBlocks) addNewIds() {
	for id, height := range rb.isNewest {
		rb.isStable[id] = height
	}
}

// flush() "flushes" newest IDs to stable IDs.
func (rb *recentBlocks) flush() {
	rb.rwLock.Lock()
	defer rb.rwLock.Unlock()
	rb.stableIds = append(rb.stableIds, rb.newestIds...)
	rb.addNewIds()
	rb.newestIds = nil
	rb.isNewest = make(map[crypto.Signature]uint64)
	if len(rb.stableIds) > rb.rangeSize {
		rb.removeOutdated(rb.stableIds[:len(rb.stableIds)-rb.rangeSize])
		rb.stableIds = rb.stableIds[len(rb.stableIds)-rb.rangeSize:]
	}
	rb.stableHeight = rb.newestHeight
}
