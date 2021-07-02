package state

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type tradeKey struct {
	id crypto.Digest
}

func (k *tradeKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = tradeKeyPrefix
	copy(buf[1:], k.id[:])
	return buf
}

type tradeHistoryKey struct {
	height uint32
	trade  crypto.Digest
}

func (k tradeHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+crypto.DigestSize)
	buf[0] = tradeHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], k.trade[:])
	return buf
}

func (k *tradeHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for tradeHistoryKey", l)
	}
	if data[0] != tradeHistoryKeyPrefix {
		return errors.New("invalid prefix for tradeHistoryKey")
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.trade[:], data[1+4:1+4+crypto.DigestSize])
	return nil
}

type marketTradeKey struct {
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	timeFrame   uint32
	trade       crypto.Digest
}

func (k marketTradeKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+4+crypto.DigestSize)
	buf[0] = marketTradesKeyPrefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	binary.BigEndian.PutUint32(buf[1+2*crypto.DigestSize:], k.timeFrame)
	copy(buf[1+2*crypto.DigestSize+4:], k.trade[:])
	return buf
}

func (k *marketTradeKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+3*crypto.DigestSize+4 {
		return errors.Errorf("%d is not enough bytes for marketTradeKey", l)
	}
	if data[0] != marketTradesKeyPrefix {
		return errors.Errorf("invalid prefix for marketTradeKey")
	}
	copy(k.amountAsset[:], data[1:1+crypto.DigestSize])
	copy(k.priceAsset[:], data[1+crypto.DigestSize:1+2*crypto.DigestSize])
	k.timeFrame = binary.BigEndian.Uint32(data[1+2*crypto.DigestSize:])
	copy(k.trade[:], data[1+2*crypto.DigestSize+4:1+2*crypto.DigestSize+4+crypto.DigestSize])
	return nil
}

type marketTradePartialKey struct {
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	timeFrame   uint32
}

func (k *marketTradePartialKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+4)
	buf[0] = marketTradesKeyPrefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	binary.BigEndian.PutUint32(buf[1+2*crypto.DigestSize:], k.timeFrame)
	return buf
}

func putTrades(bs *blockState, batch *leveldb.Batch, height uint32, trades []data.Trade) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to put trades") }
	marketsUpdated := make(map[marketHistoryKey]struct{})
	var earliestTimeFrame uint32 = math.MaxUint32
	affectedTimeFrames := make([]uint32, 0)
	for _, t := range trades {
		tk := tradeKey{t.TransactionID}
		b1, err := t.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		thk := tradeHistoryKey{height: height, trade: t.TransactionID}
		batch.Put(tk.bytes(), b1)
		batch.Put(thk.bytes(), nil)

		tf := data.TimeFrameFromTimestampMS(t.Timestamp)
		if tf < earliestTimeFrame {
			earliestTimeFrame = tf
		}
		affectedTimeFrames = append(affectedTimeFrames, tf)

		tk2 := marketTradeKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, timeFrame: tf, trade: t.TransactionID}
		batch.Put(tk2.bytes(), nil)
		tk3 := addressTradesKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, address: t.Buyer, trade: t.TransactionID}
		batch.Put(tk3.bytes(), nil)
		tk4 := addressTradesKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, address: t.Seller, trade: t.TransactionID}
		batch.Put(tk4.bytes(), nil)

		// update candles information
		candle, ck, err := bs.candle(t.AmountAsset, t.PriceAsset, tf)
		if err != nil {
			return wrapError(err)
		}
		candle.UpdateFromTrade(t)
		bs.candles[ck] = candle
		cb, err := candle.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(ck.bytes(), cb)
		chk := candleHistoryKey{timeFrame: tf, amountAsset: ck.amountAsset, priceAsset: ck.priceAsset}
		batch.Put(chk.bytes(), nil)

		// Update market information with new trades
		market, mk, err := bs.market(t.AmountAsset, t.PriceAsset)
		if err != nil {
			return wrapError(err)
		}
		mhk := marketHistoryKey{height: height, amountAsset: t.AmountAsset, priceAsset: t.PriceAsset}
		if _, ok := marketsUpdated[mhk]; !ok { // Update market history only for the first update of the block
			mb, err := market.MarshalBinary()
			if err != nil {
				return wrapError(err)
			}
			batch.Put(mhk.bytes(), mb)
			marketsUpdated[mhk] = struct{}{}
		}
		market.UpdateFromTrade(t)
		bs.markets[mk] = market
		mb, err := market.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(mk.bytes(), mb)
	}
	if len(trades) != 0 { // If block is non-empty should update earliestTimeFrame
		tfk := uint32Key{prefix: earliestTimeFrameKeyPrefix, key: height}
		v := make([]byte, 4)
		binary.BigEndian.PutUint32(v, earliestTimeFrame)
		batch.Put(tfk.bytes(), v)
		err := updateEarliestHeights(bs, batch, affectedTimeFrames, height)
		if err != nil {
			return wrapError(err)
		}
	}
	return nil
}

