package state

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	emptyAddress = proto.WavesAddress{}
)

type aliasKey struct {
	prefix byte
	alias  proto.Alias
}

func (k aliasKey) bytes() []byte {
	b := k.alias.Bytes()
	buf := make([]byte, 1+len(b))
	buf[0] = k.prefix
	copy(buf[1:], b)
	return buf
}

type aliasHistoryKey struct {
	prefix byte
	height uint32
	alias  proto.Alias
}

func (k aliasHistoryKey) bytes() []byte {
	b := k.alias.Bytes()
	buf := make([]byte, 1+4+len(b))
	buf[0] = k.prefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], b)
	return buf
}

type aliasChange struct {
	prev proto.WavesAddress
	curr proto.WavesAddress
}

func (c aliasChange) bytes() []byte {
	buf := make([]byte, 2*proto.WavesAddressSize)
	copy(buf, c.prev[:])
	copy(buf[proto.WavesAddressSize:], c.curr[:])
	return buf
}

func (c *aliasChange) fromBytes(data []byte) error {
	if l := len(data); l < 2*proto.WavesAddressSize {
		return errors.Errorf("%d is not enough bytes for aliasChange, expected %d", l, 2*proto.WavesAddressSize)
	}
	copy(c.prev[:], data[:proto.WavesAddressSize])
	copy(c.curr[:], data[proto.WavesAddressSize:2*proto.WavesAddressSize])
	return nil
}

func putAliases(bs *blockState, batch *leveldb.Batch, height uint32, binds []data.AliasBind) error {
	for _, bind := range binds {
		bk := aliasKey{aliasToAddressKeyPrefix, bind.Alias}
		batch.Put(bk.bytes(), bind.Address[:])
		pa, ok, err := bs.addressByAlias(bind.Alias)
		if err != nil {
			return errors.Wrap(err, "failed to updated aliases")
		}
		if !ok {
			pa = emptyAddress
		}
		ch := aliasChange{prev: pa, curr: bind.Address}
		hk := aliasHistoryKey{aliasHistoryKeyPrefix, height, bind.Alias}
		bs.aliasBindings[bind.Alias] = bind.Address
		batch.Put(hk.bytes(), ch.bytes())
	}
	return nil
}

func rollbackAliases(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	s := uint32Key{aliasHistoryKeyPrefix, removeHeight}
	l := uint32Key{aliasHistoryKeyPrefix, math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	remove := make([]proto.Alias, 0)
	downgrade := make(map[proto.Alias]proto.WavesAddress)
	if it.Last() {
		for {
			key := it.Key()
			ab := key[5:]
			ap, err := proto.NewAliasFromBytes(ab)
			a := *ap
			if err != nil {
				return errors.Wrap(err, "failed to rollback aliases")
			}
			var d aliasChange
			err = d.fromBytes(it.Value())
			if err != nil {
				return errors.Wrap(err, "failed to rollback aliases")
			}
			if !bytes.Equal(d.prev[:], emptyAddress[:]) {
				downgrade[a] = d.prev
			} else {
				remove = append(remove, a)
				delete(downgrade, a)
			}
			batch.Delete(key)
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	for al, ad := range downgrade {
		k := aliasKey{aliasToAddressKeyPrefix, al}
		batch.Put(k.bytes(), ad[:])
	}
	for _, a := range remove {
		k := aliasKey{aliasToAddressKeyPrefix, a}
		batch.Delete(k.bytes())
	}
	return nil
}
