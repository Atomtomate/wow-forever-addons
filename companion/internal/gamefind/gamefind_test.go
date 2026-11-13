package gamefind

import (
	"path/filepath"
	"testing"
)

func dirs(paths ...string) func(string) bool {
	return func(p string) bool {
		for _, d := range paths {
			if filepath.Clean(d) == filepath.Clean(p) {
				return true
			}
		}
		return false
	}
}

func TestNormaliseTypedFolder(t *testing.T) {
	wow := filepath.Join("games", "World of Warcraft")
	game := filepath.Join(wow, "_forever_")
	isDir := dirs(filepath.Join(game, "Interface"))
	for _, in := range []string{game, `"` + game + `"`, "  " + wow + " ", "'" + wow + string(filepath.Separator) + "'"} {
		if got, ok := Normalize(in, isDir); !ok || got != game {
			t.Errorf("%q: %q %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "nowhere", filepath.Join("games")} {
		if _, ok := Normalize(in, isDir); ok {
			t.Errorf("%q accepted", in)
		}
	}
}

func TestFirstCandidate(t *testing.T) {
	isDir := dirs(filepath.Join("b", "WTF"))
	if got, ok := First([]string{"a", "b", "c"}, isDir); !ok || got != "b" {
		t.Fatal(got, ok)
	}
	if _, ok := First([]string{"a"}, isDir); ok {
		t.Fatal("found nothing")
	}
}

func TestRunningMatchesWholeSegments(t *testing.T) {
	game := filepath.Join("games", "_forever_")
	in := filepath.Join(game, "Wow.exe")
	if !Running([]string{filepath.Join("other", "x"), in}, game, false) {
		t.Fatal("not found")
	}
	if Running([]string{filepath.Join("games", "_forever_x", "Wow.exe"), game}, game, false) {
		t.Fatal("prefix of a name matched")
	}
	if Running([]string{filepath.Join("games", "_FOREVER_", "Wow.exe")}, game, false) {
		t.Fatal("case folded without being asked")
	}
	if !Running([]string{filepath.Join("games", "_FOREVER_", "Wow.exe")}, game, true) {
		t.Fatal("case not folded")
	}
	if Running(nil, game, true) {
		t.Fatal("empty list")
	}
}
