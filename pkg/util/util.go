// Useful routines used in several other packages.
package util

import (
	"os"
)

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
