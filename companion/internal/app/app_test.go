package app

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubfake"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/instance"
)

const svFile = `ForeverFriendsDB = { version = 1, characters = {
 ["Thrallina-Everlook"] = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 23,
   skills = { { id = 171, name = "Alchimie", rank = 145, max = 150 } }, seen = 1763000000 } } }`

type rig struct {
	t             *testing.T
	h             *hubfake.Hub
	cfg           string
	game          string
	prog          string
	exe           string
	env           Env
	out           *bytes.Buffer
	now           time.Time
	images        []string
	spawned       [][]string
	opened        []string
	approve       bool
	onSleep       func()
	autostart     string
	autostartArgs []string
}

func addon() fs.FS {
	return fstest.MapFS{
		"ForeverFriends.toc": {Data: []byte("## Interface: 16001\n## Version: 0.1.0\nCore.lua\n")},
		"Core.lua":           {Data: []byte("-- addon\n")},
	}
}

func newRig(t *testing.T) *rig {
	root := t.TempDir()
	r := &rig{t: t, h: hubfake.New(t), cfg: filepath.Join(root, "cfg"), game: filepath.Join(root, "WoW", "_forever_"),
		prog: filepath.Join(root, "prog"), exe: filepath.Join(root, "downloads", "companion.exe"), out: &bytes.Buffer{}, now: time.Unix(1_800_000_000, 0), approve: true}
	for _, d := range []string{filepath.Join(r.game, "Interface", "AddOns"), filepath.Dir(r.exe), filepath.Join(r.game, "WTF", "Account", "ACC", "SavedVariables")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(r.game, "WTF", "Account", "ACC", "SavedVariables", "ForeverFriends.lua"), []byte(svFile), 0o644)
	_ = os.WriteFile(r.exe, []byte("exe-bytes"), 0o755)
	r.h.Roster = `{"members":[{"name":"Anna","characters":[{"name":"Thrallina","class":"Shaman","level":23,"main":true}]}]}`
	r.env = Env{
		Stdin: strings.NewReader(""), Stdout: r.out, Stderr: r.out,
		Now: func() time.Time { return r.now },
		Sleep: func(ctx context.Context, d time.Duration) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			r.now = r.now.Add(d)
			if r.approve {
				if dev := r.h.FirstDevice(); dev != nil && !dev.Linked {
					r.h.Approve(dev.Code, "Anna")
				}
			}
			if r.onSleep != nil {
				r.onSleep()
			}
			return nil
		},
		HTTP:     hubapi.NewHTTPClient(2 * time.Second),
		Addon:    addon(),
		Hostname: func() (string, error) { return "JULIAN-PC", nil },
		P: Platform{
			ProgramDir:      func() string { return r.prog },
			Executable:      func() (string, error) { return r.exe, nil },
			GameCandidates:  func() []string { return nil },
			ListImages:      func() ([]string, error) { return r.images, nil },
			OpenURL:         func(u string) error { r.opened = append(r.opened, u); return nil },
			SetAutostart:    func(e string, a ...string) error { r.autostart, r.autostartArgs = e, a; return nil },
			RemoveAutostart: func() error { r.autostart = ""; return nil },
			SpawnDetached: func(e string, a ...string) error {
				r.spawned = append(r.spawned, append([]string{e}, a...))
				return nil
			},
		},
	}
	return r
}

func (r *rig) main(args ...string) int {
	r.t.Helper()
	r.out.Reset()
	return Main(context.Background(), args, r.env)
}

func (r *rig) setupArgs(extra ...string) []string {
	return append([]string{"setup", "--yes", "--hub", r.h.URL, "--game", r.game, "--config-dir", r.cfg, "--no-browser", "--no-autostart", "--no-self-copy"}, extra...)
}

func (r *rig) addons() string { return filepath.Join(r.game, "Interface", "AddOns") }

