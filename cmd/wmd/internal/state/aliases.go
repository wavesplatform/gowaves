package state

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
)

var (
	EmptyAddress   = proto.Address{}
	//EmptyPublicKey = crypto.PublicKey{}
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
	prev proto.Address
	curr proto.Address
}

func (c aliasChange) bytes() []byte {
	buf := make([]byte, 2*proto.AddressSize)
	copy(buf, c.prev[:])
	copy(buf[proto.AddressSize:], c.curr[:])
	return buf
}

func (c *aliasChange) fromBytes(data []byte) error {
	if l := len(data); l < 2*proto.AddressSize {
		return errors.Errorf("%d is not enough bytes for aliasChange, expected %d", l, 2*proto.AddressSize)
	}
	copy(c.prev[:], data[:proto.AddressSize])
	copy(c.curr[:], data[proto.AddressSize:2*proto.AddressSize])
	return nil
}

func putAliasesStateUpdate(bs *blockState, batch *leveldb.Batch, height uint32, binds []AliasBind) error {
	for _, bind := range binds {
		bk := aliasKey{AliasToAddressKeyPrefix, bind.Alias}
		batch.Put(bk.bytes(), bind.Address[:])
		pa, ok, err := bs.addressByAlias(bind.Alias)
		if err != nil {
			return errors.Wrap(err, "failed to updated aliases")
		}
		if !ok {
			pa = EmptyAddress
		}
		ch := aliasChange{prev: pa, curr: bind.Address}
		hk := aliasHistoryKey{AliasHistoryKeyPrefix, height, bind.Alias}
		bs.aliasBindings[bind.Alias] = bind.Address
		batch.Put(hk.bytes(), ch.bytes())
	}
	return nil
}

func rollbackAliases(snapshot *leveldb.Snapshot, batch *leveldb.Batch, height uint32) error {
	s := uint32Key{AliasHistoryKeyPrefix, height}
	l := uint32Key{AliasHistoryKeyPrefix, math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	remove := make([]proto.Alias, 0)
	downgrade := make(map[proto.Alias]proto.Address)
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
			if !bytes.Equal(d.prev[:], EmptyAddress[:]) {
				downgrade[a] = d.prev
			} else {
				remove = append(remove, a)
			}
			batch.Delete(key)
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	for _, a := range remove {
		k := aliasKey{AliasToAddressKeyPrefix, a}
		batch.Delete(k.bytes())
	}
	for al, ad := range downgrade {
		k := aliasKey{AliasToAddressKeyPrefix, al}
		batch.Put(k.bytes(), ad[:])
	}
	return it.Error()
}
