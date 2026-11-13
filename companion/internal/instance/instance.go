// Package instance makes sure one background companion runs per user, and lets a setup run ask
// it to stop. It offers only calls that return at once: waiting is the caller's, on its clock.
package instance

import (
	"errors"
	"os"
)

// ErrHeld means another process holds the lock.
var ErrHeld = errors.New("the companion is already running")

// Lock is held for the life of the process; the operating system releases it if the process
// dies, so there is no stale lock to clean up.
type Lock struct{ f *os.File }

// Release gives the lock up.
func (l *Lock) Release() { _ = l.f.Close() }

// RequestStop asks a running companion to exit at its next tick.
func RequestStop(stopFile string) error { return os.WriteFile(stopFile, []byte("stop\n"), 0o600) }

// StopRequested reports whether a stop request is waiting.
func StopRequested(stopFile string) bool {
	_, err := os.Stat(stopFile)
	return err == nil
}

// ClearStop removes the request.
func ClearStop(stopFile string) { _ = os.Remove(stopFile) }