func rollbackTrades(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to rollback trades") }
	//remove Trades that comes with the removed blocks
	s := uint32Key{prefix: tradeHistoryKeyPrefix, key: removeHeight}
	l := uint32Key{prefix: tradeHistoryKeyPrefix, key: math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	minTradeTimestamp := uint64(math.MaxUint64)
	if it.Last() {
		for {
			var thk tradeHistoryKey
			err := thk.fromBytes(it.Key())
			if err != nil {
				return wrapError(err)
			}
			tk := tradeKey{id: thk.trade}
			tb, err := snapshot.Get(tk.bytes(), nil)
			if err != nil {
				return wrapError(err)
			}
			var t data.Trade
			err = t.UnmarshalBinary(tb)
			if err != nil {
				return wrapError(err)
			}
			if t.Timestamp < minTradeTimestamp {
				minTradeTimestamp = t.Timestamp
			}
			batch.Delete(thk.bytes())
			batch.Delete(tk.bytes())
			tf := data.TimeFrameFromTimestampMS(t.Timestamp)
			tk2 := marketTradeKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, timeFrame: tf, trade: t.TransactionID}
			batch.Delete(tk2.bytes())
			tk3 := addressTradesKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, address: t.Buyer, trade: t.TransactionID}
			batch.Delete(tk3.bytes())
			tk4 := addressTradesKey{amountAsset: t.AmountAsset, priceAsset: t.PriceAsset, address: t.Seller, trade: t.TransactionID}
			batch.Delete(tk4.bytes())
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	//remove candles affected by trades at height, if they are not removed yet
	tf := data.TimeFrameFromTimestampMS(minTradeTimestamp)
	s = uint32Key{prefix: candleHistoryKeyPrefix, key: tf}
	l = uint32Key{prefix: candleHistoryKeyPrefix, key: math.MaxUint32}
	it = snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	for it.Next() {
		var chk candleHistoryKey
		err := chk.fromBytes(it.Key())
		if err != nil {
			return wrapError(err)
		}
		ck := candleKey{timeFrame: chk.timeFrame, amountAsset: chk.amountAsset, priceAsset: chk.priceAsset}
		batch.Delete(it.Key())
		batch.Delete(ck.bytes())
	}
	it.Release()
	//bring back or remove previous state of markets
	downgradeMarkets := make(map[marketKey]data.Market)
	removeMarkets := make([]marketKey, 0)
	s = uint32Key{prefix: marketHistoryKeyPrefix, key: removeHeight}
	l = uint32Key{prefix: marketHistoryKeyPrefix, key: math.MaxUint32}
	it = snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	if it.Last() {
		for {
			var mhk marketHistoryKey
			var pm data.Market
			var mk marketKey
			err := mhk.fromBytes(it.Key())
			if err != nil {
				return wrapError(err)
			}
			err = pm.UnmarshalBinary(it.Value())
			if err != nil {
				return wrapError(err)
			}
			mk = marketKey{amountAsset: mhk.amountAsset, priceAsset: mhk.priceAsset}
			if pm.TotalTradesCount == 0 {
				removeMarkets = append(removeMarkets, mk)
				delete(downgradeMarkets, mk)
			} else {
				downgradeMarkets[mk] = pm
			}
			batch.Delete(it.Key())
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	//remove time frames
	s = uint32Key{prefix: earliestTimeFrameKeyPrefix, key: removeHeight}
	l = uint32Key{prefix: earliestTimeFrameKeyPrefix, key: math.MaxUint32}
	it = snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	for it.Next() {
		tf := binary.BigEndian.Uint32(it.Value())
		batch.Delete(it.Key())
		tfk := uint32Key{prefix: earliestHeightKeyPrefix, key: tf}
		batch.Delete(tfk.bytes())
	}
	it.Release()
	for k, v := range downgradeMarkets {
		b, err := v.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(k.bytes(), b)
	}
	for _, k := range removeMarkets {
		batch.Delete(k.bytes())
	}
	return nil
}

func trade(snapshot *leveldb.Snapshot, id crypto.Digest) (data.Trade, error) {
	k := tradeKey{id}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		return data.Trade{}, errors.Wrapf(err, "failed to locate trade '%s'", id.String())
	}
	var t data.Trade
	err = t.UnmarshalBinary(b)
	if err != nil {
		return data.Trade{}, errors.Wrap(err, "failed to read trade")
	}
	return t, nil
}

func trades(snapshot *leveldb.Snapshot, amountAsset, priceAsset crypto.Digest, from, to uint64, limit int) ([]data.Trade, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to load trades")
	}
	f := data.TimeFrameFromTimestampMS(from)
	t := data.TimeFrameFromTimestampMS(to)
	s := marketTradePartialKey{amountAsset, priceAsset, f}
	l := marketTradePartialKey{amountAsset, priceAsset, t + 1}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	c := 0
	var trades []data.Trade
	if it.Last() {
		for {
			if c >= limit {
				break
			}
			b := it.Key()
			var k marketTradeKey
			err := k.fromBytes(b)
			if err != nil {
				return nil, wrapError(err)
			}
			t, err := trade(snapshot, k.trade)
			if err != nil {
				return nil, wrapError(err)
			}
			if t.Timestamp >= from && t.Timestamp <= to {
				trades = append(trades, t)
				c++
			}
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	return trades, nil
}

type addressTradesKey struct {
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	address     proto.WavesAddress
	trade       crypto.Digest
}

func (k addressTradesKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+proto.WavesAddressSize+crypto.DigestSize)
	buf[0] = addressTradesKeyPrefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	copy(buf[1+2*crypto.DigestSize:], k.address[:])
	copy(buf[1+2*crypto.DigestSize+proto.WavesAddressSize:], k.trade[:])
	return buf
}

func (k *addressTradesKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+3*crypto.DigestSize+proto.WavesAddressSize {
		return errors.Errorf("%d is not enough bytes for addressTradesKey", l)
	}
	if data[0] != addressTradesKeyPrefix {
		return errors.New("invalid prefix for addressTradesKey")
	}
	copy(k.amountAsset[:], data[1:1+crypto.DigestSize])
	copy(k.priceAsset[:], data[1+crypto.DigestSize:1+2*crypto.DigestSize])
	copy(k.address[:], data[1+2*crypto.DigestSize:1+2*crypto.DigestSize+proto.WavesAddressSize])
	copy(k.trade[:], data[1+2*crypto.DigestSize+proto.WavesAddressSize:1+3*crypto.DigestSize+proto.WavesAddressSize])
	return nil
}

func addressTrades(snapshot *leveldb.Snapshot, amountAsset, priceAsset crypto.Digest, address proto.WavesAddress, limit int) ([]data.Trade, error) {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to collect trades for address '%s'", address.String())
	}
	s := addressTradesKey{amountAsset, priceAsset, address, minDigest}
	l := addressTradesKey{amountAsset, priceAsset, address, maxDigest}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	c := 0
	var trades []data.Trade
	if it.Last() {
		for {
			if c >= limit {
				break
			}
			b := it.Key()
			var k addressTradesKey
			err := k.fromBytes(b)
			if err != nil {
				return nil, wrapError(err)
			}
			t, err := trade(snapshot, k.trade)
			if err != nil {
				return nil, wrapError(err)
			}
			trades = append(trades, t)
			c++
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	return trades, nil
}

type candleKey struct {
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
	timeFrame   uint32
}

func (k candleKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize+4)
	buf[0] = candleKeyPrefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	binary.BigEndian.PutUint32(buf[1+2*crypto.DigestSize:], k.timeFrame)
	return buf
}

type candleHistoryKey struct {
	timeFrame   uint32
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
}

func (k candleHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+2*crypto.DigestSize)
	buf[0] = candleHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.timeFrame)
	copy(buf[1+4:], k.amountAsset[:])
	copy(buf[1+4+crypto.DigestSize:], k.priceAsset[:])
	return buf
}

