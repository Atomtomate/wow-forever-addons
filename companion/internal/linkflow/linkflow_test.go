package linkflow

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubfake"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

type fixture struct {
	h      *hubfake.Hub
	dir    config.Dir
	store  *config.Store
	env    Env
	now    time.Time
	polls  int
	onPoll func(n int)
}

func setup(t *testing.T) *fixture {
	f := &fixture{h: hubfake.New(t), dir: config.Dir(t.TempDir()), now: time.Unix(1_000_000, 0)}
	f.store, _ = f.dir.OpenStore()
	f.env = Env{HTTP: hubapi.NewHTTPClient(time.Second), UserAgent: "t", Now: func() time.Time { return f.now },
		Sleep: func(ctx context.Context, d time.Duration) error {
			f.polls++
			f.now = f.now.Add(d)
			if f.onPoll != nil {
				f.onPoll(f.polls)
			}
			return nil
		}}
	return f
}

func (f *fixture) client() *hubapi.Client {
	cfg, _ := f.dir.Load()
	key, _ := f.dir.LoadKey()
	return &hubapi.Client{Origin: cfg.Hub, Device: cfg.Device, Key: key, Seq: f.store, HTTP: f.env.HTTP, UserAgent: "t"}
}

func TestStartStoresEverythingThenWaitReturnsMember(t *testing.T) {
	f := setup(t)
	p, err := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "JULIAN-PC")
	if err != nil || p.Code == "" {
		t.Fatal(p, err)
	}
	cfg, _ := f.dir.Load()
	if cfg.Device == "" || cfg.Pending == nil || cfg.Pending.Code != p.Code || cfg.DeviceName != "JULIAN-PC" {
		t.Fatalf("%+v", cfg)
	}
	if _, err := f.dir.LoadKey(); err != nil {
		t.Fatal("key not stored")
	}
	f.onPoll = func(n int) {
		if n == 3 {
			f.h.Approve(p.Code, "Anna")
		}
	}
	member, err := Wait(context.Background(), f.env, f.client(), f.dir, p)
	if err != nil || member != "Anna" || f.polls != 3 {
		t.Fatal(member, err, f.polls)
	}
	cfg, _ = f.dir.Load()
	if cfg.Member != "Anna" || cfg.Pending != nil {
		t.Fatalf("%+v", cfg)
	}
}

func TestResumeBeforeExpiryShowsSameCodeAfterExpiryDoesNot(t *testing.T) {
	f := setup(t)
	p, _ := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc")
	cfg, _ := f.dir.Load()
	got, ok := Resume(cfg, f.h.URL, f.now.Add(599*time.Second))
	if !ok || got.Code != p.Code {
		t.Fatal(got, ok)
	}
	if _, ok := Resume(cfg, f.h.URL, f.now.Add(601*time.Second)); ok {
		t.Fatal("expired link resumed")
	}
	if _, ok := Resume(cfg, "https://other.example", f.now); ok {
		t.Fatal("link for another hub resumed")
	}
	if n := f.h.Count("/api/companion/link"); n != 1 {
		t.Fatal("started twice", n)
	}
}

func TestWaitUnauthorizedIsExpiredAndTimeoutToo(t *testing.T) {
	f := setup(t)
	p, _ := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc")
	f.onPoll = func(n int) { f.h.Remove(f.h.FirstDevice().ID) }
	if _, err := Wait(context.Background(), f.env, f.client(), f.dir, p); !errors.Is(err, ErrExpired) {
		t.Fatal(err)
	}
	// A declined or dead code must not be shown again: the next run starts a new link.
	cfg, _ := f.dir.Load()
	if cfg.Pending != nil {
		t.Fatal("pending link kept after a 401")
	}
	if _, ok := Resume(cfg, f.h.URL, f.now); ok {
		t.Fatal("dead code resumed")
	}
	f2 := setup(t)
	p2, _ := Start(context.Background(), f2.env, f2.dir, f2.store, f2.h.URL, "pc")
	if _, err := Wait(context.Background(), f2.env, f2.client(), f2.dir, p2); !errors.Is(err, ErrExpired) {
		t.Fatal(err)
	}
}

func TestOtherAnswersAreAskedAgain(t *testing.T) {
	f := setup(t)
	p, _ := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc")
	f.h.Approve(p.Code, "Anna")
	bad := 2
	f.h.Override = func(path string) (int, string, bool) {
		if path == "/api/companion/link/status" && bad > 0 {
			bad--
			return []int{500, 429}[bad], `{"ok":false}`, true
		}
		return 0, "", false
	}
	member, err := Wait(context.Background(), f.env, f.client(), f.dir, p)
	if err != nil || member != "Anna" || f.polls != 2 {
		t.Fatal(member, err, f.polls)
	}
}

func TestStartRefusals(t *testing.T) {
	f := setup(t)
	f.h.Override = func(string) (int, string, bool) { return 429, `{"ok":false}`, true }
	if _, err := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc"); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	f.h.Override = func(string) (int, string, bool) { return 400, `{"ok":false,"error":"bad"}`, true }
	if _, err := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc"); !errors.Is(err, ErrRejected) {
		t.Fatal(err)
	}
	f.h.Override = func(string) (int, string, bool) { return http.StatusInternalServerError, ``, true }
	if _, err := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "pc"); err == nil {
		t.Fatal("5xx accepted")
	}
	if cfg, _ := f.dir.Load(); cfg.Device != "" {
		t.Fatal("config written for a failed start")
	}
}

func TestDeviceNameCutToWhatTheHubKeeps(t *testing.T) {
	f := setup(t)
	if _, err := Start(context.Background(), f.env, f.dir, f.store, f.h.URL, "  "+strings.Repeat("ü", 60)); err != nil {
		t.Fatal(err)
	}
	cfg, _ := f.dir.Load()
	if n := len([]rune(cfg.DeviceName)); n != wire.MaxDeviceName {
		t.Fatalf("%d characters sent", n)
	}
}
