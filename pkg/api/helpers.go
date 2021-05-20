package api

import "time"

func unixMillis(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}
