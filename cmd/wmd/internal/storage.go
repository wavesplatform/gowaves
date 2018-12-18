package internal

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	ldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	ldbUtil "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
	"time"
)

const (
	LastHeightKeyPrefix byte = iota
	BlockAtHeightKeyPrefix
	TradesKeyPrefix
	CandlesKeyPrefix
	CandlesSecondKeyPrefix
	BlockTradesKeyPrefix
	EarliestHeightKeyPrefix
	PairTradesKeyPrefix
	PairPublicKeyTradesKeyPrefix
	AssetInfoKeyPrefix
	AssetStateKeyPrefix
	AssetDiffsKeyPrefix
	PublicKeyToAddressKeyPrefix
	AddressToPublicKeyKeyPrefix
	MarketsKeyPrefix
)

const (
	maxLimit = 1000
)

var (
	defaultReadOptions  = &opt.ReadOptions{}
	defaultWriteOptions = &opt.WriteOptions{}
	lastHeightKey       = []byte{LastHeightKeyPrefix}
)

type pairKey struct {
	prefix      byte
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
}

func (k pairKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize)
	buf[0] = k.prefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	return buf
}

type pairTimestampKey struct {
	prefix      byte
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	ts          uint64
}

func (k pairTimestampKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+8)
	buf[0] = k.prefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	binary.BigEndian.PutUint64(buf[1+2*crypto.DigestSize:], k.ts)
	return buf
}

type pairTimeFrameKey struct {
	prefix      byte
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	timeFrame   uint32
}

func (k pairTimeFrameKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+4)
	buf[0] = k.prefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	binary.BigEndian.PutUint32(buf[1+2*crypto.DigestSize:], k.timeFrame)
	return buf
}

type pairPublicKeyTimestampKey struct {
	prefix      byte
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	pk          crypto.PublicKey
	ts          uint64
}

func (k pairPublicKeyTimestampKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+crypto.PublicKeySize+8)
	buf[0] = k.prefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	copy(buf[1+2*crypto.DigestSize:], k.pk[:])
	binary.BigEndian.PutUint64(buf[1+2*crypto.DigestSize+crypto.PublicKeySize:], k.ts)
	return buf
}

//type timestampPairKey struct {
//	prefix      byte
//	ts          uint64
//	amountAsset crypto.Digest
//	priceAsset  crypto.Digest
//}
//TODO:remove
//
//func (k timestampPairKey) bytes() []byte {
//	buf := make([]byte, 1+8+2*crypto.DigestSize)
//	buf[0] = k.prefix
//	binary.BigEndian.PutUint64(buf[1:], k.ts)
//	copy(buf[1+8:], k.amountAsset[:])
//	copy(buf[1+8+crypto.DigestSize:], k.priceAsset[:])
//	return buf
//}

type timeFramePairKey struct {
	prefix      byte
	tf          uint32
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
}

func (k timeFramePairKey) bytes() []byte {
	buf := make([]byte, 1+4+2*crypto.DigestSize)
	buf[0] = k.prefix
	binary.BigEndian.PutUint32(buf[1:], k.tf)
	copy(buf[1+4:], k.amountAsset[:])
	copy(buf[1+4+crypto.DigestSize:], k.priceAsset[:])
	return buf
}

type digests []crypto.Digest

func (l digests) marshalBinary() []byte {
	n := len(l)
	buf := make([]byte, 2+n*crypto.DigestSize)
	binary.BigEndian.PutUint16(buf, uint16(n))
	for i, id := range l {
		copy(buf[2+i*crypto.DigestSize:], id[:])
	}
	return buf
}

func (l *digests) unmarshalBinary(data []byte) error {
	if l := len(data); l < 2 {
		return errors.Errorf("%d is not enough bites for digests, expected %d", l, 2)
	}
	n := int(binary.BigEndian.Uint16(data))
	data = data[2:]
	var id crypto.Digest
	ids := make([]crypto.Digest, n)
	for i := 0; i < n; i++ {
		copy(id[:], data[:crypto.DigestSize])
		data = data[crypto.DigestSize:]
		ids[i] = id
	}
	*l = digests(ids)
	return nil
}

type blockInfo struct {
	Empty             bool
	ID                crypto.Signature
	EarliestTimeFrame uint32
}

