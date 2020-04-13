// Useful routines used in several other packages.
package common

import (
	"encoding/base64"
	"os"
	"os/user"
	"path"
	"runtime/debug"
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

func MinOf(vars ...uint64) uint64 {
	min := vars[0]
	for _, i := range vars {
		if min > i {
			min = i
		}
	}
	return min
}

func CleanTemporaryDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	zap.S().Infof("%s took %s", name, elapsed)
}

// call function like this
// defer TrackLongFunc()()
func TrackLongFunc(duration time.Duration, value ...string) func() {
	s := debug.Stack()
	ch := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				return
			case <-time.After(duration):
				zap.S().Error("took long time", value, string(s))
			}
		}
	}()
	return func() {
		close(ch)
	}
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
	switch strings.ToUpper(level) {
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
	zap.ReplaceGlobals(logger)
	return logger, logger.Sugar()
}

func ParseDuration(str string) (uint64, error) {
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

func ToBase64JSON(b []byte) []byte {
	s := base64.StdEncoding.EncodeToString(b)
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String())
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

func FromBase64JSON(value []byte, size int, name string) ([]byte, error) {
	v, err := FromBase64JSONUnsized(value, name)
	if err != nil {
		return nil, err
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, size)
	}
	return v[:size], nil
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

func Bts2Str(bts [][]byte) []string {
	out := []string{}
	for _, b := range bts {
		out = append(out, string(b))
	}
	return out
}
