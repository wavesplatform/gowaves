package state

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
	"time"
)

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type Storage struct {
	Path   string
	Scheme byte
	db     *leveldb.DB
}

func (s *Storage) Open() error {
	o := &opt.Options{}
	db, err := leveldb.OpenFile(s.Path, o)
	if err != nil {
		return errors.Wrap(err, "failed to open Storage")
	}
	s.db = db
	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

const (
	maxLimit = 1000
)

func (s *Storage) PutTrades(height int, block crypto.Signature, trades []data.Trade) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store trades of block '%s' at height %d", block.String(), height)
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	batch := new(leveldb.Batch)
	err = putTrades(bs, batch, uint32(height), trades)
	if err != nil {
		return wrapError(err)
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (s *Storage) PutBalances(height int, block crypto.Signature, issues []data.IssueChange, assets []data.AssetChange, accounts []data.AccountChange, aliases []data.AliasBind) error {
	h := uint32(height)
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	batch := new(leveldb.Batch)
	err = putAliases(bs, batch, h, aliases)
	if err != nil {
		return wrapError(err)
	}
	err = putIssues(bs, batch, s.Scheme, h, issues)
	if err != nil {
		return wrapError(err)
	}
	err = putAssets(bs, batch, h, assets)
	if err != nil {
		return wrapError(err)
	}
	err = putAccounts(bs, batch, h, accounts)
	if err != nil {
		return wrapError(err)
	}
	err = putBlock(batch, h, block)
	if err != nil {
		return wrapError(err)
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil

	//
	//	assets, err := s.loadAssetStates(stateUpdates)
	//	if err != nil {
	//		return wrapError(err)
	//	}
	//
	//	candles := map[timeFramePairKey]Candle{}
	//	markets := map[pairKey]Market{}
	//	affectedTimeFrames := make([]uint32, 0)
	//	for _, t := range trades {
	//		tf := TimeFrameFromTimestampMS(t.Timestamp)
	//		if !contains(affectedTimeFrames, tf) {
	//			affectedTimeFrames = append(affectedTimeFrames, tf)
	//		}
	//		ck := timeFramePairKey{CandlesKeyPrefix, tf, t.AmountAsset, t.PriceAsset}
	//		var c Candle
	//		c, ok := candles[ck]
	//		if !ok {
	//			cb, err := s.db.Get(ck.bytes(), defaultReadOptions)
	//			if err != nil {
	//				if err != leveldb.ErrNotFound {
	//					return wrapError(err)
	//				}
	//				c = NewCandleFromTimestamp(t.Timestamp)
	//			} else {
	//				err = c.UnmarshalBinary(cb)
	//				if err != nil {
	//					return wrapError(err)
	//				}
	//			}
	//		}
	//		c.UpdateFromTrade(t)
	//		candles[ck] = c
	//		mk := pairKey{marketKeyPrefix, t.AmountAsset, t.PriceAsset}
	//		m, ok := markets[mk]
	//		if !ok {
	//			mb, err := s.db.Get(mk.bytes(), defaultReadOptions)
	//			if err != nil {
	//				if err != leveldb.ErrNotFound {
	//					return wrapError(err)
	//				}
	//				m = Market{}
	//			} else {
	//				err = m.UnmarshalBinary(mb)
	//				if err != nil {
	//					return wrapError(err)
	//				}
	//			}
	//		}
	//		m.UpdateFromTrade(t)
	//		markets[mk] = m
	//	}
	//
	//	batch := new(leveldb.Batch)
	//	//putAliases(snapshot, batch, uint32(height), aliasBinds)
	//	updateAssets(assets, batch, height, stateUpdates)
	//	n := len(trades)
	//	updateLastHeight(batch, height)
	//	mtf := min(affectedTimeFrames)
	//	putBlockInfo(batch, height, block, n == 0, mtf)
	//	putTradesIDs(batch, block, trades)
	//	err = s.putTrades(batch, trades)
	//	if err != nil {
	//		return wrapError(err)
	//	}
	//	err = s.updateAffectedTimeFrames(batch, affectedTimeFrames, height)
	//	if err != nil {
	//		return wrapError(err)
	//	}
	//	for _, t := range trades {
	//		k1 := DigestKey{tradeKeyPrefix, t.TransactionID}
	//		v1 := make([]byte, TradeSize)
	//		v1, err := t.MarshalBinary()
	//		if err != nil {
	//			return wrapError(err)
	//		}
	//		batch.Put(k1.bytes(), v1)
	//	}
	//	for ck, cv := range candles {
	//		v, err := cv.MarshalBinary()
	//		if err != nil {
	//			return wrapError(err)
	//		}
	//		ckb := ck.bytes()
	//		batch.Put(ckb, v)
	//		ck2 := candleKey{candleKeyPrefix, ck.amountAsset, ck.priceAsset, ck.tf}
	//		batch.Put(ck2.bytes(), ckb)
	//	}
	//	for mk, mv := range markets {
	//		vb, err := mv.MarshalBinary()
	//		if err != nil {
	//			return wrapError(err)
	//		}
	//		batch.Put(mk.bytes(), vb)
	//	}
	//	err = s.db.Write(batch, defaultWriteOptions)
	//	if err != nil {
	//		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	//	}
}

func (s *Storage) SafeRollbackHeight(height int) (int, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	tf, ok := earliestTimeFrame(snapshot, uint32(height))
	if !ok {
		return height, nil
	}
	eh, err := earliestAffectedHeight(snapshot, tf)
	if err != nil {
		return 0, err
	}
	if int(eh) >= height {
		return height, nil
	} else {
		return s.SafeRollbackHeight(int(eh))
	}
}

func (s *Storage) Rollback(removeHeight int) error {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to rollback to height %d", removeHeight) }
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer snapshot.Release()
	max, err := height(snapshot)
	if err != nil {
		return wrapError(err)
	}
	if removeHeight >= max {
		return wrapError(errors.Errorf("nothing to rollback, current height is %d", max))
	}
	batch := new(leveldb.Batch)
	rh := uint32(removeHeight)
	err = rollbackTrades(snapshot, batch, rh)
	if err != nil {
		return wrapError(err)
	}
	err = rollbackAccounts(snapshot, batch, rh)
	if err != nil {
		return wrapError(err)
	}
	err = rollbackAssets(snapshot, batch, rh)
	if err != nil {
		return wrapError(err)
	}
	err = rollbackAliases(snapshot, batch, rh)
	if err != nil {
		return wrapError(err)
	}
	err = rollbackBlocks(snapshot, batch, rh)
	if err != nil {
		return wrapError(err)
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (s *Storage) Height() (int, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	return height(snapshot)
}

func (s *Storage) AssetInfo(asset crypto.Digest) (*data.AssetInfo, error) {
	if bytes.Equal(asset[:], data.WavesID[:]) {
		return &data.WavesAssetInfo, nil
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	a, ok, err := bs.assetInfo(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read AssetInfo")
	}
	if !ok {
		return nil, errors.Errorf("failed to locate asset '%s'", asset.String())
	}
	return &data.AssetInfo{
		ID:         asset,
		Name:       a.name,
		Issuer:     a.issuer,
		Decimals:   a.decimals,
		Reissuable: a.reissuable,
		Supply:     a.supply,
	}, nil
}

func (s *Storage) Trades(amountAsset, priceAsset crypto.Digest, limit int) ([]data.Trade, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return trades(snapshot, amountAsset, priceAsset, 0, math.MaxInt32, limit)
}

func (s *Storage) TradesRange(amountAsset, priceAsset crypto.Digest, from, to uint64) ([]data.Trade, error) {
	f := data.TimeFrameFromTimestampMS(from)
	t := data.TimeFrameFromTimestampMS(to)
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return trades(snapshot, amountAsset, priceAsset, f, t, maxLimit)
}

func (s *Storage) TradesByAddress(amountAsset, priceAsset crypto.Digest, address proto.Address, limit int) ([]data.Trade, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return addressTrades(snapshot, amountAsset, priceAsset, address, limit)
}

func (s *Storage) CandlesRange(amountAsset, priceAsset crypto.Digest, from, to uint32, timeFrameScale int) ([]data.Candle, error) {
	limit := timeFrameScale * maxLimit
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return candles(snapshot, amountAsset, priceAsset, from, to+uint32(timeFrameScale), limit)
}

func (s *Storage) DayCandle(amountAsset, priceAsset crypto.Digest) (data.Candle, error) {
	sts := uint64(time.Now().Unix() * 1000)
	ttf := data.TimeFrameFromTimestampMS(sts)
	ftf := data.ScaleTimeFrame(ttf, 288)
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return data.Candle{}, err
	}
	defer snapshot.Release()
	cs, err := candles(snapshot, amountAsset, priceAsset, ftf, ttf, math.MaxInt32)
	if err != nil {
		return data.Candle{}, err
	}
	r := data.NewCandleFromTimestamp(data.TimestampMSFromTimeFrame(ftf))
	for _, c := range cs {
		r.Combine(c)
	}
	return r, nil
}

func (s *Storage) HasBlock(height int, block crypto.Signature) (bool, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return false, err
	}
	defer snapshot.Release()
	return hasBlock(snapshot, uint32(height), block)
}

func (s *Storage) Markets() (map[data.MarketID]data.Market, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect markets")
	}
	defer snapshot.Release()
	return marketsMap(snapshot)
}

func (s *Storage) BlockID(height int) (crypto.Signature, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return crypto.Signature{}, err
	}
	defer snapshot.Release()
	b, err := block(snapshot, uint32(height))
	return b, nil
}

func (s *Storage) IssuerBalance(issuer proto.Address, asset crypto.Digest) (uint64, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	b, _, err := bs.balance(issuer, asset)
	if err != nil {
		return 0, err
	}
	return b, nil
}

//
//var (
//	defaultReadOptions  = &opt.ReadOptions{}
//	defaultWriteOptions = &opt.WriteOptions{}
//	lastHeightKey       = []byte{LastHeightKeyPrefix}
//)
//
//type pairPublicKeyTimestampKey struct {
//	prefix      byte
//	amountAsset crypto.Digest
//	priceAsset  crypto.Digest
//	pk          crypto.PublicKey
//	ts          uint64
//}
//
//func (k pairPublicKeyTimestampKey) bytes() []byte {
//	buf := make([]byte, 1+2*crypto.DigestSize+crypto.PublicKeySize+8)
//	buf[0] = k.prefix
//	copy(buf[1:], k.amountAsset[:])
//	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
//	copy(buf[1+2*crypto.DigestSize:], k.pk[:])
//	binary.BigEndian.PutUint64(buf[1+2*crypto.DigestSize+crypto.PublicKeySize:], k.ts)
//	return buf
//}
//
//type timeFramePairKey struct {
//	prefix      byte
//	tf          uint32
//	amountAsset crypto.Digest
//	priceAsset  crypto.Digest
//}
//
//func (k timeFramePairKey) bytes() []byte {
//	buf := make([]byte, 1+4+2*crypto.DigestSize)
//	buf[0] = k.prefix
//	binary.BigEndian.PutUint32(buf[1:], k.tf)
//	copy(buf[1+4:], k.amountAsset[:])
//	copy(buf[1+4+crypto.DigestSize:], k.priceAsset[:])
//	return buf
//}
//
//type digests []crypto.Digest
//
//func (l digests) marshalBinary() []byte {
//	n := len(l)
//	buf := make([]byte, 2+n*crypto.DigestSize)
//	binary.BigEndian.PutUint16(buf, uint16(n))
//	for i, id := range l {
//		copy(buf[2+i*crypto.DigestSize:], id[:])
//	}
//	return buf
//}
//
//func (l *digests) unmarshalBinary(data []byte) error {
//	if l := len(data); l < 2 {
//		return errors.Errorf("%d is not enough bites for digests, expected %d", l, 2)
//	}
//	n := int(binary.BigEndian.Uint16(data))
//	data = data[2:]
//	var id crypto.Digest
//	ids := make([]crypto.Digest, n)
//	for i := 0; i < n; i++ {
//		copy(id[:], data[:crypto.DigestSize])
//		data = data[crypto.DigestSize:]
//		ids[i] = id
//	}
//	*l = digests(ids)
//	return nil
//}
//
//func (s *Storage) PutBlockState(height int, aliasBinds []data.AliasBind) error {
//	wrapError := func(err error) error {
//		return errors.Wrapf(err, "failed to store block's state at height %d", height)
//	}
//
//	snapshot, err := s.db.GetSnapshot()
//	if err != nil {
//		return wrapError(err)
//	}
//	bs := newBlockState(snapshot)
//	batch := new(leveldb.Batch)
//	putAliases(bs, batch, uint32(height), aliasBinds)
//
//	assets, err := s.loadAssetStates(stateUpdates)
//	if err != nil {
//		return wrapError(err)
//	}
//
//	candles := map[timeFramePairKey]Candle{}
//	markets := map[pairKey]Market{}
//	affectedTimeFrames := make([]uint32, 0)
//	for _, t := range trades {
//		tf := TimeFrameFromTimestampMS(t.Timestamp)
//		if !contains(affectedTimeFrames, tf) {
//			affectedTimeFrames = append(affectedTimeFrames, tf)
//		}
//		ck := timeFramePairKey{CandlesKeyPrefix, tf, t.AmountAsset, t.PriceAsset}
//		var c Candle
//		c, ok := candles[ck]
//		if !ok {
//			cb, err := s.db.Get(ck.bytes(), defaultReadOptions)
//			if err != nil {
//				if err != leveldb.ErrNotFound {
//					return wrapError(err)
//				}
//				c = NewCandleFromTimestamp(t.Timestamp)
//			} else {
//				err = c.UnmarshalBinary(cb)
//				if err != nil {
//					return wrapError(err)
//				}
//			}
//		}
//		c.UpdateFromTrade(t)
//		candles[ck] = c
//		mk := pairKey{marketKeyPrefix, t.AmountAsset, t.PriceAsset}
//		m, ok := markets[mk]
//		if !ok {
//			mb, err := s.db.Get(mk.bytes(), defaultReadOptions)
//			if err != nil {
//				if err != leveldb.ErrNotFound {
//					return wrapError(err)
//				}
//				m = Market{}
//			} else {
//				err = m.UnmarshalBinary(mb)
//				if err != nil {
//					return wrapError(err)
//				}
//			}
//		}
//		m.UpdateFromTrade(t)
//		markets[mk] = m
//	}
//
//	updateAssets(assets, batch, height, stateUpdates)
//	n := len(trades)
//	updateLastHeight(batch, height)
//	mtf := min(affectedTimeFrames)
//	putBlockInfo(batch, height, block, n == 0, mtf)
//	putTradesIDs(batch, block, trades)
//	err = s.putTrades(batch, trades)
//	if err != nil {
//		return wrapError(err)
//	}
//	err = s.updateAffectedTimeFrames(batch, affectedTimeFrames, height)
//	if err != nil {
//		return wrapError(err)
//	}
//	for _, t := range trades {
//		k1 := digestKey(tradeKeyPrefix, t.TransactionID)
//		v1 := make([]byte, TradeSize)
//		v1, err := t.MarshalBinary()
//		if err != nil {
//			return wrapError(err)
//		}
//		batch.Put(k1, v1)
//	}
//	for ck, cv := range candles {
//		v, err := cv.MarshalBinary()
//		if err != nil {
//			return wrapError(err)
//		}
//		ckb := ck.bytes()
//		batch.Put(ckb, v)
//		ck2 := candleKey{candleKeyPrefix, ck.amountAsset, ck.priceAsset, ck.tf}
//		batch.Put(ck2.bytes(), ckb)
//	}
//	for mk, mv := range markets {
//		vb, err := mv.MarshalBinary()
//		if err != nil {
//			return wrapError(err)
//		}
//		batch.Put(mk.bytes(), vb)
//	}
//	err = s.db.Write(batch, defaultWriteOptions)
//	if err != nil {
//		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
//	}
//return nil
//}

//func (s *Storage) loadAssetStates(updates []Change) (map[crypto.Digest]AssetState, error) {
//	r := make(map[crypto.Digest]AssetState)
//	for _, u := range updates {
//		_, ok := r[u.Info.ID]
//		if !ok {
//			var state AssetState
//			k := DigestKey{AssetStateKeyPrefix, u.Info.ID}
//			v, err := s.db.Get(k.bytes(), defaultReadOptions)
//			if err != nil {
//				if err != leveldb.ErrNotFound {
//					return nil, errors.Wrap(err, "failed to load AssetState")
//				}
//				state = AssetState{}
//			} else {
//				err = state.unmarshalBinary(v)
//				if err != nil {
//					return nil, errors.Wrap(err, "failed to load AssetState")
//				}
//			}
//			r[u.Info.ID] = state
//		}
//	}
//	return r, nil
//}
//
//func updateAssets(states map[crypto.Digest]AssetState, batch *leveldb.Batch, height int, updates []Change) {
//	diffs := make(map[crypto.Digest]AssetChange)
//	for _, u := range updates {
//		if u.Diff.Created {
//			b := DigestKey{assetKeyPrefix, u.Info.ID}
//			batch.Put(b.bytes(), u.Info.marshalBinary())
//		}
//		state := states[u.Info.ID]
//		states[u.Info.ID] = *state.Update(u.Diff)
//		diff := diffs[u.Info.ID]
//		diffs[u.Info.ID] = *diff.Add(u.Diff)
//	}
//	for k, v := range states {
//		kk :=DigestKey{AssetStateKeyPrefix, k}
//		batch.Put(kk.bytes(), v.marshalBinary())
//	}
//	for k, v := range diffs {
//		batch.Put(uint32AndDigestKey(AssetDiffsKeyPrefix, uint32(height), k), v.MarshalBinary())
//	}
//}
//

//func putAddress(batch *leveldb.Batch, pk crypto.PublicKey, scheme byte) error {
//	a, err := proto.NewAddressFromPublicKey(scheme, pk)
//	if err != nil {
//		return err
//	}
//	batch.Put(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), a[:])
//	batch.Put(addressKey(AddressToPublicKeyKeyPrefix, a), pk[:])
//	return nil
//}
//
//func (s *Storage) address(pk crypto.PublicKey) (proto.Address, error) {
//	b, err := s.db.Get(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), defaultReadOptions)
//	if err != nil {
//		return proto.Address{}, err
//	}
//	a, err := proto.NewAddressFromBytes(b)
//	if err != nil {
//		return proto.Address{}, err
//	}
//	return a, nil
//}
//
//func (s *Storage) checkAddress(pk crypto.PublicKey) bool {
//	ok, _ := s.db.Has(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), defaultReadOptions)
//	return ok
//}
//
//func (s *Storage) putTrades(batch *leveldb.Batch, trades []Trade) error {
//	d1 := map[pairTimestampKey]digests{}
//	d2 := map[pairPublicKeyTimestampKey]digests{}
//	for _, t := range trades {
//		if !s.checkAddress(t.Buyer) {
//			err := putAddress(batch, t.Buyer, s.Scheme)
//			if err != nil {
//				return err
//			}
//		}
//		if !s.checkAddress(t.Seller) {
//			err := putAddress(batch, t.Seller, s.Scheme)
//			if err != nil {
//				return err
//			}
//		}
//		k1 := pairTimestampKey{marketTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Timestamp}
//		v, ok := d1[k1]
//		if ok {
//			v = append(v, t.TransactionID)
//		} else {
//			v = digests{t.TransactionID}
//		}
//		d1[k1] = v
//		k2 := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Seller, t.Timestamp}
//		v, ok = d2[k2]
//		if ok {
//			v = append(v, t.TransactionID)
//		} else {
//			v = digests{t.TransactionID}
//		}
//		d2[k2] = v
//		k3 := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Buyer, t.Timestamp}
//		v, ok = d2[k3]
//		if ok {
//			v = append(v, t.TransactionID)
//		} else {
//			v = digests{t.TransactionID}
//		}
//		d2[k3] = v
//	}
//	for k, v := range d1 {
//		batch.Put(k.bytes(), v.marshalBinary())
//	}
//	for k, v := range d2 {
//		batch.Put(k.bytes(), v.marshalBinary())
//	}
//	return nil
//}
//

//func (s *Storage) findFirstNonEmptyBlock(height int) (bool, blockInfo, error) {
//	bi, err := s.blockInfo(height)
//	if err != nil {
//		if err != leveldb.ErrNotFound {
//			return false, blockInfo{}, err
//		}
//		return false, blockInfo{}, nil
//	}
//	if !bi.Empty {
//		return true, bi, nil
//	}
//	return s.findFirstNonEmptyBlock(height - 1)
//}
//
//func (s *Storage) getTrades(block crypto.Signature) ([]Trade, error) {
//	k := signatureKey(BlockTradesKeyPrefix, block)
//	v, err := s.db.Get(k, defaultReadOptions)
//	if err != nil && err != leveldb.ErrNotFound {
//		return []Trade{}, errors.Wrap(err, "failed to collect Trades")
//	}
//	n := len(v) / crypto.DigestSize
//	r := make([]Trade, n)
//	tid := make([]byte, crypto.DigestSize)
//	for i := 0; i < n; i++ {
//		copy(tid, v[i*crypto.DigestSize:])
//		tk := prepend(tradeKeyPrefix, tid)
//		tv, err := s.db.Get(tk, defaultReadOptions)
//		if err != nil {
//			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
//		}
//		err = r[i].UnmarshalBinary(tv)
//		if err != nil {
//			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
//		}
//	}
//	return r, nil
//}
//
//func (s *Storage) removeCandlesAfter(batch *leveldb.Batch, tf uint32) error {
//	start := timeFramePairKey{CandlesKeyPrefix, tf, crypto.Digest{}, crypto.Digest{}}
//	end := []byte{CandlesKeyPrefix + 1}
//	it := s.db.NewIterator(&util.Range{Start: start.bytes(), Limit: end}, defaultReadOptions)
//	for it.Next() {
//		batch.Delete(it.Key())
//	}
//	it.Release()
//	return it.Error()
//}
//
//func (s *Storage) removeEarliestTimeFrames(batch *leveldb.Batch, etf uint32) error {
//	start := uint32Key{EarliestHeightKeyPrefix, etf}
//	end := uint32Key{EarliestHeightKeyPrefix, math.MaxUint32}
//	it := s.db.NewIterator(&util.Range{Start: start.bytes(), Limit: end.bytes()}, defaultReadOptions)
//	for it.Next() {
//		batch.Delete(it.Key())
//	}
//	it.Release()
//	return it.Error()
//}
//
//func (s *Storage) updateAffectedTimeFrames(batch *leveldb.Batch, timeFrames []uint32, height int) error {
//	for _, tf := range timeFrames {
//		k := uint32Key{EarliestHeightKeyPrefix, tf}
//		b, err := s.db.Get(k.bytes(), defaultReadOptions)
//		h := math.MaxInt32
//		if err != nil && err != leveldb.ErrNotFound {
//			return err
//		}
//		if len(b) > 0 {
//			h = int(binary.BigEndian.Uint32(b))
//		}
//		if height < h {
//			v := make([]byte, 4)
//			binary.BigEndian.PutUint32(v, uint32(height))
//			batch.Put(k.bytes(), v)
//		}
//	}
//	return nil
//}
//
//func putBlockInfo(batch *leveldb.Batch, height int, block crypto.Signature, empty bool, earliestTimeFrame uint32) {
//	k := uint32Key{blockInfoKeyPrefix, uint32(height)}
//	bi := blockInfo{empty, block, earliestTimeFrame}
//	v := bi.marshalBinary()
//	batch.Put(k.bytes(), v)
//}
//
//func putTradesIDs(batch *leveldb.Batch, block crypto.Signature, trades []Trade) {
//	n := len(trades)
//	k := signatureKey(BlockTradesKeyPrefix, block)
//	v := make([]byte, crypto.DigestSize*n)
//	for i, t := range trades {
//		copy(v[i*crypto.DigestSize:], t.TransactionID[:])
//	}
//	batch.Put(k, v)
//}
//
//func youngestTimeFrame(trades []Trade) uint32 {
//	ts := uint64(math.MaxUint64)
//	for _, t := range trades {
//		if t.Timestamp < ts {
//			ts = t.Timestamp
//		}
//	}
//	return TimeFrameFromTimestampMS(ts)
//}
//
//func prepend(prefix byte, key []byte) []byte {
//	k := make([]byte, 1+len(key))
//	k[0] = prefix
//	copy(k[1:], key)
//	return k
//}
//
//func removeTrades(batch *leveldb.Batch, trades []Trade) {
//	for _, t := range trades {
//		k := DigestKey{tradeKeyPrefix, t.TransactionID}
//		batch.Delete(k.bytes())
//	}
//}
//
//func updateLastHeight(batch *leveldb.Batch, height int) {
//	v := make([]byte, 4)
//	binary.BigEndian.PutUint32(v, uint32(height))
//	batch.Put(lastHeightKey, v)
//}
//
//func contains(a []uint32, v uint32) bool {
//	for _, x := range a {
//		if v == x {
//			return true
//		}
//	}
//	return false
//}
//
//func min(a []uint32) uint32 {
//	r := uint32(math.MaxUint32)
//	for _, x := range a {
//		if x < r {
//			r = x
//		}
//	}
//	return r
//}