func (b *blockInfo) marshalBinary() []byte {
	buf := make([]byte, crypto.SignatureSize+1+4)
	if b.Empty {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	copy(buf[1:], b.ID[:])
	binary.BigEndian.PutUint32(buf[1+crypto.SignatureSize:], b.EarliestTimeFrame)
	return buf
}

func (b *blockInfo) unmarshalBinary(data []byte) error {
	if len(data) != crypto.SignatureSize+1+4 {
		return errors.New("incorrect data for blockInfo")
	}
	b.Empty = data[0] == 1
	copy(b.ID[:], data[1:])
	b.EarliestTimeFrame = binary.BigEndian.Uint32(data[1+crypto.SignatureSize:])
	return nil
}

type AssetState struct {
	Reissuable    bool
	Supply        uint64
	IssuerBalance uint64
}

func (a *AssetState) Update(diff AssetDiff) *AssetState {
	if diff.Disabled {
		a.Reissuable = false
	}
	a.Supply += diff.Issued
	a.Supply -= diff.Burned
	return a
}

func (a *AssetState) marshalBinary() []byte {
	buf := make([]byte, 1+8+8)
	if a.Reissuable {
		buf[0] = 1
	}
	binary.BigEndian.PutUint64(buf[1:], a.Supply)
	binary.BigEndian.PutUint64(buf[1+8:], a.IssuerBalance)
	return buf
}

func (a *AssetState) unmarshalBinary(data []byte) error {
	if l := len(data); l < 1+8+8 {
		return errors.Errorf("%d is not enough bytes for AssetState, expected %d", l, 1+8+8)
	}
	a.Reissuable = data[0] == 1
	a.Supply = binary.BigEndian.Uint64(data[1:])
	a.IssuerBalance = binary.BigEndian.Uint64(data[1+8:])
	return nil
}

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

func (s *Storage) GetLastHeight() (int, error) {
	v, err := s.db.Get(lastHeightKey, defaultReadOptions)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return 0, errors.Wrap(err, "failed to get Height")
		}
		return 0, nil
	}
	if l := len(v); l != 4 {
		return 0, errors.Errorf("%d is incorrect length for Height value, expected 4", l)
	}
	return int(binary.BigEndian.Uint32(v)), nil
}

func (s *Storage) readAssetInfo(asset crypto.Digest) (*AssetInfo, error) {
	if bytes.Equal(asset[:], wavesID[:]) {
		return &wavesAssetInfo, nil
	}
	b, err := s.db.Get(digestKey(AssetInfoKeyPrefix, asset), defaultReadOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read AssetInfo")
	}
	var i AssetInfo
	err = i.unmarshalBinary(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read AssetInfo")
	}
	return &i, nil
}

func (s *Storage) readTrade(id crypto.Digest) (*Trade, error) {
	b, err := s.db.Get(digestKey(TradesKeyPrefix, id), defaultReadOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Trade")
	}
	var t Trade
	err = t.UnmarshalBinary(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Trade")
	}
	return &t, nil
}

func (s *Storage) Trades(amountAsset, priceAsset crypto.Digest, limit int) ([]Trade, error) {
	sk := pairTimeFrameKey{PairTradesKeyPrefix, amountAsset, priceAsset, 0}
	ek := pairTimeFrameKey{PairTradesKeyPrefix, amountAsset, priceAsset, math.MaxUint32}
	it := s.db.NewIterator(&ldbUtil.Range{Start: sk.bytes(), Limit: ek.bytes()}, defaultReadOptions)
	c := 0
	var trades []Trade
	if it.Last() {
		for {
			if c >= limit {
				break
			}
			b := it.Value()
			var ids digests
			err := ids.unmarshalBinary(b)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load trades")
			}
			for _, id := range ids {
				t, err := s.readTrade(id)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load trades")
				}
				if c == limit {
					break
				}
				trades = append(trades, *t)
				c++
			}
			if !it.Prev() {
				break
			}
		}
	}
	return trades, nil
}

