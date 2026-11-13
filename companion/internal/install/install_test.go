package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

func addon(version string, files map[string]string) fstest.MapFS {
	toc := "## Interface: 16001\n## Version: " + version + "\n## OptionalDeps: ForeverFriendsData\n# comment\n"
	m := fstest.MapFS{}
	for n, c := range files {
		toc += n + "\n"
		m[n] = &fstest.MapFile{Data: []byte(c)}
	}
	m[AddonName+".toc"] = &fstest.MapFile{Data: []byte(toc)}
	return m
}

func read(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTocParseAndRefusals(t *testing.T) {
	toc, err := ParseToc([]byte("## Interface: 16001\r\n## Version: 0.1.0\r\n\r\nCore.lua\r\nsub\\Pack.lua\r\n"))
	if err != nil || toc.Version != "0.1.0" || toc.Interface != "16001" || len(toc.Files) != 2 || toc.Files[1] != "sub/Pack.lua" {
		t.Fatalf("%+v %v", toc, err)
	}
	for _, bad := range []string{"../x.lua", "/abs.lua", "C:\\x.lua", "a/../../x.lua"} {
		if _, err := ParseToc([]byte("## Interface: 1\n## Version: 1\n" + bad + "\n")); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := ParseToc([]byte("Core.lua\n")); err == nil {
		t.Error("toc without version accepted")
	}
}

func TestVersionCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"0.1.0", "0.1.0", 0}, {"0.1.0", "0.2", -1}, {"0.10.0", "0.9.9", 1}, {"1.0", "1.0.0", 0}, {"0.1.0-dev", "0.1.0", 0}, {"2", "10", -1}} {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("%s %s: %d", c.a, c.b, got)
		}
	}
}

func TestFreshInstall(t *testing.T) {
	dir := t.TempDir()
	rep, err := Install(addon("0.1.0", map[string]string{"Core.lua": "a", "Pack.lua": "b"}), dir, false, nil)
	if err != nil || rep.Action != Installed || rep.Version != "0.1.0" {
		t.Fatal(rep, err)
	}
	if read(t, filepath.Join(dir, "ForeverFriends/Core.lua")) != "a" || !strings.Contains(read(t, filepath.Join(dir, "ForeverFriends/ForeverFriends.toc")), "Core.lua") {
		t.Fatal("files missing")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "ForeverFriends"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatal("temp left", e.Name())
		}
	}
}

// The .toc is written last: an update that fails part way still shows the old version, so the
// next run repairs it instead of believing it is current.
func TestTocWrittenLastSoAFailedUpdateKeepsOldVersion(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(addon("0.1.0", map[string]string{"Core.lua": "old"}), dir, false, nil); err != nil {
		t.Fatal(err)
	}
	next := addon("0.2.0", map[string]string{"Core.lua": "new"})
	next["ForeverFriends.toc"] = &fstest.MapFile{Data: append(next["ForeverFriends.toc"].Data, []byte("Missing.lua\n")...)}
	if _, err := Install(next, dir, false, nil); err == nil {
		t.Fatal("update with a missing file succeeded")
	}
	if got := read(t, filepath.Join(dir, "ForeverFriends/ForeverFriends.toc")); !strings.Contains(got, "Version: 0.1.0") {
		t.Fatalf("toc already replaced:\n%s", got)
	}
}

func TestInterruptedFirstInstallIsFinishedNotTreatedAsForeign(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "ForeverFriends")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "Core.lua"), []byte("half"), 0o644)
	_ = os.WriteFile(filepath.Join(d, "Pack.lua.tmp"), []byte("x"), 0o644)
	rep, err := Install(addon("0.1.0", map[string]string{"Core.lua": "a", "Pack.lua": "b"}), dir, false, nil)
	if err != nil || rep.Action != Installed || read(t, filepath.Join(d, "Core.lua")) != "a" {
		t.Fatal(rep, err)
	}
}

