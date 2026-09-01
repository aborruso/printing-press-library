// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

//go:build !windows

package cliutil

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockExclusiveNonBlocking(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func isLockBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EACCES)
}

func lockExclusiveBlocking(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func unlockFile(f *os.File) {
	_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
