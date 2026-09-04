package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast/v2"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const baseFee = state.CommitmentFeeInFeeUnits * state.FeeUnit

type config struct {
	height    uint32
	sk        crypto.SecretKey
	pk        crypto.PublicKey
	fee       uint64
	timestamp uint64
}

func (c *config) parse(args []string) error {
	if len(args) < 1 {
		return errors.New("invalid number of arguments")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress automatic output; we print usage ourselves below
	var (
		height     uint64
		privateKey string
		fee        uint64
		ts         string
	)
	fs.Uint64Var(&height, "height", 0, "Height of generation period start")
	fs.StringVar(&privateKey, "private-key", "", "Waves private key in Base58")
	fs.Uint64Var(&fee, "fee", baseFee, "Transaction fee (default: 0.1 Waves)")
	fs.StringVar(&ts, "timestamp", "",
		"Transaction timestamp (default: current time), can be absolute (e.g. '15:04') or relative "+
			"(e.g. '+1h30m' or '-45s')")
	if err := fs.Parse(args[1:]); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	if height == 0 {
		return errors.New("option -height is required and must be positive")
	}
	h, err := safecast.Convert[uint32](height)
	if err != nil {
		return err
	}
	c.height = h

	if len(privateKey) == 0 {
		return errors.New("option -private-key is required")
	}
	sk, err := crypto.NewSecretKeyFromBase58(privateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	c.sk = sk
	c.pk = crypto.GeneratePublicKey(sk)

	if fee < baseFee {
		return fmt.Errorf("option -fee must be equal or more than base fee %d", baseFee)
	}
	c.fee = fee

	timestamp, tsErr := parseTimestamp(ts)
	if tsErr != nil {
		return tsErr
	}
	c.timestamp = timestamp
	return nil
}

func parseTimestamp(ts string) (uint64, error) {
	now := time.Now()
	if len(ts) == 0 {
		ms, err := safecast.Convert[uint64](now.UnixMilli())
		if err != nil {
			return 0, fmt.Errorf("failed to get current timestamp: %w", err)
		}
		return ms, nil
	}
	if strings.HasPrefix(ts, "+") || strings.HasPrefix(ts, "-") {
		return parseTimestampShift(ts, now)
	}
	return parseTimeString(ts, now)
}

func parseTimestampShift(ts string, now time.Time) (uint64, error) {
	d, err := time.ParseDuration(ts)
	if err != nil {
		return 0, fmt.Errorf("invalid time shift: %w", err)
	}
	ms, convErr := safecast.Convert[uint64](now.Add(d).UnixMilli())
	if convErr != nil {
		return 0, fmt.Errorf("invalid timestamp from time shift %q: %w", ts, convErr)
	}
	return ms, nil
}

func parseTimeString(ts string, now time.Time) (uint64, error) {
	const (
		layoutHours   = "15"
		layoutMinutes = "15:04"
		layoutSeconds = "15:04:05"
	)
	var layout string
	switch strings.Count(ts, ":") {
	case 0:
		layout = layoutHours
	case 1:
		layout = layoutMinutes
	case 2:
		layout = layoutSeconds
	default:
		return 0, fmt.Errorf("invalid timestamp format %q", ts)
	}
	t, err := time.Parse(layout, ts)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp %q: %w", ts, err)
	}
	y, m, d := now.Date()
	combined := time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), 0, now.Location())
	ms, convErr := safecast.Convert[uint64](combined.UnixMilli())
	if convErr != nil {
		return 0, fmt.Errorf("failed to convert timestamp: %w", convErr)
	}
	return ms, nil
}
