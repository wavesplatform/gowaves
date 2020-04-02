package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

type hitSourceKey struct {
	height uint64
}

func (k *hitSourceKey) bytes() []byte {
	buf := make([]byte, 9)
	buf[0] = hitSourceKeyPrefix
	binary.LittleEndian.PutUint64(buf[1:], k.height)
	return buf
}

type hitSources struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch
}

func newHitSources(db keyvalue.KeyValue, dbBatch keyvalue.Batch) (*hitSources, error) {
	return &hitSources{db: db, dbBatch: dbBatch}, nil
}

func (hss *hitSources) saveHitSource(hs []byte, height uint64) error {
	if len(hs) != 32 {
		return errInvalidDataSize
	}
	key := hitSourceKey{height: height}
	hss.dbBatch.Put(key.bytes(), hs)
	return nil
}

func (hss *hitSources) hitSource(height uint64) ([]byte, error) {
	key := hitSourceKey{height: height}
	hs, err := hss.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	return hs, nil
}

func (hss *hitSources) rollback(newHeight, oldHeight uint64) error {
	for h := oldHeight; h > newHeight; h-- {
		key := hitSourceKey{height: h}
		if err := hss.db.Delete(key.bytes()); err != nil {
			return err
		}
	}
	return nil
}