func (s *Storage) TradesRange(amountAsset, priceAsset crypto.Digest, from, to uint64) ([]Trade, error) {
	sk := pairTimestampKey{PairTradesKeyPrefix, amountAsset, priceAsset, from}
	ek := pairTimestampKey{PairTradesKeyPrefix, amountAsset, priceAsset, to + 1}
	it := s.db.NewIterator(&ldbUtil.Range{Start: sk.bytes(), Limit: ek.bytes()}, defaultReadOptions)
	c := 0
	var trades []Trade
	if it.Last() {
		for {
			if c >= maxLimit {
				break
			}
			b := it.Value()
			var ids digests
			err := ids.unmarshalBinary(b)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load trades")
			}
			for _, id := range ids {
				t, err := s.readTrade(id)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load trades")
				}
				if c == maxLimit {
					break
				}
				trades = append(trades, *t)
				c++
			}
			if !it.Prev() {
				break
			}
		}
	}
	return trades, nil
}

func (s *Storage) CandlesRange(amountAsset, priceAsset crypto.Digest, from, to uint32, timeFrameScale int) ([]Candle, error) {
	limit := timeFrameScale * maxLimit
	cs, err := s.candles(amountAsset, priceAsset, from, to+uint32(timeFrameScale), limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load candles")
	}
	return cs, nil
}

func (s *Storage) TradesByPublicKey(amountAsset, priceAsset crypto.Digest, pk crypto.PublicKey, limit int) ([]Trade, error) {
	sk := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, amountAsset, priceAsset, pk, 0}
	ek := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, amountAsset, priceAsset, pk, math.MaxUint64}
	it := s.db.NewIterator(&ldbUtil.Range{Start: sk.bytes(), Limit: ek.bytes()}, defaultReadOptions)
	c := 0
	var trades []Trade
	if it.Last() {
		for {
			if c >= maxLimit {
				break
			}
			b := it.Value()
			var ids digests
			err := ids.unmarshalBinary(b)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load trades")
			}
			for _, id := range ids {
				t, err := s.readTrade(id)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load trades")
				}
				if c == maxLimit {
					break
				}
				trades = append(trades, *t)
				c++
			}
			if !it.Prev() {
				break
			}
		}
	}
	return trades, nil
}

func (s *Storage) DayCandle(amountAsset, priceAsset crypto.Digest) (Candle, error) {
	sts := uint64(time.Now().Unix() * 1000)
	ttf := timeFrameFromTimestampMS(sts)
	ftf := scaleTimeFrame(ttf, 288)
	cs, err := s.candles(amountAsset, priceAsset, ftf, ttf, math.MaxInt32)
	if err != nil {
		return Candle{}, errors.Wrap(err, "failed to build DayCandle")
	}
	r := NewCandle(timestampMSFromTimeFrame(ftf))
	for _, c := range cs {
		r.Combine(c)
	}
	return r, nil
}

func (s *Storage) candles(amountAsset, priceAsset crypto.Digest, start, stop uint32, limit int) ([]Candle, error) {
	sk := pairTimeFrameKey{CandlesSecondKeyPrefix, amountAsset, priceAsset, start}
	ek := pairTimeFrameKey{CandlesSecondKeyPrefix, amountAsset, priceAsset, stop}
	r := make([]Candle, 0)
	it := s.db.NewIterator(&ldbUtil.Range{Start: sk.bytes(), Limit: ek.bytes()}, defaultReadOptions)
	var c Candle
	i := 0
	for it.Next() {
		k := it.Value()
		b, err := s.db.Get(k, defaultReadOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect candles")
		}
		err = c.UnmarshalBinary(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect candles")
		}
		r = append(r, c)
		i++
		if i == limit {
			break
		}
	}
	return r, nil
}

func (s *Storage) PublicKey(address proto.Address) (crypto.PublicKey, error) {
	b, err := s.db.Get(addressKey(AddressToPublicKeyKeyPrefix, address), defaultReadOptions)
	if err != nil {
		return crypto.PublicKey{}, errors.Wrap(err, "failed to find PublicKey for Address")
	}
	pk, err := crypto.NewPublicKeyFromBytes(b)
	if err != nil {
		return crypto.PublicKey{}, errors.Wrap(err, "failed to find PublicKey for Address")
	}
	return pk, nil
}

