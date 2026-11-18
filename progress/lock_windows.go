//go:build windows

package progress

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// lockRange adalah jumlah byte yang dikunci; cukup 1 byte untuk menandai lock.
const lockRange = 1

func openLockFile(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(
		name,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFile: %w", err)
	}
	return os.NewFile(uintptr(h), path), nil
}

func flockLock(f *os.File) error {
	o := &windows.Overlapped{}
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockRange,
		0,
		o,
	)
	if err != nil {
		return errLockHeld
	}
	return nil
}

func flockUnlock(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRange, 0, &windows.Overlapped{})
}