func (k *candleHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+2*crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for candleHistoryKey", l)
	}
	if data[0] != candleHistoryKeyPrefix {
		return errors.New("invalid prefix for candleHistoryKey")
	}
	k.timeFrame = binary.BigEndian.Uint32(data[1:])
	copy(k.amountAsset[:], data[1+4:1+4+crypto.DigestSize])
	copy(k.priceAsset[:], data[1+4+crypto.DigestSize:1+4+2*crypto.DigestSize])
	return nil
}

func candles(snapshot *leveldb.Snapshot, amountAsset, priceAsset crypto.Digest, start, stop uint32, limit int) ([]data.Candle, error) {
	sk := candleKey{amountAsset, priceAsset, start}
	ek := candleKey{amountAsset, priceAsset, stop}
	r := make([]data.Candle, 0)
	it := snapshot.NewIterator(&util.Range{Start: sk.bytes(), Limit: ek.bytes()}, nil)
	defer it.Release()
	var c data.Candle
	i := 0
	for it.Next() {
		err := c.UnmarshalBinary(it.Value())
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

type marketKey struct {
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
}

func (k marketKey) bytes() []byte {
	buf := make([]byte, 1+2*crypto.DigestSize)
	buf[0] = marketKeyPrefix
	copy(buf[1:], k.amountAsset[:])
	copy(buf[1+crypto.DigestSize:], k.priceAsset[:])
	return buf
}

type marketHistoryKey struct {
	height      uint32
	amountAsset crypto.Digest
	priceAsset  crypto.Digest
}

func (k marketHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+2*crypto.DigestSize)
	buf[0] = marketHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], k.amountAsset[:])
	copy(buf[1+4+crypto.DigestSize:], k.priceAsset[:])
	return buf
}

