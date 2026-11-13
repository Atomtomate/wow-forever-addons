package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/dataaddon"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/gamefind"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/install"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/instance"
)

func (c *cmdCtx) status(args []string) int {
	fs, cfgDir := c.flags("status")
	if fs.Parse(args) != nil {
		return ExitFailed
	}
	dir, err := c.configDir(*cfgDir)
	if err != nil {
		c.ui.say("%v", err)
		return ExitFailed
	}
	u := c.ui
	cfg, _ := dir.Load()
	store, _ := dir.OpenStore()
	u.say("Forever Friends companion %s", c.version)
	switch {
	case cfg.Device == "":
		u.say("Not linked. Run setup.")
		return ExitOK
	case cfg.Pending != nil:
		u.say("A link to %s is waiting for you to press Link on the site (code %s). Run setup to continue.", cfg.Hub, cfg.Pending.Code)
		return ExitOK
	case store != nil && store.Snapshot().Revoked:
		u.say("This PC is no longer linked to %s (it was removed on your profile). Run setup to link it again.", cfg.Hub)
		return ExitOK
	}
	u.say("Linked to %s as %s; this PC is called %s.", cfg.Hub, cfg.Member, cfg.DeviceName)
	u.say("Game folder: %s", cfg.Game)
	images, _ := c.env.P.ListImages()
	u.say("Game running now: %v", gamefind.Running(images, cfg.Game, c.env.P.CaseFold))
	if l, err := instance.TryAcquire(dir.Path(config.LockFile)); err == nil {
		l.Release()
		u.say("The companion is not running in the background.")
	} else {
		u.say("The companion is running in the background.")
	}
	u.say("Log: %s", dir.Path(config.LogFile))
	return ExitOK
}

func (c *cmdCtx) remove(ctx context.Context, args []string) int {
	fs, cfgDir := c.flags("remove")
	fs.BoolVar(&c.ui.yes, "yes", false, "do not ask")
	if fs.Parse(args) != nil {
		return ExitFailed
	}
	dir, err := c.configDir(*cfgDir)
	if err != nil {
		c.ui.say("%v", err)
		return ExitFailed
	}
	return c.removeWith(ctx, dir, false)
}

// removeWith unlinks this PC and takes away what the companion put on it. asked is true when
// the member has just typed "remove" and need not be asked again.
func (c *cmdCtx) removeWith(ctx context.Context, dir config.Dir, asked bool) int {
	u := c.ui
	if !asked && !u.confirm("This unlinks this PC, removes the addon folders ForeverFriends and ForeverFriendsData and the companion's own files. Continue?", false) {
		return ExitOK
	}
	cfg, _ := dir.Load()
	lock, err := c.stopRunning(ctx, dir)
	if err != nil {
		u.say("%v", err)
		return ExitFailed
	}
	if cfg.Device != "" {
		if !c.unlink(ctx, dir, cfg) && !(u.yes || u.confirmRisky("The site could not be reached, so this PC stays listed on your profile until you remove it there. Continue anyway?", false)) {
			lock.Release()
			return ExitFailed
		}
	}
	if err := c.env.P.RemoveAutostart(); err != nil {
		u.say("Could not remove the start with Windows: %v", err)
	}
	if cfg.Game != "" {
		for _, line := range install.Remove(filepath.Join(cfg.Game, "Interface", "AddOns"), install.AddonName, dataaddon.FolderName) {
			u.say("%s", line)
		}
	}
	lock.Release()
	for _, name := range []string{config.ConfigFile, config.StateFile, config.KeyFile, config.LogFile, config.LogFile + ".1", config.StopFile, config.LockFile} {
		_ = os.Remove(dir.Path(name))
	}
	_ = os.Remove(string(dir)) // only if empty
	c.removeProgram()
	u.say("Removed. The site no longer lists this PC.")
	return ExitOK
}

// unlink asks the hub to forget this device; a 401 means it already has.
func (c *cmdCtx) unlink(ctx context.Context, dir config.Dir, cfg config.Config) bool {
	key, err := dir.LoadKey()
	store, serr := dir.OpenStore()
	hub, herr := hubapi.ParseOrigin(cfg.Hub)
	if err != nil || serr != nil || herr != nil {
		return true // nothing was ever registered that could be undone from here
	}
	cl := &hubapi.Client{Origin: hub, Device: cfg.Device, Key: key, Seq: store, HTTP: c.env.HTTP, UserAgent: c.userAgent()}
	switch cl.Unlink(ctx).Outcome {
	case hubapi.OK, hubapi.Unlinked:
		return true
	}
	return false
}

func (c *cmdCtx) removeProgram() {
	progDir := c.env.P.ProgramDir()
	exe, err := c.env.P.Executable()
	if progDir == "" || err != nil {
		return
	}
	target := filepath.Join(progDir, programFile)
	if c.samePath(exe, target) {
		c.ui.say("This is the installed copy and cannot delete itself: delete the folder %s when this window closes.", progDir)
		return
	}
	if os.Remove(target) == nil {
		_ = os.Remove(progDir)
		c.ui.say("Companion program removed from %s.", progDir)
	}
}
