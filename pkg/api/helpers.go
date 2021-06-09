package api

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"time"
)

func unixMillis(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}

func fromUnixMillis(timestampMillis int64) time.Time {
	sec := timestampMillis / 1_000
	nsec := (timestampMillis % 1_000) * 1_000_000
	return time.Unix(sec, nsec)
}

// tryParseJson receives reader and out params. out MUST be a pointer
func tryParseJson(r io.Reader, out interface{}) error {
	// TODO(nickeskov): check empty reader
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		return errors.Wrapf(err, "Failed to unmarshal %T as JSON into %T", r, out)
	}
	return nil
}

func trySendJson(w io.Writer, v interface{}) error {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal %T to JSON and write it to %T", v, w)
	}
	return nil
}