func (s *Storage) ShouldImportBlock(height int, block crypto.Signature) (bool, error) {
	v, err := s.db.Get(uint32Key(BlockAtHeightKeyPrefix, uint32(height)), defaultReadOptions)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return false, errors.Wrapf(err, "failed to check existence of block '%s'", block.String())
		}
		return true, nil
	}
	var bi blockInfo
	err = bi.unmarshalBinary(v)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check existence of block '%s'", block.String())
	}
	if bytes.Equal(bi.ID[:], block[:]) {
		return false, nil
	}
	rollbackHeight, err := s.FindCorrectRollbackHeight(height - 1)
	if err != nil {
		return false, errors.Wrapf(err, "failed to find correct Rollback height while applying block '%s'", block.String())
	}
	err = s.Rollback(rollbackHeight)
	if err != nil {
		return false, errors.Wrapf(err, "failed to Rollback to height %d", height-1)
	}
	return true, nil
}

func (s *Storage) Markets() (map[MarketID]MarketData, error) {
	sk := pairKey{MarketsKeyPrefix, wavesID, wavesID}
	ek := pairKey{MarketsKeyPrefix, lastID, lastID}
	it := s.db.NewIterator(&ldbUtil.Range{Start: sk.bytes(), Limit: ek.bytes()}, defaultReadOptions)
	r := make(map[MarketID]MarketData, 0)
	for it.Next() {
		k := it.Key()
		var m MarketID
		err := m.UnmarshalBinary(k[1:])
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect markets")
		}
		var md MarketData
		err = md.UnmarshalBinary(it.Value())
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect markets")
		}
		r[m] = md
	}
	return r, nil
}

func (s *Storage) loadAssetStates(updates []AssetUpdate) (map[crypto.Digest]AssetState, error) {
	r := make(map[crypto.Digest]AssetState)
	for _, u := range updates {
		_, ok := r[u.Info.ID]
		if !ok {
			var state AssetState
			k := digestKey(AssetStateKeyPrefix, u.Info.ID)
			v, err := s.db.Get(k, defaultReadOptions)
			if err != nil {
				if err != ldbErrors.ErrNotFound {
					return nil, errors.Wrap(err, "failed to load AssetState")
				}
				state = AssetState{}
			} else {
				err = state.unmarshalBinary(v)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load AssetState")
				}
			}
			r[u.Info.ID] = state
		}
	}
	return r, nil
}

func updateAssets(states map[crypto.Digest]AssetState, batch *leveldb.Batch, height int, updates []AssetUpdate) {
	diffs := make(map[crypto.Digest]AssetDiff)
	for _, u := range updates {
		if u.Diff.Created {
			b := digestKey(AssetInfoKeyPrefix, u.Info.ID)
			batch.Put(b, u.Info.marshalBinary())
		}
		state := states[u.Info.ID]
		states[u.Info.ID] = *state.Update(u.Diff)
		diff := diffs[u.Info.ID]
		diffs[u.Info.ID] = *diff.Add(u.Diff)
	}
	for k, v := range states {
		batch.Put(digestKey(AssetStateKeyPrefix, k), v.marshalBinary())
	}
	for k, v := range diffs {
		batch.Put(uint32AndDigestKey(AssetDiffsKeyPrefix, uint32(height), k), v.marshalBinary())
	}
}

