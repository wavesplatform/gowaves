package api

import "time"

func unixMillis(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}

func fromUnixMillis(timestampMillis int64) time.Time {
	sec := timestampMillis / 1_000
	nsec := (timestampMillis % 1_000) * 1_000_000
	return time.Unix(sec, nsec)
}