func TestOlderReplacedStaleRemovedOthersKept(t *testing.T) {
	dir := t.TempDir()
	_, _ = Install(addon("0.1.0", map[string]string{"Core.lua": "a", "Old.lua": "x"}), dir, false, nil)
	_ = os.WriteFile(filepath.Join(dir, "ForeverFriends/notes.txt"), []byte("mine"), 0o644)
	rep, err := Install(addon("0.2.0", map[string]string{"Core.lua": "a2"}), dir, false, nil)
	if err != nil || rep.Action != Updated {
		t.Fatal(rep, err)
	}
	if read(t, filepath.Join(dir, "ForeverFriends/Core.lua")) != "a2" {
		t.Fatal("not replaced")
	}
	if _, err := os.Stat(filepath.Join(dir, "ForeverFriends/Old.lua")); err == nil {
		t.Fatal("stale file kept")
	}
	if read(t, filepath.Join(dir, "ForeverFriends/notes.txt")) != "mine" {
		t.Fatal("foreign file removed")
	}
}

func TestSameRepairedThenCurrent(t *testing.T) {
	dir := t.TempDir()
	src := addon("0.1.0", map[string]string{"Core.lua": "a"})
	_, _ = Install(src, dir, false, nil)
	if rep, _ := Install(src, dir, false, nil); rep.Action != Current {
		t.Fatal(rep)
	}
	_ = os.WriteFile(filepath.Join(dir, "ForeverFriends/Core.lua"), []byte("edited"), 0o644)
	if rep, _ := Install(src, dir, false, nil); rep.Action != Repaired || read(t, filepath.Join(dir, "ForeverFriends/Core.lua")) != "a" {
		t.Fatal(rep)
	}
}

func TestNewerLeftAlone(t *testing.T) {
	dir := t.TempDir()
	_, _ = Install(addon("0.3.0", map[string]string{"Core.lua": "new"}), dir, false, nil)
	rep, err := Install(addon("0.1.0", map[string]string{"Core.lua": "old"}), dir, false, nil)
	if err != nil || rep.Action != LeftNewer || read(t, filepath.Join(dir, "ForeverFriends/Core.lua")) != "new" {
		t.Fatal(rep, err)
	}
}

func TestLinkLeftAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows; the junction case is a PC check")
	}
	dir, checkout := t.TempDir(), t.TempDir()
	_ = os.WriteFile(filepath.Join(checkout, "Core.lua"), []byte("dev"), 0o644)
	if err := os.Symlink(checkout, filepath.Join(dir, "ForeverFriends")); err != nil {
		t.Fatal(err)
	}
	rep, err := Install(addon("0.1.0", map[string]string{"Core.lua": "a"}), dir, false, nil)
	if err != nil || rep.Action != LeftLinked || read(t, filepath.Join(checkout, "Core.lua")) != "dev" {
		t.Fatal(rep, err)
	}
	if _, err := os.Stat(filepath.Join(checkout, "ForeverFriends.toc")); err == nil {
		t.Fatal("wrote through the link")
	}
	out := Remove(dir, false, "ForeverFriends")
	if len(out) != 1 || !strings.Contains(out[0], "link") {
		t.Fatal(out)
	}
	if _, err := os.Stat(checkout); err != nil {
		t.Fatal("checkout gone")
	}
}

func TestForeignFolder(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "ForeverFriends"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "ForeverFriends/other.lua"), []byte("x"), 0o644)
	src := addon("0.1.0", map[string]string{"Core.lua": "a"})
	if _, err := Install(src, dir, false, nil); err != ErrForeign {
		t.Fatal(err)
	}
	if rep, _ := Install(src, dir, false, func(string) bool { return false }); rep.Action != LeftForeign {
		t.Fatal(rep)
	}
	if rep, err := Install(src, dir, false, func(string) bool { return true }); err != nil || rep.Action != Updated {
		t.Fatal(rep, err)
	}
}

func TestListedFileMissingFromBuild(t *testing.T) {
	src := fstest.MapFS{AddonName + ".toc": &fstest.MapFile{Data: []byte("## Interface: 1\n## Version: 1\nGone.lua\n")}}
	if _, err := Install(src, t.TempDir(), false, nil); err == nil {
		t.Fatal("installed a toc whose file is missing")
	}
}

func TestRemoveOnlyNamedFolders(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "ForeverFriends"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "Other"), 0o755)
	out := Remove(dir, false, "ForeverFriends", "ForeverFriendsData", "../x")
	if len(out) != 1 {
		t.Fatal(out)
	}
	if _, err := os.Stat(filepath.Join(dir, "Other")); err != nil {
		t.Fatal("removed another addon")
	}
}
