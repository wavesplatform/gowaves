package data

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestStartOfTheFrame(t *testing.T) {
	ts := uint64(1542711749 * Second)
	assert.Equal(t, 1542711600000, int(TimestampMSFromTimeFrame(TimeFrameFromTimestampMS(ts))))
}

func TestScaleTimeFrame(t *testing.T) {
	ts := uint64(1545138821 * Second)
	tf := TimeFrameFromTimestampMS(ts)
	assert.Equal(t, 5150462, int(tf))
	assert.Equal(t, 5150460, int(ScaleTimeFrame(tf, 3)))
	assert.Equal(t, 5150460, int(ScaleTimeFrame(tf, 6)))
	ts = 1545139620 * Second
	tf = TimeFrameFromTimestampMS(ts)
	assert.Equal(t, 5150465, int(tf))
	assert.Equal(t, 5150463, int(ScaleTimeFrame(tf, 3)))
	assert.Equal(t, 5150460, int(ScaleTimeFrame(tf, 6)))
}

func TestCandle_UpdateFromTrade(t *testing.T) {
	ts1 := uint64(1542712337 * Second)
	tr1 := Trade{Timestamp: ts1, Price: 1234567, Amount: 0}
	ts2 := uint64(1542712237 * Second)
	tr2 := Trade{Timestamp: ts2, Price: 7654321, Amount: 0}
	c := NewCandleFromTimestamp(tr1.Timestamp)
	assert.Equal(t, 1542712200000, int(c.MaxTimestamp))
	assert.Equal(t, 1542712500000, int(c.MinTimestamp))
	c.UpdateFromTrade(tr1)
	assert.Equal(t, ts1, c.MinTimestamp)
	assert.Equal(t, ts1, c.MaxTimestamp)
	assert.Equal(t, 1234567, int(c.Open))
	assert.Equal(t, 1234567, int(c.Close))
	assert.Equal(t, 1234567, int(c.High))
	assert.Equal(t, 1234567, int(c.Low))
	c.UpdateFromTrade(tr2)
	assert.Equal(t, ts2, c.MinTimestamp)
	assert.Equal(t, ts1, c.MaxTimestamp)
	assert.Equal(t, 7654321, int(c.Open))
	assert.Equal(t, 1234567, int(c.Close))
	assert.Equal(t, 7654321, int(c.High))
	assert.Equal(t, 1234567, int(c.Low))
}

func TestCandle_UpdateFromTrade2(t *testing.T) {
	p := uint64(math.MaxUint64 / 2)
	a := uint64(math.MaxUint64 / 10)
	ts1 := uint64(1542712337 * Second)
	tr1 := Trade{Timestamp: ts1, Price: p, Amount: a}
	ts2 := uint64(1542712237 * Second)
	tr2 := Trade{Timestamp: ts2, Price: p, Amount: a}
	c := NewCandleFromTimestamp(tr1.Timestamp)
	c.UpdateFromTrade(tr1)
	assert.Equal(t, int(a), int(c.Volume))
	assert.Equal(t, int(p), int(c.Average))
	c.UpdateFromTrade(tr2)
	assert.Equal(t, int(a+a), int(c.Volume))
	assert.Equal(t, int(p), int(c.Average))
}
