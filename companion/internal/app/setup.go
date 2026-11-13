package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/atomicfile"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/dataaddon"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/gamefind"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/install"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/instance"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/linkflow"
)

type setupOpts struct {
	hub, game, name                    string
	noBrowser, noAutostart, noSelfCopy bool
	configDirFlag                      string
}

func (c *cmdCtx) setup(ctx context.Context, args []string) int {
	fs, cfgDir := c.flags("setup")
	var o setupOpts
	fs.StringVar(&o.hub, "hub", "", "the site's address")
	fs.StringVar(&o.game, "game", "", "the game's _forever_ folder")
	fs.StringVar(&o.name, "name", "", "this PC's name on the site")
	fs.BoolVar(&c.ui.yes, "yes", false, "answer the questions with their defaults; for scripts")
	fs.BoolVar(&o.noBrowser, "no-browser", false, "do not open the browser")
	fs.BoolVar(&o.noAutostart, "no-autostart", false, "do not register the start with Windows")
	fs.BoolVar(&o.noSelfCopy, "no-self-copy", false, "do not copy this program or start it in the background")
	if fs.Parse(args) != nil {
		return ExitFailed
	}
	o.configDirFlag = *cfgDir
	u := c.ui
	dir, err := c.configDir(*cfgDir)
	if err != nil {
		u.say("Cannot use the config folder: %v", err)
		return ExitFailed
	}
	cfg, err := dir.Load()
	if err != nil {
		u.say("The saved link cannot be read (%v). Remove %s and run this again.", err, dir.Path(config.ConfigFile))
		return ExitFailed
	}
	store, err := dir.OpenStore()
	if err != nil {
		u.say("The saved state cannot be read: %v", err)
		return ExitFailed
	}
	hubFlag := ""
	if o.hub != "" {
		if hubFlag, err = hubapi.ParseOrigin(o.hub); err != nil {
			u.say("%v", err)
			return ExitFailed
		}
	}
	linked := cfg.Device != "" && cfg.Member != "" && cfg.Pending == nil && !store.Snapshot().Revoked
	again := linked && (hubFlag == "" || hubFlag == cfg.Hub)

	u.say("Forever Friends companion %s. It links this PC to your group's site, installs the addon and tells", c.version)
	u.say("the site when the game runs. Of the game's files it reads only the addon's own; it looks at which")
	u.say("programs are running only to see whether the game is one of them.")
	if again {
		u.say("This PC is linked to %s as %s. This run updates the addon and the program copy if they are older,", cfg.Hub, cfg.Member)
		u.say("and restarts the background copy.")
	} else {
		u.say("Nothing changes until you answer.")
	}

	game, ok := c.resolveGame(o.game, cfg.Game)
	if !ok {
		return ExitFailed
	}
	if !again {
		if code, done := c.link(ctx, dir, store, cfg, hubFlag, o.name, o.noBrowser); done {
			return code
		}
		cfg, _ = dir.Load()
	}
	if cfg.Game != game {
		cfg.Game = game
		if err := dir.Save(cfg); err != nil {
			u.say("Cannot save: %v", err)
			return ExitFailed
		}
	}
	rep, code := c.installAddons(game)
	if code != ExitOK {
		return code
	}
	if code := c.installProgram(ctx, dir, rep, o); code != ExitOK {
		return code
	}
	if !c.ui.yes {
		if strings.EqualFold(u.ask(`Press Enter to close, or type "remove" to unlink this PC and remove the companion`, ""), "remove") {
			return c.removeWith(ctx, dir, true)
		}
	}
	return ExitOK
}

// resolveGame finds the game folder: the flag, the saved one, a guess the member confirms, or
// what the member types.
func (c *cmdCtx) resolveGame(flagValue, saved string) (string, bool) {
	u := c.ui
	if flagValue != "" {
		g, ok := gamefind.Normalize(flagValue, c.isDir)
		if !ok {
			u.say("%s is not the game's folder (it holds no Interface or WTF folder).", flagValue)
		}
		return g, ok
	}
	if saved != "" && gamefind.Valid(saved, c.isDir) {
		return saved, true
	}
	if g, ok := gamefind.First(c.env.P.GameCandidates(), c.isDir); ok {
		if u.confirm(fmt.Sprintf("Game folder: found %s . Use it?", g), true) {
			return g, true
		}
	}
	if u.yes {
		u.say("No game folder found; give one with --game.")
		return "", false
	}
	for tries := 0; tries < 5; tries++ {
		in := u.ask(`Drag the game's "_forever_" folder here and press Enter`, "")
		if g, ok := gamefind.Normalize(in, c.isDir); ok {
			return g, true
		}
		u.say("That is not the game's folder. It holds Interface and WTF folders.")
	}
	return "", false
}