func TestSetupNonInteractiveLinksInstallsThenRunOnceSyncs(t *testing.T) {
	r := newRig(t)
	if code := r.main(r.setupArgs()...); code != ExitOK {
		t.Fatalf("setup %d\n%s", code, r.out)
	}
	out := r.out.String()
	if !strings.Contains(out, "\nCODE: ") && !strings.HasPrefix(out, "CODE: ") {
		t.Fatalf("no CODE line:\n%s", out)
	}
	cfg, _ := config.Dir(r.cfg).Load()
	if cfg.Member != "Anna" || cfg.Pending != nil || cfg.Game != r.game || cfg.DeviceName != "JULIAN-PC" {
		t.Fatalf("%+v", cfg)
	}
	for _, f := range []string{"ForeverFriends/Core.lua", "ForeverFriends/ForeverFriends.toc", "ForeverFriendsData/ForeverFriendsData.toc", "ForeverFriendsData/Data.lua"} {
		if _, err := os.Stat(filepath.Join(r.addons(), f)); err != nil {
			t.Fatal("missing", f)
		}
	}
	empty, _ := os.ReadFile(filepath.Join(r.addons(), "ForeverFriendsData/Data.lua"))
	if !strings.Contains(string(empty), "members = {},") || !strings.Contains(string(empty), "written = 0,") {
		t.Fatal(string(empty))
	}

	if code := r.main("run", "--once", "--config-dir", r.cfg); code != ExitOK || !strings.Contains(r.out.String(), "sync: ok") {
		t.Fatalf("run %d\n%s", code, r.out)
	}
	bodies := r.h.SyncBodies()
	if len(bodies) != 1 || bodies[0]["game_running"] != false || len(bodies[0]["characters"].([]any)) != 1 {
		t.Fatalf("%v", bodies)
	}
	if _, has := bodies[0]["companion"]; has {
		t.Fatal("version field sent")
	}
	data, _ := os.ReadFile(filepath.Join(r.addons(), "ForeverFriendsData/Data.lua"))
	if !strings.Contains(string(data), `name = "Anna"`) || !strings.Contains(string(data), `main = true`) {
		t.Fatal(string(data))
	}

	r.h.Remove(cfg.Device)
	if code := r.main("run", "--once", "--config-dir", r.cfg); code != ExitUnlinked {
		t.Fatalf("after removal: %d", code)
	}
	n := r.h.Count("/api/companion/sync")
	if code := r.main("run", "--once", "--config-dir", r.cfg); code != ExitUnlinked || r.h.Count("/api/companion/sync") != n {
		t.Fatal("kept calling the hub after a 401")
	}
	r.main("status", "--config-dir", r.cfg)
	if !strings.Contains(r.out.String(), "no longer linked") {
		t.Fatal(r.out)
	}
}

func TestRunOnceUnreachableAndAlreadyRunning(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	r.h.Override = func(string) (int, string, bool) { return 503, ``, true }
	if code := r.main("run", "--once", "--config-dir", r.cfg); code != ExitUnreachable {
		t.Fatal(code)
	}
	l, _ := instance.TryAcquire(filepath.Join(r.cfg, config.LockFile))
	defer l.Release()
	if code := r.main("run", "--once", "--config-dir", r.cfg); code != ExitRunning {
		t.Fatal(code)
	}
}

func TestSetupResumesPendingLinkWithoutNewStart(t *testing.T) {
	r := newRig(t)
	orig := r.env.Sleep
	r.approve = false
	ctx, cancel := context.WithCancel(context.Background())
	r.env.Sleep = func(c context.Context, d time.Duration) error { cancel(); return c.Err() }
	r.out.Reset()
	if code := Main(ctx, r.setupArgs(), r.env); code != ExitInterrupted {
		t.Fatalf("%d\n%s", code, r.out)
	}
	first := r.h.FirstDevice().Code
	if !strings.Contains(r.out.String(), first) {
		t.Fatal("code not shown")
	}
	r.env.Sleep, r.approve = orig, true
	if code := r.main(r.setupArgs()...); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if r.h.Count("/api/companion/link") != 1 || !strings.Contains(r.out.String(), first) {
		t.Fatalf("started %d links\n%s", r.h.Count("/api/companion/link"), r.out)
	}
}

