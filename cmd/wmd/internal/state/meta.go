package state

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var heightKeyBytes = []byte{heightKeyPrefix}

func putBlock(batch *leveldb.Batch, height uint32, block proto.BlockID) error {
	updateHeight(batch, height)
	k := uint32Key{prefix: blockKeyPrefix, key: height}
	batch.Put(k.bytes(), block.Bytes())
	return nil
}

func rollbackBlocks(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	s := uint32Key{prefix: blockKeyPrefix, key: removeHeight}
	l := uint32Key{prefix: blockKeyPrefix, key: math.MaxUint32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	for it.Next() {
		batch.Delete(it.Key())
	}
	it.Release()
	updateHeight(batch, removeHeight-1)
	return nil
}

func updateHeight(batch *leveldb.Batch, height uint32) {
	hv := make([]byte, 4)
	binary.BigEndian.PutUint32(hv, height)
	batch.Put(heightKeyBytes, hv)
}

func height(snapshot *leveldb.Snapshot) (int, error) {
	b, err := snapshot.Get(heightKeyBytes, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, errors.Wrap(err, "failed to get current height")
	}
	h := int(binary.BigEndian.Uint32(b))
	return h, nil
}

func block(snapshot *leveldb.Snapshot, height uint32) (proto.BlockID, bool, error) {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to locate block at height %d", height)
	}
	k := uint32Key{prefix: blockKeyPrefix, key: height}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return proto.BlockID{}, false, wrapError(err)
		}
		return proto.BlockID{}, false, nil
	}
	bid, err := proto.NewBlockIDFromBytes(b)
	if err != nil {
		return proto.BlockID{}, false, wrapError(err)
	}
	return bid, true, nil
}

func hasBlock(snapshot *leveldb.Snapshot, height uint32, id proto.BlockID) (bool, error) {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to locate block '%s' at height %d", id.String(), height)
	}
	b, ok, err := block(snapshot, height)
	if err != nil {
		return false, wrapError(err)
	}
	if ok {
		if id != b {
			return false, wrapError(errors.Errorf("different block id '%s' at height %d", b.String(), height))
		}
		return true, nil
	}
	return false, nil
}
