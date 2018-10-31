package client

import "time"

func NewTimestampFromTime(t time.Time) uint64 {
	return NewTimestampFromUnixNano(t.UnixNano())
}

func NewTimestampFromUnixNano(nano int64) uint64 {
	return uint64(nano / 1000000)
}
