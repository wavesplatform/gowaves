package main

import (
	"flag"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidArgs(t *testing.T) {
	cfg := config{}
	err := cfg.parse([]string{"commit", "-height", "100", "-private-key", "25Um7fKYkySZnweUEVAn9RLtxN5xHRd7iqpqYSMNQEeT"})
	require.NoError(t, err)
	assert.Equal(t, uint32(100), cfg.height)
	assert.Equal(t, uint64(baseFee), cfg.fee)
	assert.Greater(t, cfg.timestamp, uint64(0))
	assert.Equal(t, "25Um7fKYkySZnweUEVAn9RLtxN5xHRd7iqpqYSMNQEeT", cfg.sk.String())
}

func TestParse_MissingHeight(t *testing.T) {
	cfg := config{}
	err := cfg.parse([]string{"commit", "-private-key", "25Um7fKYkySZnweUEVAn9RLtxN5xHRd7iqpqYSMNQEeT"})
	assert.EqualError(t, err, "option -height is required and must be positive")
}

func TestParse_InvalidPrivateKey(t *testing.T) {
	cfg := config{}
	err := cfg.parse([]string{"commit", "-height", "100", "-private-key", "25Um7fKYkySZnweUEVAn9RLtxN5xHRd7iqpqYSM"})
	assert.EqualError(t, err, "failed to parse private key: incorrect SecretKey length 28, expected 32")
}

func TestParse_MissingPrivateKey(t *testing.T) {
	cfg := config{}
	err := cfg.parse([]string{"commit", "-height", "100"})
	assert.EqualError(t, err, "option -private-key is required")
}

func TestParse_HelpFlag(t *testing.T) {
	cfg := config{}
	err := cfg.parse([]string{"commit", "-h"})
	assert.EqualError(t, err, flag.ErrHelp.Error())
}

func TestParse_CalledTwiceNoPanic(t *testing.T) {
	args := []string{"commit", "-height", "100", "-private-key", "25Um7fKYkySZnweUEVAn9RLtxN5xHRd7iqpqYSMNQEeT"}
	cfg1 := config{}
	require.NoError(t, cfg1.parse(args))
	cfg2 := config{}
	require.NoError(t, cfg2.parse(args))
}

func TestParseTimestamp_EmptyReturnsCurrentTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := time.Now().UnixMilli()
		ts, err := parseTimestamp("")
		require.NoError(t, err)
		assert.Equal(t, tm, int64(ts))
	})
}

func TestParseTimestamp_PositiveShift(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := time.Now().Add(time.Hour).UnixMilli()
		ts, err := parseTimestamp("+1h")
		require.NoError(t, err)
		assert.Equal(t, tm, int64(ts))
	})
}

func TestParseTimestamp_NegativeShift(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := time.Now().Add(-30 * time.Minute).UnixMilli()
		ts, err := parseTimestamp("-30m")
		require.NoError(t, err)
		assert.Equal(t, tm, int64(ts))
	})
}

func TestParseTimestamp_TimeStringHours(t *testing.T) {
	ts, err := parseTimestamp("14")
	require.NoError(t, err)
	assert.Greater(t, ts, uint64(0))
}

func TestParseTimestamp_TimeStringMinutes(t *testing.T) {
	ts, err := parseTimestamp("14:30")
	require.NoError(t, err)
	assert.Greater(t, ts, uint64(0))
}

func TestParseTimestamp_TimeStringSeconds(t *testing.T) {
	ts, err := parseTimestamp("14:30:45")
	require.NoError(t, err)
	assert.Greater(t, ts, uint64(0))
}

func TestParseTimestamp_InvalidShift(t *testing.T) {
	_, err := parseTimestamp("+notaduration")
	assert.EqualError(t, err, "invalid time shift: time: invalid duration \"+notaduration\"")
}

func TestParseTimestamp_InvalidNegativeShift(t *testing.T) {
	_, err := parseTimestamp("--1h")
	assert.EqualError(t, err, "invalid time shift: time: invalid duration \"--1h\"")
}

func TestParseTimestamp_InvalidPositiveShift(t *testing.T) {
	_, err := parseTimestamp("++5m")
	assert.EqualError(t, err, "invalid time shift: time: invalid duration \"++5m\"")
}

func TestParseTimestamp_InvalidTimeString(t *testing.T) {
	_, err := parseTimestamp("not-a-time")
	assert.EqualError(t, err,
		"invalid timestamp \"not-a-time\": parsing time \"not-a-time\" as \"15\": cannot parse \"not-a-time\" as \"15\"")
}

func TestParseTimestamp_TooManyColons(t *testing.T) {
	_, err := parseTimestamp("14:30:45:00")
	assert.EqualError(t, err, "invalid timestamp format \"14:30:45:00\"")
}

func TestParseTimestampShift_Positive(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	ms, err := parseTimestampShift("+1h", now)
	require.NoError(t, err)
	assert.Equal(t, uint64(now.Add(time.Hour).UnixMilli()), ms)
}

func TestParseTimestampShift_Negative(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	ms, err := parseTimestampShift("-30m", now)
	require.NoError(t, err)
	assert.Equal(t, uint64(now.Add(-30*time.Minute).UnixMilli()), ms)
}

func TestParseTimestampShift_CompoundDuration(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	ms, err := parseTimestampShift("+1h30m", now)
	require.NoError(t, err)
	assert.Equal(t, uint64(now.Add(90*time.Minute).UnixMilli()), ms)
}

func TestParseTimestampShift_Invalid(t *testing.T) {
	now := time.Now()
	_, err := parseTimestampShift("+invalid", now)
	assert.EqualError(t, err, "invalid time shift: time: invalid duration \"+invalid\"")
}

func TestParseTimeString_HoursOnly(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	ms, err := parseTimeString("14", now)
	require.NoError(t, err)
	expected := time.Date(2025, 6, 1, 14, 0, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, uint64(expected), ms)
}

func TestParseTimeString_HoursAndMinutes(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	ms, err := parseTimeString("14:30", now)
	require.NoError(t, err)
	expected := time.Date(2025, 6, 1, 14, 30, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, uint64(expected), ms)
}

func TestParseTimeString_HoursMinutesSeconds(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	ms, err := parseTimeString("14:30:45", now)
	require.NoError(t, err)
	expected := time.Date(2025, 6, 1, 14, 30, 45, 0, time.UTC).UnixMilli()
	assert.Equal(t, uint64(expected), ms)
}

func TestParseTimeString_TooManyColons(t *testing.T) {
	now := time.Now()
	_, err := parseTimeString("14:30:45:00", now)
	assert.EqualError(t, err, "invalid timestamp format \"14:30:45:00\"")
}

func TestParseTimeString_InvalidHour(t *testing.T) {
	now := time.Now()
	_, err := parseTimeString("99", now)
	assert.EqualError(t, err, "invalid timestamp \"99\": parsing time \"99\": hour out of range")
}

func TestParseTimeString_PreservesDate(t *testing.T) {
	now := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	ms, err := parseTimeString("10:00", now)
	require.NoError(t, err)
	expected := time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, uint64(expected), ms)
}