func (k *marketHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+2*crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for marketHistoryKey", l)
	}
	if data[0] != marketHistoryKeyPrefix {
		return errors.New("incorrect prefix for marketHistoryKey")
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.amountAsset[:], data[1+4:1+4+crypto.DigestSize])
	copy(k.priceAsset[:], data[1+4+crypto.DigestSize:1+4+2*crypto.DigestSize])
	return nil
}

func marketsMap(snapshot *leveldb.Snapshot) (map[data.MarketID]data.Market, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to collect markets") }
	s := marketKey{data.WavesID, data.WavesID}
	l := marketKey{maxDigest, maxDigest}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	r := make(map[data.MarketID]data.Market)
	for it.Next() {
		k := it.Key()
		var m data.MarketID
		err := m.UnmarshalBinary(k[1:])
		if err != nil {
			return nil, wrapError(err)
		}
		var md data.Market
		err = md.UnmarshalBinary(it.Value())
		if err != nil {
			return nil, wrapError(err)
		}
		r[m] = md
	}
	it.Release()
	return r, nil
}

func earliestTimeFrame(snapshot *leveldb.Snapshot, height uint32) (uint32, bool) {
	s := uint32Key{prefix: earliestTimeFrameKeyPrefix, key: height}
	l := uint32Key{prefix: earliestTimeFrameKeyPrefix, key: math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	defer it.Release()
	if it.Next() {
		tf := binary.BigEndian.Uint32(it.Value())
		return tf, true
	}
	return 0, false
}

func updateEarliestHeights(bs *blockState, batch *leveldb.Batch, timeFrames []uint32, height uint32) error {
	for _, tf := range timeFrames {
		h, k, err := bs.earliestHeight(tf)
		if err != nil {
			return err
		}
		if height < h {
			bs.earliestHeights[k] = height
			v := make([]byte, 4)
			binary.BigEndian.PutUint32(v, height)
			batch.Put(k.bytes(), v)
		}
	}
	return nil
}

func earliestAffectedHeight(snapshot *leveldb.Snapshot, timeFrame uint32) (uint32, error) {
	k := uint32Key{prefix: earliestHeightKeyPrefix, key: timeFrame}
	b, err := snapshot.Get(k.bytes(), nil)
	if err != nil {
		return 0, err
	}
	h := binary.BigEndian.Uint32(b)
	return h, nil
}
