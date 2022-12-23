package common

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AddInt64 makes safe sum for int64.
func AddInt64(a, b int64) (int64, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("64-bit signed integer overflow")
}

// AddInt makes safe sum for int.
func AddInt(a, b int) (int, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("signed integer overflow")
}

// AddUint64 makes safe sum for uint64.
func AddUint64(a, b uint64) (uint64, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("64-bit unsigned integer overflow")
}

// Dup duplicate (copy) bytes.
func Dup(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func GetStatePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".gowaves"), nil
}

func SetupLogger(level string) (*zap.Logger, *zap.SugaredLogger) {
	al := zap.NewAtomicLevel()
	var opts []zap.Option
	switch strings.ToUpper(level) {
	case "DEV":
		al.SetLevel(zap.DebugLevel)
		opts = append(opts, zap.AddCaller())
	case "DEBUG":
		al.SetLevel(zap.DebugLevel)
	case "INFO":
		al.SetLevel(zap.InfoLevel)
	case "ERROR":
		al.SetLevel(zap.ErrorLevel)
	case "WARN":
		al.SetLevel(zap.WarnLevel)
	case "FATAL":
		al.SetLevel(zap.FatalLevel)
	default:
		al.SetLevel(zap.InfoLevel)
	}
	ec := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.Lock(os.Stdout), al)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger.WithOptions(opts...))
	return logger, logger.Sugar()
}

func ToBase58JSON(b []byte) []byte {
	s := base58.Encode(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func ToBase64JSON(b []byte) []byte {
	s := base64.StdEncoding.EncodeToString(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func FromBase58JSONUnchecked(value []byte, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := base58.Decode(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from Base58 string", name)
	}
	return v, nil
}

func FromBase58JSON(value []byte, size int, name string) ([]byte, error) {
	v, err := FromBase58JSONUnchecked(value, name)
	if err != nil {
		return nil, err
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, size)
	}
	return v[:size], nil
}

func ToHexJSON(b []byte) []byte {
	return []byte(fmt.Sprintf("\"0x%x\"", b))
}

func FromHexJSONUnchecked(value []byte, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from hex string", name)
	}
	return v, nil
}

func FromHexJSON(value []byte, size int, name string) ([]byte, error) {
	v, err := FromHexJSONUnchecked(value, name)
	if err != nil {
		return nil, err
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, size)
	}
	return v[:size], nil
}

type tm interface {
	Now() time.Time
}

// EnsureTimeout ensures that no way when expected can be higher than current, but if somehow its happened...
func EnsureTimeout(tm tm, expected uint64) {
	for {
		current := uint64(tm.Now().UnixNano() / 1000000)
		if expected > current {
			<-time.After(5 * time.Millisecond)
			continue
		}
		break
	}
}

func UnixMillisToTime(ts int64) time.Time {
	sec := ts / 1_000
	ns := (ts % 1_000) * 1_000_000
	return time.Unix(sec, ns)
}

func UnixMillisFromTime(t time.Time) int64 {
	return t.UnixMilli()
}

// ReplaceInvalidUtf8Chars replaces invalid utf8 characters with '?' to reproduce JVM behaviour.
func ReplaceInvalidUtf8Chars(s string) string {
	var b strings.Builder

	// Ranging over a string in Go produces runes. When the range keyword
	// encounters an invalid UTF-8 encoding, it returns REPLACEMENT CHARACTER.
	for _, v := range s {
		b.WriteRune(v)
	}
	return b.String()
}