func TestRemoveCallsUnlinkThenDeletes(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	if len(r.h.Devices) != 1 {
		t.Fatal("not linked")
	}
	if code := r.main("remove", "--yes", "--config-dir", r.cfg); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if len(r.h.Devices) != 0 || r.h.Count("/api/companion/unlink") != 1 {
		t.Fatal("hub still lists the PC")
	}
	for _, p := range []string{filepath.Join(r.addons(), "ForeverFriends"), filepath.Join(r.addons(), "ForeverFriendsData"), r.cfg} {
		if _, err := os.Stat(p); err == nil {
			t.Fatal("left behind:", p)
		}
	}
	if _, err := os.Stat(filepath.Join(r.game, "WTF", "Account", "ACC", "SavedVariables", "ForeverFriends.lua")); err != nil {
		t.Fatal("removed the game's own file")
	}
}

func TestRemoveWhenHubUnreachableAsks(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	r.h.Override = func(string) (int, string, bool) { return 503, ``, true }
	r.env.Stdin = strings.NewReader("y\nn\n")
	if code := r.main("remove", "--config-dir", r.cfg); code != ExitFailed {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if _, err := os.Stat(filepath.Join(r.cfg, config.ConfigFile)); err != nil {
		t.Fatal("config removed although the member said no")
	}
	r.env.Stdin = strings.NewReader("y\ny\n")
	if code := r.main("remove", "--config-dir", r.cfg); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
}

func TestRemoveAlreadyGoneOnHubIsDone(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	cfg, _ := config.Dir(r.cfg).Load()
	r.h.Remove(cfg.Device)
	if code := r.main("remove", "--yes", "--config-dir", r.cfg); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
}

func TestSetupCopiesProgramStopsRunningInstanceAndStartsIt(t *testing.T) {
	r := newRig(t)
	if err := os.MkdirAll(r.cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	old, err := instance.TryAcquire(filepath.Join(r.cfg, config.LockFile))
	if err != nil {
		t.Fatal(err)
	}
	r.onSleep = func() { // the running copy notices the request at its next tick and leaves
		if instance.StopRequested(filepath.Join(r.cfg, config.StopFile)) && old != nil {
			old.Release()
			old = nil
		}
	}
	args := []string{"setup", "--yes", "--hub", r.h.URL, "--game", r.game, "--config-dir", r.cfg, "--no-browser"}
	if code := r.main(args...); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if old != nil {
		t.Fatal("the running copy was never asked to stop")
	}
	if b, _ := os.ReadFile(filepath.Join(r.prog, programFile)); string(b) != "exe-bytes" {
		t.Fatal("program not copied")
	}
	if len(r.spawned) != 1 || r.spawned[0][1] != "run" || r.autostart != filepath.Join(r.prog, programFile) {
		t.Fatalf("spawned %v autostart %q", r.spawned, r.autostart)
	}
	if instance.StopRequested(filepath.Join(r.cfg, config.StopFile)) {
		t.Fatal("stop request left behind, the new copy would exit at once")
	}
}

func TestSetupCannotStopRunningInstanceReplacesNothing(t *testing.T) {
	r := newRig(t)
	_ = os.MkdirAll(r.cfg, 0o700)
	old, _ := instance.TryAcquire(filepath.Join(r.cfg, config.LockFile))
	defer old.Release()
	args := []string{"setup", "--yes", "--hub", r.h.URL, "--game", r.game, "--config-dir", r.cfg, "--no-browser"}
	if code := r.main(args...); code != ExitFailed || !strings.Contains(r.out.String(), "Task Manager") {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if _, err := os.Stat(filepath.Join(r.prog, programFile)); err == nil {
		t.Fatal("copied over a running program")
	}
}

// A download older than what is installed replaces neither the addon nor the program copy, says
// so in one line, and goes on.
func TestOlderDownloadReplacesNeitherAddonNorProgram(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	cfgBefore, _ := os.ReadFile(filepath.Join(r.cfg, config.ConfigFile))
	_ = os.MkdirAll(r.prog, 0o755)
	_ = os.WriteFile(filepath.Join(r.prog, programFile), []byte("newer program"), 0o755)
	tocPath := filepath.Join(r.addons(), "ForeverFriends", "ForeverFriends.toc")
	_ = os.WriteFile(tocPath, []byte("## Interface: 16001\n## Version: 9.0.0\nCore.lua\n"), 0o644)
	_ = os.WriteFile(filepath.Join(r.addons(), "ForeverFriends", "Core.lua"), []byte("newer addon"), 0o644)
	args := []string{"setup", "--yes", "--config-dir", r.cfg, "--no-browser", "--no-autostart"}
	if code := r.main(args...); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if b, _ := os.ReadFile(filepath.Join(r.prog, programFile)); string(b) != "newer program" {
		t.Fatal("program copy replaced by an older download")
	}
	if b, _ := os.ReadFile(filepath.Join(r.addons(), "ForeverFriends", "Core.lua")); string(b) != "newer addon" {
		t.Fatal("addon replaced by an older download")
	}
	if !strings.Contains(r.out.String(), "older than the installed version (9.0.0)") {
		t.Fatal(r.out)
	}
	if len(r.spawned) == 0 {
		t.Fatal("the background copy was not started")
	}
	after, _ := os.ReadFile(filepath.Join(r.cfg, config.ConfigFile))
	if string(after) != string(cfgBefore) {
		t.Fatal("the link changed")
	}
}

func TestDecliningANewLinkStopsSetup(t *testing.T) {
	r := newRig(t)
	r.approve = false
	r.env.Sleep = func(ctx context.Context, d time.Duration) error { r.now = r.now.Add(d); return nil }
	// a link started and left open, then the window closed: pending and not yet expired, then late
	ctx, cancel := context.WithCancel(context.Background())
	r.env.Sleep = func(c context.Context, d time.Duration) error { cancel(); return c.Err() }
	Main(ctx, r.setupArgs(), r.env)
	r.now = r.now.Add(time.Hour)
	r.env.Sleep = func(ctx context.Context, d time.Duration) error { r.now = r.now.Add(d); return nil }
	r.env.Stdin = strings.NewReader("n\n")
	args := []string{"setup", "--hub", r.h.URL, "--game", r.game, "--config-dir", r.cfg, "--no-browser", "--no-autostart"}
	if code := r.main(args...); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	if len(r.spawned) != 0 || r.autostart != "" {
		t.Fatal("started or registered an unlinked PC")
	}
	if _, err := os.Stat(filepath.Join(r.addons(), "ForeverFriends")); err == nil {
		t.Fatal("installed the addon after the member declined")
	}
}

func TestDeclinedCodeIsNotShownAgain(t *testing.T) {
	r := newRig(t)
	r.approve = false
	r.onSleep = func() {
		if d := r.h.FirstDevice(); d != nil {
			r.h.Remove(d.ID) // the member pressed Decline
		}
	}
	if code := r.main(r.setupArgs()...); code != ExitNotLinked {
		t.Fatalf("%d\n%s", code, r.out)
	}
	first := r.h.Count("/api/companion/link")
	r.onSleep, r.approve = nil, true
	if code := r.main(r.setupArgs()...); code != ExitOK || r.h.Count("/api/companion/link") != first+1 {
		t.Fatalf("%d, %d starts\n%s", code, r.h.Count("/api/companion/link"), r.out)
	}
}

func TestAutostartKeepsTheConfigFolder(t *testing.T) {
	r := newRig(t)
	args := []string{"setup", "--yes", "--hub", r.h.URL, "--game", r.game, "--config-dir", r.cfg, "--no-browser"}
	r.main(args...)
	if len(r.autostartArgs) != 2 || r.autostartArgs[0] != "--config-dir" || r.autostartArgs[1] != r.cfg {
		t.Fatalf("%v", r.autostartArgs)
	}
}

func TestChangingTheGameFolderStillFillsTheNewMemberList(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	r.main("run", "--once", "--config-dir", r.cfg) // the hub's list is written, its etag stored
	other := filepath.Join(filepath.Dir(r.game), "other", "_forever_")
	_ = os.MkdirAll(filepath.Join(other, "Interface", "AddOns"), 0o755)
	if code := r.main("setup", "--yes", "--game", other, "--config-dir", r.cfg, "--no-browser", "--no-autostart", "--no-self-copy"); code != ExitOK {
		t.Fatalf("%d\n%s", code, r.out)
	}
	r.main("run", "--once", "--config-dir", r.cfg)
	data, _ := os.ReadFile(filepath.Join(other, "Interface", "AddOns", "ForeverFriendsData", "Data.lua"))
	if !strings.Contains(string(data), `name = "Anna"`) {
		t.Fatalf("the new game folder's list stayed empty:\n%s", data)
	}
}

func TestRemoveSaysWhatHappenedAndWhatStays(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	r.h.Override = func(string) (int, string, bool) { return 503, ``, true }
	r.main("remove", "--yes", "--config-dir", r.cfg)
	out := r.out.String()
	if strings.Contains(out, "The site no longer lists this PC") || !strings.Contains(out, "may still list it") {
		t.Fatalf("claims success after a failed unlink:\n%s", out)
	}
	for _, want := range []string{"your characters on the site", "ForeverFriends.lua"} {
		if !strings.Contains(out, want) {
			t.Errorf("does not say that %q stays:\n%s", want, out)
		}
	}
	r2 := newRig(t)
	r2.main(r2.setupArgs()...)
	r2.main("remove", "--yes", "--config-dir", r2.cfg)
	if !strings.Contains(r2.out.String(), "The site no longer lists this PC") {
		t.Fatal(r2.out)
	}
}

func TestStatusShowsCharactersAndAnUnreadableFile(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	r.main("status", "--config-dir", r.cfg)
	if !strings.Contains(r.out.String(), "Characters in the addon's file: 1") {
		t.Fatal(r.out)
	}
	_ = os.WriteFile(filepath.Join(r.game, "WTF", "Account", "ACC", "SavedVariables", "ForeverFriends.lua"), []byte("ForeverFriendsDB = { "), 0o644)
	r.main("status", "--config-dir", r.cfg)
	if !strings.Contains(r.out.String(), "Cannot use WTF/Account/ACC/SavedVariables/ForeverFriends.lua") {
		t.Fatal(r.out)
	}
}

func TestSetupSaysNothingChangesOnlyOnAFirstRun(t *testing.T) {
	r := newRig(t)
	r.main(r.setupArgs()...)
	if !strings.Contains(r.out.String(), "Nothing changes until you answer.") || !strings.Contains(r.out.String(), "looks at which") {
		t.Fatal(r.out)
	}
	r.main(r.setupArgs()...)
	if strings.Contains(r.out.String(), "Nothing changes until you answer.") || !strings.Contains(r.out.String(), "updates the addon") {
		t.Fatal(r.out)
	}
}

func TestNothingWrittenOutsideTheTwoAddonFolders(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.game)
	r.main(r.setupArgs()...)
	r.main("run", "--once", "--config-dir", r.cfg)
	after := snapshot(t, r.game)
	for p, c := range after {
		if old, had := before[p]; had && old == c {
			continue
		}
		rel, _ := filepath.Rel(r.game, p)
		seg := strings.Split(filepath.ToSlash(rel), "/")
		if len(seg) < 4 || seg[0] != "Interface" || seg[1] != "AddOns" || (seg[2] != "ForeverFriends" && seg[2] != "ForeverFriendsData") {
			t.Errorf("wrote or changed %s", rel)
		}
	}
}

func snapshot(t *testing.T, root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := os.ReadFile(p)
			out[p] = string(b)
		}
		return nil
	})
	return out
}

func TestGameFolderPromptAndBadFlag(t *testing.T) {
	r := newRig(t)
	args := []string{"setup", "--yes", "--hub", r.h.URL, "--game", filepath.Join(r.game, "nowhere"), "--config-dir", r.cfg, "--no-browser"}
	if code := r.main(args...); code != ExitFailed {
		t.Fatal(code)
	}
	r2 := newRig(t)
	r2.env.P.GameCandidates = func() []string { return []string{r2.game} }
	r2.env.Stdin = strings.NewReader("\n\n\n")
	a := []string{"setup", "--hub", r2.h.URL, "--config-dir", r2.cfg, "--no-browser", "--no-autostart", "--no-self-copy"}
	if code := r2.main(a...); code != ExitOK {
		t.Fatalf("%d\n%s", code, r2.out)
	}
	if !strings.Contains(r2.out.String(), "found "+r2.game) {
		t.Fatal(r2.out)
	}
}

func TestUnknownCommandAndVersion(t *testing.T) {
	r := newRig(t)
	if r.main("frobnicate") != ExitFailed {
		t.Fatal("unknown command accepted")
	}
	if r.main("version") != ExitOK || strings.TrimSpace(r.out.String()) != "0.1.0" {
		t.Fatal(r.out)
	}
}
