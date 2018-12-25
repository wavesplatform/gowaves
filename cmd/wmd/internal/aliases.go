package internal

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
)

var (
	emptyAddress = proto.Address{}
)

type AliasKey proto.Alias

func (k AliasKey) key() []byte {
	a := proto.Alias(k)
	b, err := a.MarshalBinary()
	if err != nil {
		return nil
	}
	buf := make([]byte, 1+len(b))
	buf[0] = AliasToAddressKeyPrefix
	copy(buf[1:], b)
	return buf
}

type AliasBind struct {
	Alias   proto.Alias
	Address proto.Address
}

func (b AliasBind) key() []byte {
	return AliasKey(b.Alias).key()
}

func (b AliasBind) value() []byte {
	return b.Address[:]
}

type AliasDiff struct {
	Prev proto.Address
	Curr proto.Address
}

func (d AliasDiff) bytes() []byte {
	buf := make([]byte, 2*proto.AddressSize)
	copy(buf, d.Prev[:])
	copy(buf[proto.AddressSize:], d.Curr[:])
	return buf
}

func (d *AliasDiff) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 2*proto.AddressSize {
		return errors.Errorf("%d is not enough bytes for AliasDiff, expected %d", l, 2*proto.AddressSize)
	}
	copy(d.Prev[:], data[:proto.AddressSize])
	copy(d.Curr[:], data[proto.AddressSize:2*proto.AddressSize])
	return nil
}

func PutAliasesStateUpdate(snapshot *leveldb.Snapshot, batch *leveldb.Batch, height uint32, binds []AliasBind) error {
	hk := uint32Key(AliasHistoryKeyPrefix, height)
	hkl := len(hk)
	for _, b := range binds {
		bk := b.key()
		batch.Put(bk, b.value())
		var ea proto.Address
		eab, err := snapshot.Get(bk, defaultReadOptions)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return errors.Wrap(err, "failed to update state of aliases")
			}
			ea = emptyAddress
		} else {
			ea, err = proto.NewAddressFromBytes(eab)
			if err != nil {
				return errors.Wrap(err, "failed to update state of aliases")
			}
		}
		ab, err := b.Alias.MarshalBinary()
		if err != nil {
			return errors.Wrap(err, "failed to update state of aliases")
		}
		k := make([]byte, hkl+len(ab))
		copy(k, hk)
		copy(k[hkl:], ab)
		ch := AliasDiff{Prev: ea, Curr: b.Address}
		batch.Put(k, ch.bytes())
	}
	return nil
}

func RollbackAliases(snapshot *leveldb.Snapshot, batch *leveldb.Batch, height uint32) error {
	it := snapshot.NewIterator(&util.Range{Start: uint32Key(AliasHistoryKeyPrefix, height), Limit: uint32Key(AliasHistoryKeyPrefix, math.MaxUint32)}, defaultReadOptions)
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
			var d AliasDiff
			err = d.UnmarshalBinary(it.Value())
			if err != nil {
				return errors.Wrap(err, "failed to rollback aliases")
			}
			if !bytes.Equal(d.Prev[:], emptyAddress[:]) {
				downgrade[a] = d.Prev
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
		batch.Delete(AliasKey(a).key())
	}
	for al, ad := range downgrade {
		batch.Put(AliasKey(al).key(), ad[:])
	}
	return it.Error()
}

func GetAddress(snapshot *leveldb.Snapshot, alias proto.Alias) (proto.Address, error) {
	bind := AliasBind{Alias: alias}
	ab, err := snapshot.Get(bind.key(), defaultReadOptions)
	if err != nil {
		return proto.Address{}, errors.Wrap(err, "failed to get an Address")
	}
	var r proto.Address
	copy(r[:], ab)
	if _, err := r.Validate(); err != nil {
		return proto.Address{}, errors.Wrap(err, "failed to get an Address")
	}
	return r, nil
}
