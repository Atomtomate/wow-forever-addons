// Package platform holds the few things that differ by operating system, each behind one small
// function with a _windows.go and a _other.go version. Nothing here decides anything: package
// app wraps these functions in its Env so tests replace them.
package platform

import "os"

// Executable returns the path of the running program.
func Executable() (string, error) { return os.Executable() }