func (s *Storage) PutBlock(height int, block crypto.Signature, trades []Trade, assetUpdates []AssetUpdate) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}

	assets, err := s.loadAssetStates(assetUpdates)
	if err != nil {
		return wrapError(err)
	}

	candles := map[timeFramePairKey]Candle{}
	markets := map[pairKey]MarketData{}
	affectedTimeFrames := make([]uint32, 0)
	for _, t := range trades {
		tf := timeFrameFromTimestampMS(t.Timestamp)
		if !contains(affectedTimeFrames, tf) {
			affectedTimeFrames = append(affectedTimeFrames, tf)
		}
		ck := timeFramePairKey{CandlesKeyPrefix, tf, t.AmountAsset, t.PriceAsset}
		var c Candle
		c, ok := candles[ck]
		if !ok {
			cb, err := s.db.Get(ck.bytes(), defaultReadOptions)
			if err != nil {
				if err != ldbErrors.ErrNotFound {
					return wrapError(err)
				}
				c = NewCandle(t.Timestamp)
			} else {
				err = c.UnmarshalBinary(cb)
				if err != nil {
					return wrapError(err)
				}
			}
		}
		c.UpdateFromTrade(t)
		candles[ck] = c
		mk := pairKey{MarketsKeyPrefix, t.AmountAsset, t.PriceAsset}
		m, ok := markets[mk]
		if !ok {
			mb, err := s.db.Get(mk.bytes(), defaultReadOptions)
			if err != nil {
				if err != ldbErrors.ErrNotFound {
					return wrapError(err)
				}
				m = MarketData{}
			} else {
				err = m.UnmarshalBinary(mb)
				if err != nil {
					return wrapError(err)
				}
			}
		}
		m.UpdateFromTrade(t)
		markets[mk] = m
	}

	batch := new(leveldb.Batch)
	updateAssets(assets, batch, height, assetUpdates)
	n := len(trades)
	updateLastHeight(batch, height)
	mtf := min(affectedTimeFrames)
	putBlockInfo(batch, height, block, n == 0, mtf)
	putTradesIDs(batch, block, trades)
	err = s.putTrades(batch, trades)
	if err != nil {
		return wrapError(err)
	}
	err = s.updateAffectedTimeFrames(batch, affectedTimeFrames, height)
	if err != nil {
		return wrapError(err)
	}
	for _, t := range trades {
		k1 := digestKey(TradesKeyPrefix, t.TransactionID)
		v1 := make([]byte, TradeSize)
		v1, err := t.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(k1, v1)
	}
	for ck, cv := range candles {
		v, err := cv.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		ckb := ck.bytes()
		batch.Put(ckb, v)
		ck2 := pairTimeFrameKey{CandlesSecondKeyPrefix, ck.amountAsset, ck.priceAsset, ck.tf}
		batch.Put(ck2.bytes(), ckb)
	}
	for mk, mv := range markets {
		vb, err := mv.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(mk.bytes(), vb)
	}
	err = s.db.Write(batch, defaultWriteOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}
	return nil
}

func putAddress(batch *leveldb.Batch, pk crypto.PublicKey, scheme byte) error {
	a, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return err
	}
	batch.Put(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), a[:])
	batch.Put(addressKey(AddressToPublicKeyKeyPrefix, a), pk[:])
	return nil
}

func (s *Storage) address(pk crypto.PublicKey) (proto.Address, error) {
	b, err := s.db.Get(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), defaultReadOptions)
	if err != nil {
		return proto.Address{}, err
	}
	a, err := proto.NewAddressFromBytes(b)
	if err != nil {
		return proto.Address{}, err
	}
	return a, nil
}

func (s *Storage) checkAddress(pk crypto.PublicKey) bool {
	ok, _ := s.db.Has(publicKeyKey(PublicKeyToAddressKeyPrefix, pk), defaultReadOptions)
	return ok
}

func (s *Storage) putTrades(batch *leveldb.Batch, trades []Trade) error {
	d1 := map[pairTimestampKey]digests{}
	d2 := map[pairPublicKeyTimestampKey]digests{}
	for _, t := range trades {
		if !s.checkAddress(t.Buyer) {
			err := putAddress(batch, t.Buyer, s.Scheme)
			if err != nil {
				return err
			}
		}
		if !s.checkAddress(t.Seller) {
			err := putAddress(batch, t.Seller, s.Scheme)
			if err != nil {
				return err
			}
		}
		k1 := pairTimestampKey{PairTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Timestamp}
		v, ok := d1[k1]
		if ok {
			v = append(v, t.TransactionID)
		} else {
			v = digests{t.TransactionID}
		}
		d1[k1] = v
		k2 := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Seller, t.Timestamp}
		v, ok = d2[k2]
		if ok {
			v = append(v, t.TransactionID)
		} else {
			v = digests{t.TransactionID}
		}
		d2[k2] = v
		k3 := pairPublicKeyTimestampKey{PairPublicKeyTradesKeyPrefix, t.AmountAsset, t.PriceAsset, t.Buyer, t.Timestamp}
		v, ok = d2[k3]
		if ok {
			v = append(v, t.TransactionID)
		} else {
			v = digests{t.TransactionID}
		}
		d2[k3] = v
	}
	for k, v := range d1 {
		batch.Put(k.bytes(), v.marshalBinary())
	}
	for k, v := range d2 {
		batch.Put(k.bytes(), v.marshalBinary())
	}
	return nil
}

