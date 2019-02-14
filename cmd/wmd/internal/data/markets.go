package data

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type MarketID struct {
	AmountAsset crypto.Digest
	PriceAsset  crypto.Digest
}

func (id *MarketID) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2*crypto.DigestSize)
	copy(buf, id.AmountAsset[:])
	copy(buf[crypto.DigestSize:], id.PriceAsset[:])
	return buf, nil
}

func (id *MarketID) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 2*crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for MarketID", l)
	}
	copy(id.AmountAsset[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	copy(id.PriceAsset[:], data[:crypto.DigestSize])
	return nil
}

type Market struct {
	FirstTradeTimestamp uint64
	LastTradeTimestamp  uint64
	TotalTradesCount    int
}

func (md *Market) UpdateFromTrade(t Trade) {
	if md.FirstTradeTimestamp == 0 || md.FirstTradeTimestamp > t.Timestamp {
		md.FirstTradeTimestamp = t.Timestamp
	}
	if md.LastTradeTimestamp < t.Timestamp {
		md.LastTradeTimestamp = t.Timestamp
	}
	md.TotalTradesCount++
}

func (md *Market) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8+8+4)
	binary.BigEndian.PutUint64(buf, md.FirstTradeTimestamp)
	binary.BigEndian.PutUint64(buf[8:], md.LastTradeTimestamp)
	binary.BigEndian.PutUint32(buf[8+8:], uint32(md.TotalTradesCount))
	return buf, nil
}

func (md *Market) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 8+8+4 {
		return errors.Errorf("%d is not enough bytes for Market", l)
	}
	md.FirstTradeTimestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	md.LastTradeTimestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	md.TotalTradesCount = int(binary.BigEndian.Uint32(data))
	return nil
}

type MarketInfo struct {
	TickerInfo
	TotalTrades   int    `json:"totalTrades"`
	FirstTradeDay uint64 `json:"firstTradeDay"`
	LastTradeDay  uint64 `json:"lastTradeDay"`
}

func NewMarketInfo(ticker TickerInfo, md Market) MarketInfo {
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
	return uint64(TimeFromMilliseconds(ts).Truncate(24*time.Hour).UnixNano() / 1000000)
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
