package instance

import (
	"errors"
	"os"
	"syscall"
)

const (
	errSharingViolation = syscall.Errno(32)
)

// TryAcquire opens the lock file with a share mode of zero, so a second open fails with a
// sharing violation for as long as this process lives.
func TryAcquire(path string) (*Lock, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, errSharingViolation) {
			return nil, ErrHeld
		}
		return nil, err
	}
	return &Lock{f: os.NewFile(uintptr(h), path)}, nil
}

// Release gives the lock up.
func (l *Lock) Release() { _ = l.f.Close() }
