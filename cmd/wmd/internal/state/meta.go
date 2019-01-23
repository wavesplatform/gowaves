package state

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type heightKey struct{}

func (k *heightKey) bytes() []byte {
	buf := make([]byte, 1)
	buf[0] = heightKeyPrefix
	return buf
}

type blockInfoKey struct {
	height uint32
}

func (k *blockInfoKey) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = blockInfoKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	return buf
}

type blockInfo struct {
	block             crypto.Signature
	empty             bool
	earliestTimeFrame uint32
}

func (b *blockInfo) bytes() []byte {
	buf := make([]byte, crypto.SignatureSize+1+4)
	copy(buf, b.block[:])
	if b.empty {
		buf[crypto.SignatureSize] = 1
	} else {
		buf[crypto.SignatureSize] = 0
	}
	binary.BigEndian.PutUint32(buf[crypto.SignatureSize+1:], b.earliestTimeFrame)
	return buf
}

func (b *blockInfo) fromBytes(data []byte) error {
	if l := len(data); l < crypto.SignatureSize+1+4 {
		return errors.Errorf("%d is not enough bytes for for blockInfo", l)
	}
	copy(b.block[:], data[:crypto.SignatureSize])
	b.empty = data[crypto.SignatureSize] == 1
	b.earliestTimeFrame = binary.BigEndian.Uint32(data[crypto.SignatureSize+1:])
	return nil
}

func putBlock(bs *blockState, batch *leveldb.Batch, height uint32, block crypto.Signature) error {
	//TODO: implement
	return nil
}

func rollbackBlocks(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	//TODO: implement
	return nil
}

func height(snapshot *leveldb.Snapshot) (int, error) {
	k := heightKey{}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get current height")
	}
	h := int(binary.BigEndian.Uint32(b))
	return h, nil
}

func block(snapshot *leveldb.Snapshot, height uint32) (blockInfo, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to locate block at height %d", height) }
	k := blockInfoKey{height}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		return blockInfo{}, wrapError(err)
	}
	var bi blockInfo
	err = bi.fromBytes(b)
	if err != nil {
		return blockInfo{}, wrapError(err)
	}
	return bi, nil
}

func hasBlock(snapshot *leveldb.Snapshot, height uint32, id crypto.Signature) (bool, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to locate block '%s' at height %d", id.String(), height) }
	k := blockInfoKey{height}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		return false, wrapError(err)
	}
	var bi blockInfo
	err = bi.fromBytes(b)
	if err != nil {
		return false, wrapError(err)
	}
	if !bytes.Equal(id[:], bi.block[:]) {
		return false, wrapError(errors.Errorf("different block signature '%s' at height %d", bi.block.String(), height))
	}
	return true, nil
}
