// Useful routines used in several other packages.
package common

import (
	"encoding/base64"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Safe sum for int64.
func AddInt64(a, b int64) (int64, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("64-bit signed integer overflow")
}

// Safe sum for uint64.
func AddUint64(a, b uint64) (uint64, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("64-bit unsigned integer overflow")
}

func CleanTemporaryDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

// duplicate (copy) bytes
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
	return path.Join(u.HomeDir, ".gowaves"), nil
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

type seconds = uint64

func ParseDuration(str string) (seconds, error) {
	if str == "" {
		return 0, errors.New("empty string")
	}
	total := uint64(0)
	cur := uint64(0)
	expectNum := true
	for _, v := range str {
		switch v {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			expectNum = false
			cur = cur*10 + uint64(v-'0')
		case 'd', 'h', 'm':
			if expectNum {
				return 0, errors.Errorf("invalid char %c, expected 0 <= value <= 9", v)
			}
			expectNum = true
			switch v {
			case 'd':
				total += cur * 86400
			case 'h':
				total += cur * 3600
			case 'm':
				total += cur * 60
			}
			cur = 0
		default:
			return 0, errors.Errorf("invalid char '%c'", v)
		}
	}
	if !expectNum {
		return 0, errors.Errorf("invalid format")
	}
	return total, nil
}

func FromBase64JSONUnsized(value []byte, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from Base64 string", name)
	}
	return v, nil
}

func ToBase58JSON(b []byte) []byte {
	s := base58.Encode(b)
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String())
}

func FromBase58JSONUnsized(value []byte, name string) ([]byte, error) {
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
	v, err := FromBase58JSONUnsized(value, name)
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

// no way when expected can be higher than current, but if somehow its happened...
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

func TimestampMillisToTime(ts uint64) time.Time {
	ts64 := int64(ts)
	s := ts64 / 1000
	ns := ts64 % 1000 * 1000000
	return time.Unix(s, ns)
}
