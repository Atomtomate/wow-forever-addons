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
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
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

// Pending is a started link the member has not approved yet.
type Pending struct {
	Code     string
	Expires  time.Time
	Interval time.Duration
}

const defaultInterval = 5 * time.Second

// Resume returns the link a closed companion left behind, while it has not expired and belongs
// to the same hub; a new start would only spend the hub's limit on link starts.
func Resume(cfg config.Config, hub string, now time.Time) (Pending, bool) {
	p := cfg.Pending
	if p == nil || cfg.Hub != hub || cfg.Device == "" || !now.Before(time.Unix(p.ExpiresUnix, 0)) {
		return Pending{}, false
	}
	iv := time.Duration(p.IntervalSec) * time.Second
	if iv <= 0 {
		iv = defaultInterval
	}
	return Pending{Code: p.Code, Expires: time.Unix(p.ExpiresUnix, 0), Interval: iv}, true
}

// Start makes a new key pair, asks the hub for a code and stores the key, the device id, the
// code and the expiry before returning, so that a member who closes the window can resume.
func Start(ctx context.Context, e Env, dir config.Dir, store *config.Store, hub, deviceName string) (Pending, error) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Pending{}, err
	}
	out, res := hubapi.StartLink(ctx, e.HTTP, hub, e.UserAgent, deviceName, pub)
	switch res.Outcome {
	case hubapi.OK:
	case hubapi.Throttled:
		return Pending{}, ErrBusy
	case hubapi.Rejected:
		return Pending{}, fmt.Errorf("%w: %s", ErrRejected, res.Error)
	default:
		return Pending{}, fmt.Errorf("could not reach the site (%s)", reason(res))
	}
	cfg, err := dir.Load()
	if err != nil {
		return Pending{}, err
	}
	p := Pending{Code: out.UserCode, Expires: e.Now().Add(time.Duration(out.ExpiresIn) * time.Second), Interval: time.Duration(out.Interval) * time.Second}
	if p.Interval <= 0 {
		p.Interval = defaultInterval
	}
	if err := dir.SaveKey(key); err != nil {
		return Pending{}, err
	}
	if err := store.Reset(); err != nil {
		return Pending{}, err
	}
	cfg.Hub, cfg.Device, cfg.DeviceName, cfg.Member = hub, out.Device, deviceName, ""
	cfg.Pending = &config.Pending{Code: p.Code, ExpiresUnix: p.Expires.Unix(), IntervalSec: int(p.Interval / time.Second)}
	return p, dir.Save(cfg)
}

func reason(r hubapi.Result) string {
	if r.Error != "" {
		return r.Error
	}
	return fmt.Sprintf("status %d", r.Status)
}

// Wait asks the hub every interval until the member approved the code. It returns the member's
// name and clears the pending link from the config; ErrExpired when the time ran out or the
// hub answered 401 (expired or declined).
func Wait(ctx context.Context, e Env, dir config.Dir, store *config.Store, p Pending) (string, error) {
	cfg, err := dir.Load()
	if err != nil {
		return "", err
	}
	key, err := dir.LoadKey()
	if err != nil {
		return "", err
	}
	c := &hubapi.Client{Origin: cfg.Hub, Device: cfg.Device, Key: key, Seq: store, HTTP: e.HTTP, UserAgent: e.UserAgent}
	for {
		if !e.Now().Before(p.Expires) {
			return "", ErrExpired
		}
		st, res := c.LinkStatus(ctx)
		switch res.Outcome {
		case hubapi.OK:
			if st.Linked {
				cfg.Member, cfg.Pending = st.Member, nil
				return st.Member, dir.Save(cfg)
			}
		case hubapi.Unlinked:
			return "", ErrExpired
		case hubapi.Rejected:
			return "", fmt.Errorf("%w: %s", ErrRejected, res.Error)
		}
		// Any other answer is asked again at the same interval until the expiry.
		if err := e.Sleep(ctx, p.Interval); err != nil {
			return "", err
		}
	}
}
