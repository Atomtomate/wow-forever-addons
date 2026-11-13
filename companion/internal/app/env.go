package app

import (
	"context"
	iofs "io/fs"
	"os"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/platform"
)

// DefaultEnv is the real thing: the real clock, the real operating system, the embedded addon.
func DefaultEnv(addon iofs.FS) Env {
	return Env{
		Now:      time.Now,
		Sleep:    sleep,
		Addon:    addon,
		Hostname: os.Hostname,
		P: Platform{
			ConfigDir:       platform.ConfigDir,
			ProgramDir:      platform.ProgramDir,
			Executable:      platform.Executable,
			QueryVersion:    platform.QueryVersion,
			GameCandidates:  platform.GameCandidates,
			ListImages:      platform.ListImages,
			OpenURL:         platform.OpenURL,
			SetAutostart:    platform.SetAutostart,
			RemoveAutostart: platform.RemoveAutostart,
			SpawnDetached:   platform.SpawnDetached,
			PrepareConsole:  platform.PrepareConsole,
			CaseFold:        platform.CaseFoldPaths(),
		},
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
