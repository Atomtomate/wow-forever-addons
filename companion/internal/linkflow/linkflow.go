// Package linkflow is docs/CONTRACT.md §3.1: start a link with a new key pair, remember it so a
// closed window can resume, and poll until the member has approved the code in the browser.
package linkflow

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// Errors the caller turns into words for the member.
var (
	ErrBusy     = errors.New("the site is busy; try again in a few minutes")
	ErrExpired  = errors.New("the link expired or was declined")
	ErrRejected = errors.New("the site refused the request")
)

// Env is what the flow takes from outside.
type Env struct {
	HTTP      *http.Client
	UserAgent string
	Now       func() time.Time
	Sleep     func(context.Context, time.Duration) error
}

// Resume returns the link a closed companion left behind, while it has not expired and belongs
// to the same hub; a new start would only spend the hub's limit on link starts.
func Resume(cfg config.Config, hub string, now time.Time) (config.Pending, bool) {
	p := cfg.Pending
	if p == nil || cfg.Hub != hub || cfg.Device == "" || !now.Before(p.Expires()) {
		return config.Pending{}, false
	}
	return *p, true
}

// Start makes a new key pair, asks the hub for a code and stores the key, the device id, the
// code and the expiry before returning, so that a member who closes the window can resume. The
// PC's name is cut to what the hub keeps; the hub cleans it.
func Start(ctx context.Context, e Env, dir config.Dir, store *config.Store, hub, deviceName string) (config.Pending, error) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return config.Pending{}, err
	}
	if r := []rune(strings.TrimSpace(deviceName)); len(r) > wire.MaxDeviceName {
		deviceName = string(r[:wire.MaxDeviceName])
	}
	out, res := hubapi.StartLink(ctx, e.HTTP, hub, e.UserAgent, deviceName, pub)
	switch res.Outcome {
	case hubapi.OK:
	case hubapi.Throttled:
		return config.Pending{}, ErrBusy
	case hubapi.Rejected:
		return config.Pending{}, fmt.Errorf("%w: %s", ErrRejected, res.Error)
	default:
		return config.Pending{}, fmt.Errorf("could not reach the site (%s)", reason(res))
	}
	cfg, err := dir.Load()
	if err != nil {
		return config.Pending{}, err
	}
	p := config.Pending{Code: out.UserCode, ExpiresUnix: e.Now().Add(time.Duration(out.ExpiresIn) * time.Second).Unix(), IntervalSec: out.Interval}
	if err := dir.SaveKey(key); err != nil {
		return config.Pending{}, err
	}
	if err := store.Reset(); err != nil {
		return config.Pending{}, err
	}
	cfg.Hub, cfg.Device, cfg.DeviceName, cfg.Member = hub, out.Device, deviceName, ""
	cfg.Pending = &p
	return p, dir.Save(cfg)
}

func reason(r hubapi.Result) string {
	if r.Error != "" {
		return r.Error
	}
	return fmt.Sprintf("status %d", r.Status)
}

// Wait asks the hub every interval until the member approved the code, with c, the client for
// the device Start made. It returns the member's name and clears the pending link from the
// config; ErrExpired when the time ran out or the hub answered 401 (expired or declined), which
// also clears the link so that the next run starts a new one instead of showing a dead code.
func Wait(ctx context.Context, e Env, c *hubapi.Client, dir config.Dir, p config.Pending) (string, error) {
	cfg, err := dir.Load()
	if err != nil {
		return "", err
	}
	expire := func() (string, error) {
		cfg.Pending = nil
		if err := dir.Save(cfg); err != nil {
			return "", err
		}
		return "", ErrExpired
	}
	for {
		if !e.Now().Before(p.Expires()) {
			return expire()
		}
		st, res := c.LinkStatus(ctx)
		switch res.Outcome {
		case hubapi.OK:
			if st.Linked {
				cfg.Member, cfg.Pending = st.Member, nil
				return st.Member, dir.Save(cfg)
			}
		case hubapi.Unlinked:
			return expire()
		case hubapi.Rejected:
			return "", fmt.Errorf("%w: %s", ErrRejected, res.Error)
		}
		// Any other answer is asked again at the same interval until the expiry.
		if err := e.Sleep(ctx, p.Interval()); err != nil {
			return "", err
		}
	}
}