func (s *Storage) FindCorrectRollbackHeight(height int) (int, error) {
	ok, bi, err := s.findFirstNonEmptyBlock(height)
	if err != nil {
		return 0, errors.Wrap(err, "failed to find first non empty block")
	}
	if !ok {
		return height, nil
	}
	etf := bi.EarliestTimeFrame
	k := uint32Key(EarliestHeightKeyPrefix, etf)
	b, err := s.db.Get(k, defaultReadOptions)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get earliest height for time frame %d", etf)
	}
	eh := int(binary.BigEndian.Uint32(b))
	if eh >= height {
		return height, nil
	} else {
		return s.FindCorrectRollbackHeight(eh)
	}
}

func (s *Storage) Rollback(height int) error {
	max, err := s.GetLastHeight()
	if err != nil {
		return errors.Wrap(err, "failed to Rollback")
	}
	if height >= max {
		return errors.Errorf("Rollback is impossible, current height %d is not lower then given %d", max, height)
	}
	for h := max; h > height; h-- {
		bi, err := s.blockInfo(h)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
		trades, err := s.getTrades(bi.ID)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
		batch := new(leveldb.Batch)
		err = s.removeCandlesAfter(batch, youngestTimeFrame(trades))
		if err != nil {
			batch.Reset()
			return errors.Wrapf(err, "rollback failure at height %d", h)
		}
		removeTrades(batch, trades)
		err = s.removeEarliestTimeFrames(batch, bi.EarliestTimeFrame)
		if err != nil {
			batch.Reset()
			return errors.Wrapf(err, "rollback failure at height %d", h)
		}
		batch.Delete(signatureKey(BlockTradesKeyPrefix, bi.ID))
		hk := uint32Key(BlockAtHeightKeyPrefix, uint32(h))
		batch.Delete(hk)
		updateLastHeight(batch, h-1)
		err = s.db.Write(batch, defaultWriteOptions)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
	}
	return nil
}

func (s *Storage) findFirstNonEmptyBlock(height int) (bool, blockInfo, error) {
	bi, err := s.blockInfo(height)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return false, blockInfo{}, err
		}
		return false, blockInfo{}, nil
	}
	if !bi.Empty {
		return true, bi, nil
	}
	return s.findFirstNonEmptyBlock(height - 1)
}

func (s *Storage) blockInfo(height int) (blockInfo, error) {
	k := uint32Key(BlockAtHeightKeyPrefix, uint32(height))
	v, err := s.db.Get(k, defaultReadOptions)
	if err != nil {
		return blockInfo{}, errors.Wrapf(err, "failed to get info about block at height %d", height)
	}
	var bi blockInfo
	err = bi.unmarshalBinary(v)
	if err != nil {
		return blockInfo{}, errors.Wrapf(err, "Rollback failure at height %d", height)
	}
	return bi, nil
}

func (s *Storage) getTrades(block crypto.Signature) ([]Trade, error) {
	k := signatureKey(BlockTradesKeyPrefix, block)
	v, err := s.db.Get(k, defaultReadOptions)
	if err != nil && err != ldbErrors.ErrNotFound {
		return []Trade{}, errors.Wrap(err, "failed to collect Trades")
	}
	n := len(v) / crypto.DigestSize
	r := make([]Trade, n)
	tid := make([]byte, crypto.DigestSize)
	for i := 0; i < n; i++ {
		copy(tid, v[i*crypto.DigestSize:])
		tk := prepend(TradesKeyPrefix, tid)
		tv, err := s.db.Get(tk, defaultReadOptions)
		if err != nil {
			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
		}
		err = r[i].UnmarshalBinary(tv)
		if err != nil {
			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
		}
	}
	return r, nil
}

func (s *Storage) removeCandlesAfter(batch *leveldb.Batch, tf uint32) error {
	start := timeFramePairKey{CandlesKeyPrefix, tf, crypto.Digest{}, crypto.Digest{}}
	end := []byte{CandlesKeyPrefix + 1}
	it := s.db.NewIterator(&ldbUtil.Range{Start: start.bytes(), Limit: end}, defaultReadOptions)
	for it.Next() {
		batch.Delete(it.Key())
	}
	it.Release()
	return it.Error()
}

