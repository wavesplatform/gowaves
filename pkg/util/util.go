// Useful routines used in several other packages.
package util

import (
	"os"
	"os/user"
	"path"
	"runtime/debug"
	"strings"
	"time"

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
