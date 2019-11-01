// +build !windows

package main

import (
	"syscall"

	"github.com/pkg/errors"
)

func setMaxOpenFiles(limit uint64) error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Errorf("error getting rlimit: %v", err)
	}
	rLimit.Cur = limit

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Errorf("error setting rlimit: %v", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Errorf("error getting rlimit: %v", err)
	}
	return nil
}
