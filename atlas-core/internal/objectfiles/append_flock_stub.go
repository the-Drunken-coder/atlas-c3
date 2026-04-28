//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package objectfiles

import "os"

// Windows and other platforms: advisory flock is not applied; single-process
// correctness still holds; document if multi-writer on shared volumes matters.
func flockAppendLock(f *os.File) error { return nil }

func flockAppendUnlock(f *os.File) error { return nil }
