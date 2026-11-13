// Package install puts the ForeverFriends addon, which the companion carries inside itself,
// into the game's AddOns folder (docs/DECISIONS.md D7). It never writes outside that folder
// and leaves a link alone, because a link is how a developer loads a checkout.
package install

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/atomicfile"
)

// AddonName is the folder, and the .toc, this package installs.
const AddonName = "ForeverFriends"

// Toc is what the companion needs of a .toc file.
type Toc struct {
	Version   string
	Interface string
	Files     []string // forward-slash, relative, in the .toc's order
}

// ParseToc reads a .toc: `## Key: value` lines and one file per other non-blank line. A file
// name that is empty, absolute, has a drive letter or climbs out with .. is refused.
func ParseToc(b []byte) (Toc, error) {
	var t Toc
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case strings.HasPrefix(line, "##"):
			k, v, ok := strings.Cut(strings.TrimSpace(line[2:]), ":")
			if !ok {
				continue
			}
			switch strings.TrimSpace(k) {
			case "Version":
				t.Version = strings.TrimSpace(v)
			case "Interface":
				t.Interface = strings.TrimSpace(v)
			}
		case strings.HasPrefix(line, "#"):
		default:
			name := path.Clean(strings.ReplaceAll(line, "\\", "/"))
			if name == "." || path.IsAbs(name) || strings.HasPrefix(name, "../") || name == ".." || strings.Contains(name, ":") {
				return Toc{}, fmt.Errorf("the addon's .toc lists a file name that is not allowed: %q", line)
			}
			t.Files = append(t.Files, name)
		}
	}
	if t.Version == "" || t.Interface == "" {
		return Toc{}, errors.New("the addon's .toc has no Version or Interface line")
	}
	return t, nil
}

// ReadToc reads ForeverFriends.toc from the embedded addon.
func ReadToc(src fs.FS) (Toc, error) {
	b, err := fs.ReadFile(src, AddonName+".toc")
	if err != nil {
		return Toc{}, err
	}
	return ParseToc(b)
}