func (s *Storage) removeEarliestTimeFrames(batch *leveldb.Batch, etf uint32) error {
	start := uint32Key(EarliestHeightKeyPrefix, etf)
	end := uint32Key(EarliestHeightKeyPrefix, math.MaxUint32)
	it := s.db.NewIterator(&ldbUtil.Range{Start: start, Limit: end}, defaultReadOptions)
	for it.Next() {
		batch.Delete(it.Key())
	}
	it.Release()
	return it.Error()
}

func (s *Storage) updateAffectedTimeFrames(batch *leveldb.Batch, timeFrames []uint32, height int) error {
	for _, tf := range timeFrames {
		k := uint32Key(EarliestHeightKeyPrefix, tf)
		b, err := s.db.Get(k, defaultReadOptions)
		h := math.MaxInt32
		if err != nil && err != ldbErrors.ErrNotFound {
			return err
		}
		if len(b) > 0 {
			h = int(binary.BigEndian.Uint32(b))
		}
		if height < h {
			v := make([]byte, 4)
			binary.BigEndian.PutUint32(v, uint32(height))
			batch.Put(k, v)
		}
	}
	return nil
}

func putBlockInfo(batch *leveldb.Batch, height int, block crypto.Signature, empty bool, earliestTimeFrame uint32) {
	k := uint32Key(BlockAtHeightKeyPrefix, uint32(height))
	bi := blockInfo{empty, block, earliestTimeFrame}
	v := bi.marshalBinary()
	batch.Put(k, v)
}

func putTradesIDs(batch *leveldb.Batch, block crypto.Signature, trades []Trade) {
	n := len(trades)
	k := signatureKey(BlockTradesKeyPrefix, block)
	v := make([]byte, crypto.DigestSize*n)
	for i, t := range trades {
		copy(v[i*crypto.DigestSize:], t.TransactionID[:])
	}
	batch.Put(k, v)
}

func youngestTimeFrame(trades []Trade) uint32 {
	ts := uint64(math.MaxUint64)
	for _, t := range trades {
		if t.Timestamp < ts {
			ts = t.Timestamp
		}
	}
	return timeFrameFromTimestampMS(ts)
}

func uint32Key(prefix byte, key uint32) []byte {
	k := make([]byte, 5)
	k[0] = prefix
	binary.BigEndian.PutUint32(k[1:], key)
	return k
}

//func uint64Key(prefix byte, key uint64) []byte {
//	k := make([]byte, 9)
//	k[0] = prefix
//	binary.BigEndian.PutUint64(k[1:], key)
//	return k
//}
// TODO: remove

func digestKey(prefix byte, key crypto.Digest) []byte {
	k := make([]byte, 1+crypto.DigestSize)
	k[0] = prefix
	copy(k[1:], key[:])
	return k
}

func publicKeyKey(prefix byte, pk crypto.PublicKey) []byte {
	k := make([]byte, 1+crypto.PublicKeySize)
	k[0] = prefix
	copy(k[1:], pk[:])
	return k
}

func addressKey(prefix byte, address proto.Address) []byte {
	k := make([]byte, 1+proto.AddressSize)
	k[0] = prefix
	copy(k[1:], address[:])
	return k
}

func signatureKey(prefix byte, sig crypto.Signature) []byte {
	k := make([]byte, 1+crypto.SignatureSize)
	k[0] = prefix
	copy(k[1:], sig[:])
	return k
}

func uint32AndDigestKey(prefix byte, a uint32, b crypto.Digest) []byte {
	k := make([]byte, 1+4+crypto.DigestSize)
	k[0] = prefix
	binary.BigEndian.PutUint32(k[1:], a)
	copy(k[1+4:], b[:])
	return k
}
func prepend(prefix byte, key []byte) []byte {
	k := make([]byte, 1+len(key))
	k[0] = prefix
	copy(k[1:], key)
	return k
}

func removeTrades(batch *leveldb.Batch, trades []Trade) {
	for _, t := range trades {
		batch.Delete(digestKey(TradesKeyPrefix, t.TransactionID))
	}
}

func updateLastHeight(batch *leveldb.Batch, height int) {
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(height))
	batch.Put(lastHeightKey, v)
}

func contains(a []uint32, v uint32) bool {
	for _, x := range a {
		if v == x {
			return true
		}
	}
	return false
}

func min(a []uint32) uint32 {
	r := uint32(math.MaxUint32)
	for _, x := range a {
		if x < r {
			r = x
		}
	}
	return r
}
