package client

import "time"

// timestamp in milliseconds
type Timestamp uint64

func NewTimestampFromTime(t time.Time) Timestamp {
	return NewTimestampFromUnixNano(t.UnixNano())
}

func NewTimestampFromUnixNano(nano int64) Timestamp {
	return Timestamp(nano / 1000000)
}
