//go:build windows
// +build windows

package fdlimit

import (
	"github.com/pkg/errors"
)

// hardFDLimit is the number of file descriptors allowed at max by the kernel.
const hardFDLimit = 16 * 1024

func RaiseMaxFDs(max uint64) (uint64, error) {
	// This function is No-op:
	//  * Linux/Darwin counterparts need to manually increase per process limits
	//  * On Windows Go uses the CreateFile API, which is limited to 16K files, non
	//    changeable from within a running process
	if max > hardFDLimit {
		return 0, errors.Errorf("FD limit (%d) reached", hardFDLimit)
	}
	return max, nil
}

func CurrentFDs() (uint64, error) {
	return hardFDLimit, nil
}

func MaxFDs() (uint64, error) {
	return hardFDLimit, nil
}
