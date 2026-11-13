// Package syncer is the background loop: when to send, what to send, and what each answer does
// to the schedule (docs/CONTRACT.md §3.3). It takes the clock, the hub, the process list and the
// files as parameters, so a test steps through hours without waiting a second.
package syncer

import (
	"context"
	"maps"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/savedvars"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// The schedule, in the contract's numbers.
const (
	TickEvery    = 10 * time.Second // the process is looked at far more often than every 15 s
	RunningEvery = 60 * time.Second
	IdleEvery    = 300 * time.Second
	ThrottleWait = 300 * time.Second
	RetryFast    = 30 * time.Second
	RetrySlow    = 120 * time.Second
	// FastRetries is how many failures in a row are retried at RetryFast before RetrySlow.
	FastRetries = 5
	// ResumeGap is a silence between ticks long enough to mean the PC slept.
	ResumeGap = 60 * time.Second
	// ResumeGrace is how long after a wake failures are not counted: a laptop's own network
	// coming back is not the hub being down.
	ResumeGrace = 60 * time.Second
	timeFormat  = "2006-01-02T15:04:05Z"
)

// Hub is the part of hubapi.Client the loop uses.
type Hub interface {
	Sync(ctx context.Context, req wire.SyncRequest) (wire.SyncResponse, hubapi.Result)
}

// Deps is everything the loop reaches outside itself through.
type Deps struct {
	Hub     Hub
	Now     func() time.Time
	Running func() bool
	Read    func() savedvars.Snapshot
	Store   *config.Store
	// RosterHash names the member list now in the game folder (dataaddon.Hash), "" if none.
	RosterHash  func() string
	WriteRoster func(wire.Roster, time.Time) error
	Log         func(format string, args ...any)
}

// Syncer holds the loop's memory.
type Syncer struct {
	d           Deps
	revoked     bool
	last        hubapi.Outcome
	lastTick    time.Time
	haveRunning bool
	running     bool
	nextDue     time.Time
	fails       int
	firstFail   time.Time
	graceUntil  time.Time
	prevFP      map[string]string
	reported    map[string]string // fingerprints of files already reported as unreadable
}

// New returns a loop that has not sent anything.
func New(d Deps) *Syncer {
	return &Syncer{d: d, revoked: d.Store.Snapshot().Revoked, reported: map[string]string{}}
}

// Revoked reports whether the hub answered 401: this PC is no longer linked.
func (s *Syncer) Revoked() bool { return s.revoked }

// LastOutcome is the outcome of the most recent request.
func (s *Syncer) LastOutcome() hubapi.Outcome { return s.last }

func every(running bool) time.Duration {
	if running {
		return RunningEvery
	}
	return IdleEvery
}

// Tick looks at the game and the files, and sends when something is due.
func (s *Syncer) Tick(ctx context.Context) {
	if s.revoked {
		return
	}
	// Round(0) drops the monotonic reading, which stops while a Linux or macOS PC sleeps: the
	// gap that tells a resume from sleep is measured on the wall clock.
	now := s.d.Now().Round(0)
	if !s.lastTick.IsZero() && now.Sub(s.lastTick) > ResumeGap {
		s.fails, s.firstFail = 0, time.Time{}
		s.graceUntil = now.Add(ResumeGrace)
		s.nextDue = now
	}
	s.lastTick = now

	running := s.d.Running()
	if !s.haveRunning || running != s.running {
		s.nextDue = now
	}
	s.haveRunning, s.running = true, running

	snap := s.d.Read()
	state := s.d.Store.Snapshot()
	fps, dirty := map[string]string{}, false
	for _, f := range snap.Files {
		fps[f.Path] = f.Fingerprint
		if !f.OK && s.reported[f.Path] != f.Fingerprint {
			// Once per rewrite: a file the client is still writing is read again next tick.
			s.reported[f.Path] = f.Fingerprint
			s.d.Log("cannot use %s: %s", f.Path, f.Problem)
		}
		if f.OK && state.Uploaded[f.Path] != f.Fingerprint {
			dirty = true
		}
	}
	if dirty && s.fails == 0 && !maps.Equal(fps, s.prevFP) {
		s.nextDue = now
	}
	s.prevFP = fps
	if now.Before(s.nextDue) {
		return
	}
	s.send(ctx, now, snap, state, dirty)
}

func (s *Syncer) send(ctx context.Context, now time.Time, snap savedvars.Snapshot, state config.State, dirty bool) {
	req := wire.SyncRequest{V: wire.Version, GameRunning: s.running}
	if dirty && len(snap.Characters) > 0 {
		req.Characters = snap.Characters
	}
	if s.fails >= 2 && !s.firstFail.IsZero() {
		req.Outage = &wire.Outage{From: s.firstFail.UTC().Format(timeFormat), To: now.UTC().Format(timeFormat)}
	}
	if h := s.d.RosterHash(); state.RosterEtag != "" && h != "" && h == state.RosterHash {
		req.RosterEtag = state.RosterEtag
	}
	resp, res := s.d.Hub.Sync(ctx, req)
	s.last = res.Outcome
	switch res.Outcome {
	case hubapi.OK:
		s.heard(now)
		s.markUploaded(snap, dirty)
		if resp.Roster != nil {
			if err := s.d.WriteRoster(*resp.Roster, now); err != nil {
				s.d.Log("could not write the member list: %v", err)
			} else if resp.RosterEtag != "" {
				h := s.d.RosterHash()
				s.save(func(st *config.State) { st.RosterEtag, st.RosterHash = resp.RosterEtag, h })
			}
		}
	case hubapi.Rejected:
		// The hub read the request and refused its body; sending the same files again would
		// be refused again.
		s.d.Log("the site refused a sync (status %d): %s", res.Status, res.Error)
		s.heard(now)
		s.markUploaded(snap, dirty)
	case hubapi.Throttled:
		s.nextDue = now.Add(ThrottleWait)
	case hubapi.Unlinked:
		s.revoked = true
		s.save(func(st *config.State) { st.Revoked = true })
		s.d.Log("the site no longer knows this PC; syncing stopped. Run the companion again to link it")
	default:
		s.failed(now, res)
	}
}

// heard is the bookkeeping of any answer that shows the hub was reached.
func (s *Syncer) heard(now time.Time) {
	if s.fails >= 2 {
		s.d.Log("the site is reachable again after %d failed requests", s.fails)
	}
	s.fails, s.firstFail = 0, time.Time{}
	s.nextDue = now.Add(every(s.running))
}

func (s *Syncer) failed(now time.Time, res hubapi.Result) {
	if now.Before(s.graceUntil) {
		s.nextDue = now.Add(RetryFast)
		return
	}
	s.fails++
	if s.fails == 1 {
		s.firstFail = now
	}
	if s.fails == 1 || s.fails == 2 {
		s.d.Log("the site is not reachable (status %d %s)", res.Status, res.Error)
	}
	if s.fails <= FastRetries {
		s.nextDue = now.Add(RetryFast)
	} else {
		// After five failures a retry is slower, but never slower than the normal pace.
		s.nextDue = now.Add(min(RetrySlow, every(s.running)))
	}
}

func (s *Syncer) markUploaded(snap savedvars.Snapshot, dirty bool) {
	if !dirty {
		return
	}
	s.save(func(st *config.State) {
		if st.Uploaded == nil {
			st.Uploaded = map[string]string{}
		}
		for _, f := range snap.Files {
			if f.OK {
				st.Uploaded[f.Path] = f.Fingerprint
			}
		}
	})
}

func (s *Syncer) save(f func(*config.State)) {
	if err := s.d.Store.Update(f); err != nil {
		s.d.Log("could not save state: %v", err)
	}
}

// Run ticks until the context ends, the loop is revoked or stop reports true. sleep waits on
// the caller's clock.
func (s *Syncer) Run(ctx context.Context, sleep func(context.Context, time.Duration) error, stop func() bool) error {
	for !s.revoked {
		if stop() {
			return nil
		}
		s.Tick(ctx)
		if err := sleep(ctx, TickEvery); err != nil {
			return err
		}
	}
	return nil
}
