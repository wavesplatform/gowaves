package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"math/big"
)

const (
	CandleSize = 6 * 8
	Second     = 1000
	Minute     = 60 * Second
	TimeFrame  = 5 * Minute
	Hour       = 60 * Minute
	Day        = 24 * Hour
)

type Candle struct {
	Open         uint64
	High         uint64
	Low          uint64
	Close        uint64
	Average      uint64
	Volume       uint64
	minTimestamp uint64
	maxTimestamp uint64
}

func NewCandle(ts uint64) Candle {
	b := StartOfTheFrame(ts)
	return Candle{minTimestamp: b + TimeFrame, maxTimestamp: b}
}

func (c *Candle) UpdateFromTrade(t Trade) {
	if t.Timestamp < c.minTimestamp {
		c.Open = t.Price
		c.minTimestamp = t.Timestamp
	}
	if t.Timestamp > c.maxTimestamp {
		c.Close = t.Price
		c.maxTimestamp = t.Timestamp
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
	return nil
}

func StartOfTheDay(ts uint64) uint64 {
	return (ts / Day) * Day
}

func StartOfTheFrame(ts uint64) uint64 {
	b := StartOfTheDay(ts)
	off := (ts - b) / TimeFrame
	return b + off*TimeFrame
}
