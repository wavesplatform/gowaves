// Useful routines used in several other packages.
package util

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
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
func TrackLongFunc() func() {
	s := debug.Stack()
	ch := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-ch:
				return
			case <-time.After(1 * time.Second):

				zap.S().Error("took long time", string(s))
			}
		}
	}()
	return func() {
		ch <- struct{}{}
	}
}
