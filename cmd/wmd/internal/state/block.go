package state

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockState struct {
	snapshot      *leveldb.Snapshot
	aliasBindings map[proto.Alias]proto.Address
	balances      map[assetBalanceKey]uint64
	issuers       map[assetIssuerKey]struct{}
	assets        map[assetInfoKey]assetInfo
}

func newBlockState(snapshot *leveldb.Snapshot) *blockState {
	return &blockState{
		snapshot:      snapshot,
		aliasBindings: make(map[proto.Alias]proto.Address),
		balances:      make(map[assetBalanceKey]uint64),
		issuers:       make(map[assetIssuerKey]struct{}),
		assets:        make(map[assetInfoKey]assetInfo),
	}
}

func (s *blockState) addressByAlias(alias proto.Alias) (proto.Address, bool, error) {
	var a proto.Address
	a, ok := s.aliasBindings[alias]
	if !ok {
		k := aliasKey{prefix: AliasToAddressKeyPrefix, alias: alias}
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return EmptyAddress, false, err
			}
			return EmptyAddress, false, nil
		}
		a, err = proto.NewAddressFromBytes(b)
		if err != nil {
			return a, false, err
		}
	}
	return a, true, nil
}

func (s *blockState) isIssuer(address proto.Address, asset crypto.Digest) (bool, error) {
	k := assetIssuerKey{address: address, asset: asset}
	_, ok := s.issuers[k]
	if !ok {
		_, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return false, err
			}
			return false, nil
		}
	}
	return true, nil
}

func (s *blockState) balance(address proto.Address, asset crypto.Digest) (uint64, assetBalanceKey, error) {
	k := assetBalanceKey{address: address, asset: asset}
	var b uint64
	b, ok := s.balances[k]
	if !ok {
		bb, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return 0, k, err
			}
			return 0, k, nil
		}
		b = binary.BigEndian.Uint64(bb)
	}
	return b, k, nil
}

func (s *blockState) assetInfo(asset crypto.Digest) (assetInfo, bool, error) {
	k := assetInfoKey{asset: asset}
	var ai assetInfo
	ai, ok := s.assets[k]
	if !ok {
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return assetInfo{}, false, err
			}
			return assetInfo{}, false, nil
		}
		err = ai.fromBytes(b)
		if err != nil {
			return assetInfo{}, false, err
		}

	}
	return ai, true, nil
}
