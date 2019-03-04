// Useful routines used in several other packages.
package util

import (
	"os"

	"github.com/pkg/errors"
)

// Check for overflow.
func AddInt64(a, b int64) (int64, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("integer overflow")
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
