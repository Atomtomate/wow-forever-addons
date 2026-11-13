// Package gamefind decides which folder is the game and whether the game is running. It looks
// at nothing itself: the caller passes in what exists and what runs.
package gamefind

import (
	"path/filepath"
	"strings"
)

// FlavourDir is the folder the game keeps inside "World of Warcraft".
const FlavourDir = "_forever_"

// Valid reports whether dir looks like a game folder: it holds Interface or WTF.
func Valid(dir string, isDir func(string) bool) bool {
	return isDir(filepath.Join(dir, "Interface")) || isDir(filepath.Join(dir, "WTF"))
}

// Normalize turns what a member typed or dragged in (maybe quoted, maybe the parent
// "World of Warcraft" folder) into the game folder, or reports that it is none.
func Normalize(input string, isDir func(string) bool) (string, bool) {
	in := strings.TrimSpace(input)
	in = strings.Trim(in, "\"'")
	in = strings.TrimSpace(in)
	if in == "" {
		return "", false
	}
	in = filepath.Clean(in)
	if Valid(in, isDir) {
		return in, true
	}
	if sub := filepath.Join(in, FlavourDir); Valid(sub, isDir) {
		return sub, true
	}
	return "", false
}

// First returns the first candidate that is a game folder.
func First(candidates []string, isDir func(string) bool) (string, bool) {
	for _, c := range candidates {
		if Valid(c, isDir) {
			return c, true
		}
	}
	return "", false
}

// Running reports whether any process image lies inside the game folder. The match is on whole
// path segments, so ".../_forever_x/Wow.exe" is not inside ".../_forever_". foldCase is for
// Windows, whose paths ignore case.
func Running(images []string, gameDir string, foldCase bool) bool {
	norm := func(p string) string {
		p = filepath.Clean(p)
		if foldCase {
			p = strings.ToLower(p)
		}
		return p
	}
	prefix := norm(gameDir) + string(filepath.Separator)
	for _, img := range images {
		if strings.HasPrefix(norm(img), prefix) {
			return true
		}
	}
	return false
}