// link runs section 3.1: resume or start, show the code, open the browser, wait. done is true
// when setup should stop here with code: the link failed, or the member chose not to start one.
func (c *cmdCtx) link(ctx context.Context, dir config.Dir, store *config.Store, cfg config.Config, hubFlag, nameFlag string, noBrowser bool) (code int, done bool) {
	u := c.ui
	hub := hubFlag
	for hub == "" {
		in := u.ask("Site address (shown on the site's /companion page)", cfg.Hub)
		var err error
		if hub, err = hubapi.ParseOrigin(in); err != nil {
			u.say("%v", err)
			hub = ""
			if u.yes || in == "" {
				return ExitFailed, true
			}
		}
	}
	env := linkflow.Env{HTTP: c.env.HTTP, UserAgent: c.userAgent(), Now: c.env.Now, Sleep: c.env.Sleep}
	p, resumed := linkflow.Resume(cfg, hub, c.env.Now())
	if !resumed {
		if cfg.Pending != nil && cfg.Hub == hub && !u.confirm("The earlier link expired. Start a new one?", true) {
			return ExitOK, true
		}
		name := nameFlag
		if name == "" {
			host, _ := c.env.Hostname()
			name = u.ask("Name of this PC on the site", strings.TrimSpace(host))
		}
		var err error
		if p, err = linkflow.Start(ctx, env, dir, store, hub, name); err != nil {
			u.say("%v", err)
			return ExitFailed, true
		}
	}
	link := hub + "/companion/link?code=" + url.QueryEscape(p.Code)
	u.say("Your code:  %s", p.Code)
	if u.yes {
		u.say("CODE: %s", p.Code)
	}
	if !noBrowser {
		if err := c.env.P.OpenURL(link); err != nil {
			u.say("Could not open a browser (%v).", err)
		}
	}
	u.say("If no browser opened, go to %s", link)
	u.say("Waiting for you to press Link on the site (%d minutes; Ctrl+C stops, run this again to continue)...", int(p.Expires().Sub(c.env.Now()).Minutes())+1)
	cfg, _ = dir.Load()
	client, err := c.client(dir, cfg, store)
	if err != nil {
		u.say("%v", err)
		return ExitFailed, true
	}
	member, err := linkflow.Wait(ctx, env, client, dir, p)
	switch {
	case err == nil:
		u.say("Linked as %s.", member)
		return ExitOK, false
	case errors.Is(err, linkflow.ErrExpired):
		u.say("The code expired or was declined. Run this again to get a new one.")
		return ExitNotLinked, true
	case errors.Is(err, context.Canceled):
		u.say("Stopped. Run this again to continue with the same code.")
		return ExitInterrupted, true
	}
	u.say("%v", err)
	return ExitFailed, true
}

// installAddons puts both addon folders in place and says whether the game must be restarted.
// The report says what became of ForeverFriends, whose version the program copy follows.
func (c *cmdCtx) installAddons(game string) (install.Report, int) {
	u := c.ui
	addons := filepath.Join(game, "Interface", "AddOns")
	if err := os.MkdirAll(addons, 0o755); err != nil {
		u.say("Cannot create %s: %v", addons, err)
		return install.Report{}, ExitFailed
	}
	rep, err := install.Install(c.env.Addon, addons, c.env.P.CaseFold, func(q string) bool { return u.confirmRisky(q, false) })
	if err != nil {
		u.say("The addon could not be installed: %v", err)
		return rep, ExitFailed
	}
	u.say("Addon ForeverFriends %s: %s.", rep.Version, rep.Action)
	created, err := dataaddon.EnsureEmpty(filepath.Join(addons, dataaddon.FolderName), c.iface)
	if err != nil {
		u.say("The member list folder could not be written: %v", err)
		return rep, ExitFailed
	}
	if rep.Action == install.Installed || rep.Action == install.Updated || rep.Action == install.Repaired || created {
		u.say("Restart the game once so that it loads the addon (a running game does not see a new folder).")
	}
	return rep, ExitOK
}

