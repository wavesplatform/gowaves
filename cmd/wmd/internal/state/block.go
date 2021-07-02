package state

import (
	"encoding/binary"
	"math"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockState struct {
	snapshot        *leveldb.Snapshot
	aliasBindings   map[proto.Alias]proto.WavesAddress
	balances        map[assetBalanceKey]uint64
	issuers         map[assetIssuerKey]struct{}
	assets          map[assetKey]asset
	candles         map[candleKey]data.Candle
	markets         map[marketKey]data.Market
	earliestHeights map[uint32Key]uint32
}

func newBlockState(snapshot *leveldb.Snapshot) *blockState {
	return &blockState{
		snapshot:        snapshot,
		aliasBindings:   make(map[proto.Alias]proto.WavesAddress),
		balances:        make(map[assetBalanceKey]uint64),
		issuers:         make(map[assetIssuerKey]struct{}),
		assets:          make(map[assetKey]asset),
		candles:         make(map[candleKey]data.Candle),
		markets:         make(map[marketKey]data.Market),
		earliestHeights: make(map[uint32Key]uint32),
	}
}

func (s *blockState) addressByAlias(alias proto.Alias) (proto.WavesAddress, bool, error) {
	var a proto.WavesAddress
	a, ok := s.aliasBindings[alias]
	if !ok {
		k := aliasKey{prefix: aliasToAddressKeyPrefix, alias: alias}
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return emptyAddress, false, err
			}
			return emptyAddress, false, nil
		}
		a, err = proto.NewAddressFromBytes(b)
		if err != nil {
			return a, false, err
		}
	}
	return a, true, nil
}

func (s *blockState) isIssuer(address proto.WavesAddress, asset crypto.Digest) (bool, error) {
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

func (s *blockState) balance(address proto.WavesAddress, asset crypto.Digest) (uint64, assetBalanceKey, error) {
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

func (s *blockState) assetInfo(assetID crypto.Digest) (asset, bool, error) {
	k := assetKey{asset: assetID}
	var a asset
	a, ok := s.assets[k]
	if !ok {
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return asset{}, false, err
			}
			return asset{}, false, nil
		}
		err = a.fromBytes(b)
		if err != nil {
			return asset{}, false, err
		}
	}
	return a, true, nil
}

func (s *blockState) candle(amountAsset, priceAsset crypto.Digest, timeFrame uint32) (data.Candle, candleKey, error) {
	k := candleKey{amountAsset: amountAsset, priceAsset: priceAsset, timeFrame: timeFrame}
	var c data.Candle
	c, ok := s.candles[k]
	if !ok {
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return data.Candle{}, k, err
			}
			c = data.NewCandleFromTimeFrame(timeFrame)
			return c, k, nil
		}
		err = c.UnmarshalBinary(b)
		if err != nil {
			return data.Candle{}, k, err
		}
	}
	return c, k, nil
}

func (s *blockState) market(amountAsset, priceAsset crypto.Digest) (data.Market, marketKey, error) {
	k := marketKey{amountAsset: amountAsset, priceAsset: priceAsset}
	var m data.Market
	m, ok := s.markets[k]
	if !ok {
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return data.Market{}, k, err
			}
			return m, k, nil
		}
		err = m.UnmarshalBinary(b)
		if err != nil {
			return data.Market{}, k, err
		}
	}
	return m, k, nil
}

func (s *blockState) earliestHeight(timeFrame uint32) (uint32, uint32Key, error) {
	k := uint32Key{prefix: earliestHeightKeyPrefix, key: timeFrame}
	var eh uint32
	eh, ok := s.earliestHeights[k]
	if !ok {
		b, err := s.snapshot.Get(k.bytes(), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return 0, k, err
			}
			return math.MaxUint32, k, nil
		}
		eh = binary.BigEndian.Uint32(b)
	}
	return eh, k, nil
}
