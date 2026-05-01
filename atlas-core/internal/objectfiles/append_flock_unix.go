//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package objectfiles

import (
	"os"

	"golang.org/x/sys/unix"
)

func flockAppendLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func flockAppendUnlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
