// Package applog is the companion's small log: one file, one rotation. It records what happened
// and why, never keys, signatures, link codes or character names.
package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Files returns the log's file and the one it rotates to, for whoever deletes them.
func Files(path string) []string { return []string{path, path + ".1"} }

// Logger appends lines to a file and moves it to path+".1" once it passes max bytes.
type Logger struct {
	path string
	max  int64
	now  func() time.Time
	mu   sync.Mutex
}

// New returns a logger; now is the clock the caller wants stamped on lines.
func New(path string, max int64, now func() time.Time) *Logger {
	return &Logger{path: path, max: max, now: now}
}

// Printf writes one line. Failure to log is not an error worth stopping for.
func (l *Logger) Printf(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(l.path), 0o700)
	if st, err := os.Stat(l.path); err == nil && st.Size() > l.max {
		_ = os.Rename(l.path, Files(l.path)[1])
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", l.now().UTC().Format("2006-01-02T15:04:05Z"), fmt.Sprintf(format, a...))
}
