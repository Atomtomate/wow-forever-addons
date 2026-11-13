// Package dataaddon writes ForeverFriendsData, the folder through which the game learns the
// hub's member list (docs/CONTRACT.md §2). Everything in it comes from people typing on the hub
// and is executed by the game, so no string is written except through Quote.
package dataaddon

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// FolderName is the addon folder this package owns inside AddOns.
const FolderName = "ForeverFriendsData"

var interfaceValue = regexp.MustCompile(`^[0-9]{4,6}(,[0-9]{4,6})*$`)

// Quote returns s as a double-quoted Lua literal. A backslash and a double quote are escaped
// and every byte below 32, and byte 127, becomes \ddd with exactly three digits (Lua reads up to
// three, so \1 followed by a digit would be another byte); bytes from 128 up pass through.
func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' || c == '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 32 || c == 127:
			b.WriteByte('\\')
			b.WriteString(string([]byte{'0' + c/100, '0' + c/10%10, '0' + c%10}))
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Render returns Data.lua for a member list. Numbers come from strconv, never from text.
func Render(r wire.Roster, written int64) []byte {
	var b bytes.Buffer
	b.WriteString("ForeverFriendsData = {\n    version = 1,\n    written = " + strconv.FormatInt(written, 10) + ",\n")
	if len(r.Members) == 0 {
		b.WriteString("    members = {},\n}\n")
		return b.Bytes()
	}
	b.WriteString("    members = {\n")
	for _, m := range r.Members {
		b.WriteString("        { name = " + Quote(m.Name) + ", characters = {\n")
		for _, c := range m.Characters {
			b.WriteString("            { name = " + Quote(c.Name) + ", class = " + Quote(c.Class))
			if c.Level > 0 {
				b.WriteString(", level = " + strconv.Itoa(c.Level))
			}
			if c.Main {
				b.WriteString(", main = true")
			}
			b.WriteString(" },\n")
		}
		b.WriteString("        } },\n")
	}
	b.WriteString("    },\n}\n")
	return b.Bytes()
}

// Toc returns ForeverFriendsData.toc with the Interface value of the ForeverFriends.toc the
// companion carries, so a client patch never marks one of the two addons out of date alone.
func Toc(iface string) ([]byte, error) {
	if !interfaceValue.MatchString(iface) {
		return nil, errors.New("the addon's Interface value is not a number list: " + iface)
	}
	return []byte("## Interface: " + iface + "\n## Title: Forever Friends (member list)\n" +
		"## Notes: Written by the Forever Friends companion. Changes here are overwritten.\nData.lua\n"), nil
}

var writtenLine = regexp.MustCompile(`(?m)^    written = ([0-9]+),$`)

func writeIfDifferent(path string, data []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	return true, config.WriteFileAtomic(path, data, 0o644)
}

// Write replaces the member list in dir (the ForeverFriendsData folder) and reports whether
// anything changed. A file that differs only in `written` is left as it is, so the game's
// folder is quiet while the list is.
func Write(dir string, r wire.Roster, iface string, now time.Time) (bool, error) {
	toc, err := Toc(iface)
	if err != nil {
		return false, err
	}
	path := filepath.Join(dir, "Data.lua")
	changed := true
	if old, err := os.ReadFile(path); err == nil {
		if m := writtenLine.FindSubmatch(old); m != nil {
			if n, err := strconv.ParseInt(string(m[1]), 10, 64); err == nil {
				changed = !bytes.Equal(old, Render(r, n))
			}
		}
	}
	// The data file is written before the toc: a folder without a toc is not loaded, so the
	// game never sees a toc pointing at a half-finished list.
	if changed {
		if err := config.WriteFileAtomic(path, Render(r, now.Unix()), 0o644); err != nil {
			return false, err
		}
	}
	tocChanged, err := writeIfDifferent(filepath.Join(dir, FolderName+".toc"), toc)
	return changed || tocChanged, err
}

// EnsureEmpty creates the folder at install with `members = {}` and `written = 0`, so that one
// restart of the game is enough for both addon folders. An existing Data.lua is kept: running
// setup again must not wipe a member list the hub has already sent.
func EnsureEmpty(dir, iface string) error {
	toc, err := Toc(iface)
	if err != nil {
		return err
	}
	if _, err := writeIfDifferent(filepath.Join(dir, FolderName+".toc"), toc); err != nil {
		return err
	}
	path := filepath.Join(dir, "Data.lua")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return config.WriteFileAtomic(path, Render(wire.Roster{}, 0), 0o644)
}
