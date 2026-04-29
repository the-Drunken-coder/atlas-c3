//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package objectfiles

import (
	"errors"
	"os"
)

// On unsupported platforms (Windows, plan9, etc.) advisory file locking is not
// available; both functions return errFlockUnsupported so callers fail fast
// rather than silently losing multi-writer correctness.
var errFlockUnsupported = errors.New("objectfiles: advisory file locking not supported on this platform; refusing append")

func flockAppendLock(f *os.File) error   { return errFlockUnsupported }
func flockAppendUnlock(f *os.File) error { return errFlockUnsupported }