// CompareVersions compares dotted numbers (0.1.0 against 0.2); text after the first
// non-numeric part of a segment is ignored. It returns -1, 0 or 1.
func CompareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = leadingInt(pa[i])
		}
		if i < len(pb) {
			y = leadingInt(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func leadingInt(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	v, _ := strconv.Atoi(s[:n])
	return v
}

// IsLink reports whether p is a symlink or a Windows junction, or lies behind one. Go 1.23
// reports a junction as ModeIrregular; the second test catches whatever the mode misses.
// foldCase is the platform's answer to whether paths ignore case.
func IsLink(p string, foldCase bool) bool {
	st, err := os.Lstat(p)
	if err != nil {
		return false
	}
	if st.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return true
	}
	realParent, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(p)
	want := filepath.Join(realParent, filepath.Base(p))
	if foldCase {
		return err == nil && !strings.EqualFold(real, want)
	}
	return err == nil && real != want
}

// What Install did.
const (
	Installed   = "installed"
	Updated     = "updated"
	Repaired    = "repaired"
	Current     = "already current"
	LeftLinked  = "left alone: it is a link, presumably to a checkout"
	LeftNewer   = "left alone: a newer version is installed"
	LeftForeign = "left alone: the folder is not ours and you chose not to replace it"
)

// ErrForeign is returned when the folder holds something else and nobody may be asked.
var ErrForeign = errors.New("the ForeverFriends folder exists and is not this addon")

// Report says what Install did and which version is in place afterwards.
type Report struct {
	Action  string
	Version string
}

// Install copies the addon in src (the embedded files, rooted at the addon folder) to
// addonsDir/ForeverFriends. confirm is asked before replacing a folder that is not ours; nil
// means no.
func Install(src fs.FS, addonsDir string, foldCase bool, confirm func(question string) bool) (Report, error) {
	toc, err := ReadToc(src)
	if err != nil {
		return Report{}, err
	}
	dest := filepath.Join(addonsDir, AddonName)
	rep := Report{Version: toc.Version}
	if IsLink(dest, foldCase) {
		rep.Action = LeftLinked
		return rep, nil
	}
	action := Installed
	entries, err := os.ReadDir(dest)
	switch {
	case errors.Is(err, os.ErrNotExist) || (err == nil && len(entries) == 0):
	case err != nil:
		return rep, err
	default:
		old, terr := os.ReadFile(filepath.Join(dest, AddonName+".toc"))
		if terr != nil && onlyOurs(entries, toc) {
			break // an interrupted first install: finish it
		}
		if terr != nil {
			if confirm == nil {
				return rep, ErrForeign
			}
			if !confirm("The folder " + dest + " holds something that is not this addon. Replace it?") {
				rep.Action = LeftForeign
				return rep, nil
			}
			action = Updated
			break
		}
		oldToc, _ := ParseToc(old)
		rep.Version = oldToc.Version
		switch c := CompareVersions(oldToc.Version, toc.Version); {
		case c > 0:
			rep.Action = LeftNewer
			return rep, nil
		case c < 0:
			action = Updated
		default:
			action = Current
		}
	}
	changed, err := copyFiles(src, dest, toc)
	if err != nil {
		return rep, err
	}
	rep.Version = toc.Version
	switch {
	case action == Current && changed:
		rep.Action = Repaired
	default:
		rep.Action = action
	}
	return rep, nil
}

// onlyOurs reports whether every entry of a folder without a .toc is a file the embedded .toc
// lists, or the temporary file of one: what an interrupted first install leaves behind.
func onlyOurs(entries []os.DirEntry, toc Toc) bool {
	listed := map[string]bool{}
	for _, f := range toc.Files {
		listed[path.Base(f)], listed[path.Base(f)+".tmp"] = true, true
	}
	for _, e := range entries {
		if e.IsDir() || !listed[e.Name()] {
			return false
		}
	}
	return true
}

// copyFiles writes every listed file whose bytes differ, the .toc last so that a half-finished
// copy still shows the old version and is repaired by the next run, then removes the .lua,
// .toc and .xml files the new version no longer lists.
func copyFiles(src fs.FS, dest string, toc Toc) (bool, error) {
	changed := false
	names := append([]string{}, toc.Files...)
	names = append(names, AddonName+".toc")
	listed := map[string]bool{}
	for _, name := range names {
		listed[name] = true
		want, err := fs.ReadFile(src, name)
		if err != nil {
			return changed, fmt.Errorf("the addon's .toc lists %s, which this build does not carry: %w", name, err)
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if have, err := os.ReadFile(target); err == nil && bytes.Equal(have, want) {
			continue
		}
		if err := atomicfile.Write(target, want, 0o755, 0o644); err != nil {
			return changed, err
		}
		changed = true
	}
	entries, _ := os.ReadDir(dest)
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() || listed[e.Name()] || (ext != ".lua" && ext != ".toc" && ext != ".xml") {
			continue
		}
		if err := os.Remove(filepath.Join(dest, e.Name())); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

// Remove deletes the named addon folders inside addonsDir, except one that is a link. It
// returns a line for each folder it looked at.
func Remove(addonsDir string, foldCase bool, names ...string) []string {
	var out []string
	for _, n := range names {
		if n == "" || n == "." || n == ".." || n != filepath.Base(n) {
			continue
		}
		p := filepath.Join(addonsDir, n)
		switch _, err := os.Lstat(p); {
		case err != nil:
		case IsLink(p, foldCase):
			out = append(out, n+": left alone, it is a link")
		default:
			if err := os.RemoveAll(p); err != nil {
				out = append(out, n+": could not be removed: "+err.Error())
			} else {
				out = append(out, n+": removed")
			}
		}
	}
	return out
}
