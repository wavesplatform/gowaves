package state

const hitSourceSize = 32

type hitSources struct {
	hs *historyStorage
	rw *blockReadWriter
}

func newHitSources(hs *historyStorage, rw *blockReadWriter) *hitSources {
	return &hitSources{hs, rw}
}

func (hss *hitSources) saveHitSource(hs []byte, height uint64) error {
	if len(hs) != hitSourceSize {
		return errInvalidDataSize
	}
	blockID, err := hss.rw.newestBlockIDByHeight(height)
	if err != nil {
		return err
	}
	key := hitSourceKey{height: height}
	return hss.hs.addNewEntry(hitSource, key.bytes(), hs, blockID)
}

func (hss *hitSources) hitSource(height uint64, filter bool) ([]byte, error) {
	key := hitSourceKey{height: height}
	hs, err := hss.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	if len(hs) != hitSourceSize {
		return nil, errInvalidDataSize
	}
	return hs, nil
}
