package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"time"
)

const (
	MarketIDSize   = 2 * crypto.DigestSize
	MarketDataSize = 8 + 8 + 4
)

type MarketID struct {
	AmountAsset crypto.Digest
	PriceAsset  crypto.Digest
}

func (id *MarketID) MarshalBinary() ([]byte, error) {
	buf := make([]byte, MarketIDSize)
	copy(buf, id.AmountAsset[:])
	copy(buf[crypto.DigestSize:], id.PriceAsset[:])
	return buf, nil
}

func (id *MarketID) UnmarshalBinary(data []byte) error {
	if l := len(data); l < MarketIDSize {
		return errors.Errorf("%d is not enough bytes for MarketID, expected %d", l, MarketIDSize)
	}
	copy(id.AmountAsset[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	copy(id.PriceAsset[:], data[:crypto.DigestSize])
	return nil
}

type MarketData struct {
	FirstTradeTimestamp uint64
	LastTradeTimestamp  uint64
	TotalTradesCount    int
}

func (md *MarketData) MarshalBinary() ([]byte, error) {
	buf := make([]byte, MarketDataSize)
	binary.BigEndian.PutUint64(buf, md.FirstTradeTimestamp)
	binary.BigEndian.PutUint64(buf[8:], md.LastTradeTimestamp)
	binary.BigEndian.PutUint32(buf[8+8:], uint32(md.TotalTradesCount))
	return buf, nil
}

func (md *MarketData) UnmarshalBinary(data []byte) error {
	if l := len(data); l < MarketDataSize {
		return errors.Errorf("%d is not enough bytes for MarketData, expected %d", l, MarketDataSize)
	}
	md.FirstTradeTimestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	md.LastTradeTimestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	md.TotalTradesCount = int(binary.BigEndian.Uint32(data))
	return nil
}

func (md *MarketData) UpdateFromTrade(t Trade) {
	if md.FirstTradeTimestamp == 0 || md.FirstTradeTimestamp > t.Timestamp {
		md.FirstTradeTimestamp = t.Timestamp
	}
	if md.LastTradeTimestamp < t.Timestamp {
		md.LastTradeTimestamp = t.Timestamp
	}
	md.TotalTradesCount++
}

type MarketInfo struct {
	TickerInfo
	TotalTrades   int    `json:"totalTrades"`
	FirstTradeDay uint64 `json:"firstTradeDay"`
	LastTradeDay  uint64 `json:"lastTradeDay"`
}

func NewMarketInfo(ticker TickerInfo, md MarketData) MarketInfo {
	return MarketInfo{
		TickerInfo:    ticker,
		TotalTrades:   md.TotalTradesCount,
		FirstTradeDay: StartOfTheDayMilliseconds(md.FirstTradeTimestamp),
		LastTradeDay:  StartOfTheDayMilliseconds(md.LastTradeTimestamp),
	}
}

func TimeFromMilliseconds(ms uint64) time.Time {
	s := ms / 1000
	ns := (ms % 1000) * 1000000
	return time.Unix(int64(s), int64(ns))
}

func StartOfTheDayMilliseconds(ts uint64) uint64 {
	return uint64(TimeFromMilliseconds(ts).Truncate(24 * time.Hour).UnixNano() / 1000000)
}

type ByMarkets []MarketInfo

func (a ByMarkets) Len() int {
	return len(a)
}

func (a ByMarkets) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByMarkets) Less(i, j int) bool {
	x := a[i].Symbol
	y := a[j].Symbol

	switch {
	case x == "" && y != "":
		return false
	case x != "" && y == "":
		return true
	case x != "" && y != "":
		return x < y
	default:
		return false
	}
}


