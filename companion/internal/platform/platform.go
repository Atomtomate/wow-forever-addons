// Package platform holds the few things that differ by operating system, each behind one small
// function with a _windows.go and a _other.go version. Nothing here decides anything: package
// app wraps these functions in its Env so tests replace them.
package platform

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Executable returns the path of the running program.
func Executable() (string, error) { return os.Executable() }

// QueryVersion runs `exe version` and returns its first line, or "" when it cannot be run. It
// is how setup learns the version of the installed copy without a second source of truth.
func QueryVersion(exe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "version").Output()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line)
}
