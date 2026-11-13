package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/applog"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/dataaddon"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/gamefind"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/instance"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/savedvars"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/syncer"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// run is the background command: one instance, the loop until asked to stop. With --once it
// does one tick and one request, for scripts and for the end-to-end test.
func (c *cmdCtx) run(ctx context.Context, args []string) int {
	fs, cfgDir := c.flags("run")
	once := fs.Bool("once", false, "one sync, then exit (3 unlinked, 4 unreachable, 5 already running)")
	if fs.Parse(args) != nil {
		return ExitFailed
	}
	dir, err := c.configDir(*cfgDir)
	if err != nil {
		fmt.Fprintln(c.env.Stderr, err)
		return ExitFailed
	}
	log := applog.New(dir.Path(config.LogFile), logMaxBytes, c.env.Now)
	lock, err := instance.TryAcquire(dir.Path(config.LockFile))
	if err != nil {
		if *once {
			fmt.Fprintln(c.env.Stdout, "The companion is already running.")
			return ExitRunning
		}
		return ExitOK
	}
	defer lock.Release()
	instance.ClearStop(dir.Path(config.StopFile))

	cfg, err := dir.Load()
	if err != nil || cfg.Device == "" || cfg.Pending != nil || cfg.Member == "" {
		log.Printf("not linked; run setup")
		fmt.Fprintln(c.env.Stdout, "This PC is not linked. Run setup first.")
		return ExitFailed
	}
	store, err := dir.OpenStore()
	if err != nil {
		log.Printf("state unreadable: %v", err)
		return ExitFailed
	}
	if store.Snapshot().Revoked {
		fmt.Fprintln(c.env.Stdout, "sync: unlinked (this PC is no longer linked; run setup)")
		return mapExit(*once, ExitUnlinked)
	}
	key, err := dir.LoadKey()
	hub, herr := hubapi.ParseOrigin(cfg.Hub)
	if err != nil || herr != nil {
		log.Printf("link unusable: key %v, address %v", err, herr)
		return ExitFailed
	}
	client := &hubapi.Client{Origin: hub, Device: cfg.Device, Key: key, Seq: store, HTTP: c.env.HTTP, UserAgent: c.userAgent()}
	dataDir := filepath.Join(cfg.Game, "Interface", "AddOns", dataaddon.FolderName)
	s := syncer.New(syncer.Deps{
		Hub: client,
		Now: c.env.Now,
		Running: func() bool {
			images, _ := c.env.P.ListImages()
			return gamefind.Running(images, cfg.Game, c.env.P.CaseFold)
		},
		Read:             func() savedvars.Snapshot { return savedvars.Read(os.DirFS(cfg.Game)) },
		Store:            store,
		RosterFileExists: func() bool { _, err := os.Stat(filepath.Join(dataDir, "Data.lua")); return err == nil },
		WriteRoster: func(r wire.Roster, now time.Time) error {
			_, err := dataaddon.Write(dataDir, r, c.iface, now)
			return err
		},
		Log: log.Printf,
	})
	if *once {
		s.Tick(ctx)
		return c.onceResult(s)
	}
	log.Printf("companion %s started", c.version)
	err = s.Run(ctx, c.env.Sleep, func() bool { return instance.StopRequested(dir.Path(config.StopFile)) })
	log.Printf("companion stopped (status %d, %v)", s.Status(), err)
	return ExitOK
}

func mapExit(once bool, code int) int {
	if once {
		return code
	}
	return ExitOK
}

func (c *cmdCtx) onceResult(s *syncer.Syncer) int {
	out := c.env.Stdout
	if s.Status() == syncer.Revoked {
		fmt.Fprintln(out, "sync: unlinked")
		return ExitUnlinked
	}
	switch s.LastOutcome() {
	case hubapi.OK:
		fmt.Fprintln(out, "sync: ok")
		return ExitOK
	case hubapi.Rejected:
		fmt.Fprintln(out, "sync: refused")
		return ExitRefused
	case hubapi.Throttled:
		fmt.Fprintln(out, "sync: throttled")
		return ExitThrottled
	}
	fmt.Fprintln(out, "sync: unreachable")
	return ExitUnreachable
}