// stopRunning takes the lock, asking a running companion to leave first and waiting up to
// stopWait on the clock it was given.
func (c *cmdCtx) stopRunning(ctx context.Context, dir config.Dir) (*instance.Lock, error) {
	l, err := instance.TryAcquire(dir.Path(config.LockFile))
	if err == nil {
		return l, nil
	}
	if !errors.Is(err, instance.ErrHeld) {
		return nil, err
	}
	stop := dir.Path(config.StopFile)
	if err := instance.RequestStop(stop); err != nil {
		return nil, err
	}
	defer instance.ClearStop(stop)
	deadline := c.env.Now().Add(stopWait)
	for c.env.Now().Before(deadline) {
		if err := c.env.Sleep(ctx, time.Second); err != nil {
			return nil, err
		}
		if l, err = instance.TryAcquire(dir.Path(config.LockFile)); err == nil {
			return l, nil
		}
	}
	return nil, errCannotStop
}

// installProgram copies this program to its own folder, replaces a running copy, starts it in
// the background and registers its start with Windows. Where the platform has no program
// folder, or with --no-self-copy, it only says how to start it. A download older than what is
// installed (the version of the installed addon, which the program shares) replaces neither
// the addon nor the program copy.
func (c *cmdCtx) installProgram(ctx context.Context, dir config.Dir, rep install.Report, o setupOpts) int {
	u := c.ui
	exe, err := c.env.P.Executable()
	progDir := c.env.P.ProgramDir()
	if err != nil || progDir == "" || o.noSelfCopy {
		if exe != "" {
			u.say("Start the companion with: %s run   (leave it running while you play; it does not start by itself on this system.)", exe)
		}
		return ExitOK
	}
	target := filepath.Join(progDir, programFile)
	if rep.Action == install.LeftNewer {
		if _, err := os.Stat(target); err == nil {
			u.say("This download (%s) is older than the installed version (%s): the addon and the program copy were left alone.", c.version, rep.Version)
			return c.ensureRunning(dir, target, o)
		}
	}
	if c.samePath(exe, target) {
		return c.ensureRunning(dir, target, o)
	}
	lock, err := c.stopRunning(ctx, dir)
	if err != nil {
		u.say("%v", err)
		return ExitFailed
	}
	data, err := os.ReadFile(exe)
	if err == nil {
		err = atomicfile.Write(target, data, 0o755, 0o755)
	}
	lock.Release()
	if err != nil {
		u.say("Cannot copy the companion to %s: %v", progDir, err)
		return ExitFailed
	}
	u.say("Companion copied to %s.", progDir)
	return c.startBackground(dir, target, o)
}

// ensureRunning starts the background copy unless one runs already.
func (c *cmdCtx) ensureRunning(dir config.Dir, target string, o setupOpts) int {
	if l, err := instance.TryAcquire(dir.Path(config.LockFile)); err == nil {
		l.Release()
		return c.startBackground(dir, target, o)
	}
	c.ui.say("The companion is running in the background.")
	return ExitOK
}

func (c *cmdCtx) startBackground(dir config.Dir, target string, o setupOpts) int {
	u := c.ui
	var extra []string
	if o.configDirFlag != "" {
		extra = []string{"--config-dir", string(dir)}
	}
	if err := c.env.P.SpawnDetached(target, append([]string{"run"}, extra...)...); err != nil {
		u.say("Could not start the companion in the background: %v", err)
		return ExitFailed
	}
	if o.noAutostart {
		u.say("It is running now.")
		return ExitOK
	}
	if err := c.env.P.SetAutostart(target, extra...); err != nil {
		u.say("The companion runs now but could not be set to start with Windows: %v", err)
	} else {
		u.say("It starts with Windows and is running now.")
	}
	return ExitOK
}
