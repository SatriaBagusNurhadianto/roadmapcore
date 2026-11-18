//go:build !windows

package progress

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	lockExclusive = unix.LOCK_EX | unix.LOCK_NB
	lockRelease   = unix.LOCK_UN
)

func openLockFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
}

func flockLock(f *os.File) error {
	if err := unix.Flock(int(f.Fd()), lockExclusive); err != nil {
		return errLockHeld
	}
	return nil
}

func flockUnlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), lockRelease)
}
