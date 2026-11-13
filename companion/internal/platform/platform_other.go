//go:build !windows

package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// ConfigDir is the folder for the key, the link and the log.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "forever-friends"), nil
}

// ProgramDir is empty: off Windows the program runs where the member put it and is not copied.
func ProgramDir() string { return "" }

// CaseFoldPaths reports whether file paths ignore case.
func CaseFoldPaths() bool { return false }

// GameCandidates has no guesses off Windows; the member names the folder.
func GameCandidates() []string { return nil }

// ListImages returns the executable path of every process this user can see. Only Linux
// exposes it (/proc/<pid>/exe); on macOS it returns nothing, so the game is never reported as
// running there.
func ListImages() ([]string, error) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}
	dirs, err := filepath.Glob("/proc/[0-9]*/exe")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, d := range dirs {
		if p, err := os.Readlink(d); err == nil {
			out = append(out, strings.TrimSuffix(p, " (deleted)"))
		}
	}
	return out, nil
}

// OpenURL opens an address in the default browser; the caller builds it from a checked origin.
func OpenURL(url string) error {
	bin := "xdg-open"
	if runtime.GOOS == "darwin" {
		bin = "open"
	}
	return exec.Command(bin, url).Start()
}

// SetAutostart is not offered off Windows.
func SetAutostart(string, ...string) error { return errors.New("autostart is only built for Windows") }

// RemoveAutostart has nothing to remove off Windows.
func RemoveAutostart() error { return nil }

// SpawnDetached starts exe in its own session, so it outlives the terminal that started it.
func SpawnDetached(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// PrepareConsole does nothing here.
func PrepareConsole() {}
