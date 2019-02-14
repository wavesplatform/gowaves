package data

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"math"
	"math/big"
)

const (
	CandleSize       = 8 * 8
	Second           = 1000
	Minute           = 60 * Second
	DefaultTimeFrame = 5
	TimeFrame        = DefaultTimeFrame * Minute
)

type Candle struct {
	Open         uint64
	High         uint64
	Low          uint64
	Close        uint64
	Average      uint64
	Volume       uint64
	MinTimestamp uint64
	MaxTimestamp uint64
}

func NewCandleFromTimestamp(ts uint64) Candle {
	b := TimestampMSFromTimeFrame(TimeFrameFromTimestampMS(ts))
	return Candle{MinTimestamp: b + TimeFrame, MaxTimestamp: b} //Initialize in opposite to support update
}

func NewCandleFromTimeFrame(tf uint32) Candle {
	b := TimestampMSFromTimeFrame(tf)
	return Candle{MinTimestamp: b + TimeFrame, MaxTimestamp: b} //Initialize in opposite to support update
}

func (c *Candle) UpdateFromTrade(t Trade) {
	if c.MinTimestamp == 0 || t.Timestamp < c.MinTimestamp {
		c.Open = t.Price
		c.MinTimestamp = t.Timestamp
	}
	if c.MaxTimestamp == 0 || t.Timestamp > c.MaxTimestamp {
		c.Close = t.Price
		c.MaxTimestamp = t.Timestamp
	}
	if t.Price > c.High {
		c.High = t.Price
	}
	if c.Low == 0 || t.Price < c.Low {
		c.Low = t.Price
	}
	if t.Amount > 0 {
		v := c.Volume + t.Amount
		var a1 big.Int
		var v1 big.Int
		var v2 big.Int
		a1.SetUint64(c.Average)
		v1.SetUint64(c.Volume)
		v2.SetUint64(v)

		var p big.Int
		var a big.Int
		p.SetUint64(t.Price)
		a.SetUint64(t.Amount)

		var av big.Int
		av.Mul(&a1, &v1)

		var pa big.Int
		pa.Mul(&p, &a)

		var x big.Int
		x.Add(&av, &pa)

		var y big.Int
		y.Div(&x, &v2)

		c.Average = y.Uint64()
		c.Volume = v
	}
}

func (c *Candle) Combine(x Candle) {
	if c.MinTimestamp == 0 || x.MinTimestamp < c.MinTimestamp {
		c.Open = x.Open
		c.MinTimestamp = x.MinTimestamp
	}
	if x.MaxTimestamp > c.MaxTimestamp {
		c.Close = x.Close
		c.MaxTimestamp = x.MaxTimestamp
	}
	if x.High > c.High {
		c.High = x.High
	}
	if c.Low == 0 || x.Low < c.Low {
		c.Low = x.Low
	}
	if x.Volume > 0 {
		var a1 big.Int
		var v1 big.Int
		var a2 big.Int
		var v2 big.Int
		a1.SetUint64(c.Average)
		v1.SetUint64(c.Volume)
		a2.SetUint64(x.Average)
		v2.SetUint64(x.Volume)
		var tv big.Int
		tv.Add(&v1, &v2)
		var a1v1 big.Int
		a1v1.Mul(&a1, &v1)
		var a2v2 big.Int
		a2v2.Mul(&a2, &v2)
		var s big.Int
		s.Add(&a1v1, &a2v2)
		var r big.Int
		r.Div(&s, &tv)
		c.Average = r.Uint64()
		c.Volume = tv.Uint64()
	}
}

func (c *Candle) MarshalBinary() ([]byte, error) {
	buf := make([]byte, CandleSize)
	p := 0
	binary.BigEndian.PutUint64(buf[p:], c.Open)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.High)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.Low)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.Close)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.Average)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.Volume)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.MinTimestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], c.MaxTimestamp)
	return buf, nil
}

func (c *Candle) UnmarshalBinary(data []byte) error {
	if l := len(data); l < CandleSize {
		return errors.Errorf("%d is not enough bytes for Candle, expected %d", l, CandleSize)
	}
	c.Open = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.High = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.Low = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.Close = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.Average = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.Volume = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.MinTimestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.MaxTimestamp = binary.BigEndian.Uint64(data)
	return nil
}

func TimeFrameFromTimestampMS(ts uint64) uint32 {
	return uint32(ts / TimeFrame)
}

func TimestampMSFromTimeFrame(tf uint32) uint64 {
	return uint64(tf) * TimeFrame
}

func ScaleTimeFrame(tf uint32, scale int) uint32 {
	s := uint32(scale)
	return (tf / s) * s
}

type CandleInfo struct {
	Timestamp   uint64  `json:"timestamp"`
	Open        Decimal `json:"open"`
	High        Decimal `json:"high"`
	Low         Decimal `json:"low"`
	Close       Decimal `json:"close"`
	Average     Decimal `json:"vwap"`
	Volume      Decimal `json:"volume"`
	PriceVolume Decimal `json:"priceVolume"`
	Confirmed   bool    `json:"confirmed"`
}

func EmptyCandleInfo(amountAssetDecimals, priceAssetDecimals uint, timestamp uint64) CandleInfo {
	return CandleInfo{
		Timestamp:   timestamp,
		Open:        Decimal{0, priceAssetDecimals},
		High:        Decimal{0, priceAssetDecimals},
		Low:         Decimal{0, priceAssetDecimals},
		Close:       Decimal{0, priceAssetDecimals},
		Average:     Decimal{0, priceAssetDecimals},
		Volume:      Decimal{0, amountAssetDecimals},
		PriceVolume: Decimal{0, priceAssetDecimals},
		Confirmed:   true,
	}
}

func CandleInfoFromCandle(candle Candle, amountAssetDecimals, priceAssetDecimals uint, timeFrameScale int) CandleInfo {
	tf := ScaleTimeFrame(TimeFrameFromTimestampMS(candle.MinTimestamp), timeFrameScale)
	pv := priceVolume(candle.Average, candle.Volume, amountAssetDecimals)
	return CandleInfo{
		Timestamp:   TimestampMSFromTimeFrame(tf),
		Open:        Decimal{candle.Open, priceAssetDecimals},
		High:        Decimal{candle.High, priceAssetDecimals},
		Low:         Decimal{candle.Low, priceAssetDecimals},
		Close:       Decimal{candle.Close, priceAssetDecimals},
		Average:     Decimal{candle.Average, priceAssetDecimals},
		Volume:      Decimal{candle.Volume, amountAssetDecimals},
		PriceVolume: Decimal{pv, priceAssetDecimals},
		Confirmed:   true,
	}
}

func priceVolume(average, volume uint64, amountAssetDecimals uint) uint64 {
	var a big.Int
	var v big.Int
	var av big.Int
	var s big.Int
	var pv big.Int
	a.SetUint64(average)
	v.SetUint64(volume)
	av.Mul(&a, &v)
	s.SetUint64(uint64(math.Pow10(int(amountAssetDecimals))))
	pv.Div(&av, &s)
	return pv.Uint64()
}

type ByCandlesTimestampBackward []CandleInfo

func (a ByCandlesTimestampBackward) Len() int {
	return len(a)
}

func (a ByCandlesTimestampBackward) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByCandlesTimestampBackward) Less(i, j int) bool {
	return a[i].Timestamp > a[j].Timestamp
}
