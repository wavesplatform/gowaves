package state

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestTradesState1(t *testing.T) {
	db, closeDB := openDB(t, "wmd-trades-state-db")
	defer closeDB()

	b, err := proto.NewAddressFromString("3P4KdaNYJq7BBcsgrsAPArc66LyLQAQvJc2")
	s, err := proto.NewAddressFromString("3PAmhzHgxzxqVttGFRgVCFUFHoGHqmuchec")
	m, err := proto.NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	aa := data.WavesID
	pa, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	tID1, err := randomDigest()
	assert.NoError(t, err)
	ts1 := uint64(1548230341666)
	t1 := data.Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: tID1, OrderType: proto.Buy, Buyer: b, Seller: s, Matcher: m, Price: 12345, Amount: 67890, Timestamp: ts1}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putTrades(bs, batch, 1, []data.Trade{t1})
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		stf := data.TimeFrameFromTimestampMS(ts1) - 1
		ltf := data.TimeFrameFromTimestampMS(ts1) + 1
		tds, err := trades(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(tds))
		assert.ElementsMatch(t, []data.Trade{t1}, tds)
		cs, err := candles(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cs))
		ec := data.NewCandleFromTimestamp(ts1)
		ec.Open, ec.High, ec.Low, ec.Close, ec.Average = 12345, 12345, 12345, 12345, 12345
		ec.Volume = 67890
		ec.MinTimestamp, ec.MaxTimestamp = ts1, ts1
		assert.Equal(t, ec, cs[0])
		msm, err := marketsMap(snapshot)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(msm))
		em := data.Market{FirstTradeTimestamp: ts1, LastTradeTimestamp: ts1, TotalTradesCount: 1}
		mk := data.MarketID{AmountAsset: aa, PriceAsset: pa}
		assert.Equal(t, em, msm[mk])
	}

	tID2, err := randomDigest()
	assert.NoError(t, err)
	ts2 := uint64(1548230345666)
	t2 := data.Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: tID2, OrderType: proto.Buy, Buyer: b, Seller: s, Matcher: m, Price: 12356, Amount: 67887, Timestamp: ts2}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putTrades(bs, batch, 2, []data.Trade{t2})
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		stf := data.TimeFrameFromTimestampMS(ts2) - 1
		ltf := data.TimeFrameFromTimestampMS(ts2) + 1
		tds, err := trades(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tds))
		assert.ElementsMatch(t, []data.Trade{t1, t2}, tds)
		cs, err := candles(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cs))
		ec := data.NewCandleFromTimestamp(ts2)
		ec.Open, ec.High, ec.Low, ec.Close, ec.Average = 12345, 12356, 12345, 12356, 12350
		ec.Volume = 135777
		ec.MinTimestamp, ec.MaxTimestamp = ts1, ts2
		assert.Equal(t, ec, cs[0])
		msm, err := marketsMap(snapshot)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(msm))
		em := data.Market{FirstTradeTimestamp: ts1, LastTradeTimestamp: ts2, TotalTradesCount: 2}
		mk := data.MarketID{AmountAsset: aa, PriceAsset: pa}
		assert.Equal(t, em, msm[mk])
	}

	tID3, err := randomDigest()
	assert.NoError(t, err)
	tID4, err := randomDigest()
	assert.NoError(t, err)
	ts3 := uint64(1548230642000)
	ts4 := uint64(1548230643000)
	t3 := data.Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: tID3, OrderType: proto.Sell, Buyer: b, Seller: s, Matcher: m, Price: 12300, Amount: 1000, Timestamp: ts3}
	t4 := data.Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: tID4, OrderType: proto.Sell, Buyer: b, Seller: s, Matcher: m, Price: 12200, Amount: 2000, Timestamp: ts4}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putTrades(bs, batch, 3, []data.Trade{t3, t4})
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		stf := data.TimeFrameFromTimestampMS(ts1) - 1
		ltf := data.TimeFrameFromTimestampMS(ts4) + 1
		tds, err := trades(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(tds))
		assert.ElementsMatch(t, []data.Trade{t1, t2, t3, t4}, tds)
		cs, err := candles(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cs))
		ec1 := data.NewCandleFromTimestamp(ts2)
		ec1.Open, ec1.High, ec1.Low, ec1.Close, ec1.Average = 12345, 12356, 12345, 12356, 12350
		ec1.Volume = 135777
		ec1.MinTimestamp, ec1.MaxTimestamp = ts1, ts2
		ec2 := data.NewCandleFromTimestamp(ts4)
		ec2.Open, ec2.High, ec2.Low, ec2.Close, ec2.Average = 12300, 12300, 12200, 12200, 12233
		ec2.Volume = 3000
		ec2.MinTimestamp, ec2.MaxTimestamp = ts3, ts4
		assert.ElementsMatch(t, []data.Candle{ec1, ec2}, cs)
		msm, err := marketsMap(snapshot)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(msm))
		em := data.Market{FirstTradeTimestamp: ts1, LastTradeTimestamp: ts4, TotalTradesCount: 4}
		mk := data.MarketID{AmountAsset: aa, PriceAsset: pa}
		assert.Equal(t, em, msm[mk])
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackTrades(snapshot, batch, 3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		stf := data.TimeFrameFromTimestampMS(ts2) - 1
		ltf := data.TimeFrameFromTimestampMS(ts2) + 1
		tds, err := trades(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tds))
		assert.ElementsMatch(t, []data.Trade{t1, t2}, tds)
		cs, err := candles(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cs))
		ec := data.NewCandleFromTimestamp(ts2)
		ec.Open, ec.High, ec.Low, ec.Close, ec.Average = 12345, 12356, 12345, 12356, 12350
		ec.Volume = 135777
		ec.MinTimestamp, ec.MaxTimestamp = ts1, ts2
		assert.Equal(t, ec, cs[0])
		msm, err := marketsMap(snapshot)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(msm))
		em := data.Market{FirstTradeTimestamp: ts1, LastTradeTimestamp: ts2, TotalTradesCount: 2}
		mk := data.MarketID{AmountAsset: aa, PriceAsset: pa}
		assert.Equal(t, em, msm[mk])
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackTrades(snapshot, batch, 1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		stf := data.TimeFrameFromTimestampMS(ts1) - 1
		ltf := data.TimeFrameFromTimestampMS(ts1) + 1
		tds, err := trades(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tds))
		cs, err := candles(snapshot, aa, pa, stf, ltf, 100)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cs))
		msm, err := marketsMap(snapshot)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(msm))
	}
}

func randomDigest() (crypto.Digest, error) {
	d := crypto.Digest{}
	_, err := rand.Read(d[:])
	if err != nil {
		return crypto.Digest{}, err
	}
	return d, nil
}
