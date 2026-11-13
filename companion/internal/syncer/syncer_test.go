package syncer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/savedvars"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

type stubHub struct {
	reqs []wire.SyncRequest
	next func(n int, req wire.SyncRequest) (wire.SyncResponse, hubapi.Result)
}

func (h *stubHub) Sync(_ context.Context, req wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
	h.reqs = append(h.reqs, req)
	return h.next(len(h.reqs), req)
}

func ok(etag string, roster *wire.Roster) (wire.SyncResponse, hubapi.Result) {
	return wire.SyncResponse{OK: true, RosterEtag: etag, Roster: roster}, hubapi.Result{Outcome: hubapi.OK, Status: 200}
}

func res(o hubapi.Outcome, status int) (wire.SyncResponse, hubapi.Result) {
	return wire.SyncResponse{}, hubapi.Result{Outcome: o, Status: status}
}

type rig struct {
	s        *Syncer
	hub      *stubHub
	now      time.Time
	running  bool
	snap     savedvars.Snapshot
	store    *config.Store
	rosterOn bool
	written  []wire.Roster
	writeErr error
	logs     []string
}

func newRig(t *testing.T) *rig {
	r := &rig{hub: &stubHub{}, now: time.Date(2026, 11, 13, 19, 0, 0, 0, time.UTC)}
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return ok("", nil) }
	r.store, _ = config.Dir(t.TempDir()).OpenStore()
	r.s = New(Deps{Hub: r.hub, Now: func() time.Time { return r.now }, Running: func() bool { return r.running },
		Read: func() savedvars.Snapshot { return r.snap }, Store: r.store, RosterFileExists: func() bool { return r.rosterOn },
		WriteRoster: func(ro wire.Roster, _ time.Time) error {
			if r.writeErr != nil {
				return r.writeErr
			}
			r.written = append(r.written, ro)
			return nil
		}, Log: func(f string, a ...any) { r.logs = append(r.logs, f) }})
	return r
}

// advance moves the clock by d in ticks of TickEvery and returns how many requests went out.
func (r *rig) advance(d time.Duration) int {
	before := len(r.hub.reqs)
	for end := r.now.Add(d); r.now.Before(end); r.now = r.now.Add(TickEvery) {
		r.s.Tick(context.Background())
	}
	return len(r.hub.reqs) - before
}

func fileSnap(fp string, chars ...string) savedvars.Snapshot {
	s := savedvars.Snapshot{Files: []savedvars.File{{Path: "WTF/A.lua", Fingerprint: fp, OK: true}}}
	for _, c := range chars {
		s.Characters = append(s.Characters, wire.Character{Name: c, Realm: "R", Class: "MAGE", Skills: []wire.Skill{}})
	}
	return s
}

func TestScheduleRunning60AndIdle300(t *testing.T) {
	r := newRig(t)
	r.running = true
	if n := r.advance(10 * time.Minute); n != 10 { // at 0, 60, ... 540
		t.Fatalf("running: %d requests in 10 minutes", n)
	}
	r = newRig(t)
	if n := r.advance(10 * time.Minute); n != 2 { // at 0 and 300
		t.Fatalf("idle: %d requests in 10 minutes", n)
	}
	if r.hub.reqs[0].GameRunning {
		t.Fatal("running flag")
	}
}

func TestGameStartAndExitSyncAtOnce(t *testing.T) {
	r := newRig(t)
	r.advance(20 * time.Second)
	r.running = true
	if n := r.advance(TickEvery); n != 1 || !r.hub.reqs[len(r.hub.reqs)-1].GameRunning {
		t.Fatal("start not sent at once")
	}
	r.running = false
	if n := r.advance(TickEvery); n != 1 || r.hub.reqs[len(r.hub.reqs)-1].GameRunning {
		t.Fatal("exit not sent at once")
	}
}

func TestFirstSyncAlwaysCarriesCharactersThenOnlyOnChange(t *testing.T) {
	r := newRig(t)
	r.snap = fileSnap("1-1", "Thrall")
	r.advance(TickEvery)
	if len(r.hub.reqs[0].Characters) != 1 {
		t.Fatal("first sync without characters")
	}
	r.advance(2 * IdleEvery)
	for _, q := range r.hub.reqs[1:] {
		if len(q.Characters) != 0 {
			t.Fatal("resent unchanged characters")
		}
	}
	r.snap = fileSnap("2-2", "Thrall", "Bank")
	if n := r.advance(TickEvery); n != 1 || len(r.hub.reqs[len(r.hub.reqs)-1].Characters) != 2 {
		t.Fatal("changed file not uploaded at once")
	}
}

func TestUnparsableFileIsNotUploadedAndNotDirty(t *testing.T) {
	r := newRig(t)
	r.snap = savedvars.Snapshot{Files: []savedvars.File{{Path: "WTF/A.lua", Fingerprint: "1-1", OK: false, Problem: "cut off"}}}
	r.advance(TickEvery)
	if len(r.hub.reqs[0].Characters) != 0 || r.store.Snapshot().Uploaded["WTF/A.lua"] != "" {
		t.Fatal("partial file used")
	}
}

func TestOneFailureNoOutageTwoFailuresOutageOnce(t *testing.T) {
	r := newRig(t)
	r.running = true
	fail := 0
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if n >= 2 && n <= 1+fail {
			return res(hubapi.Unreachable, 503)
		}
		return ok("", nil)
	}
	fail = 1
	r.advance(5 * time.Minute)
	for _, q := range r.hub.reqs {
		if q.Outage != nil {
			t.Fatal("outage after one failure")
		}
	}
	r = newRig(t)
	r.running = true
	start := r.now
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if n >= 2 && n <= 4 {
			return res(hubapi.Unreachable, 0)
		}
		return ok("", nil)
	}
	r.advance(4 * time.Minute)
	var outages []*wire.Outage
	for i, q := range r.hub.reqs {
		if q.Outage != nil && !(i+1 >= 2 && i+1 <= 4) { // requests 2 to 4 failed
			outages = append(outages, q.Outage)
		}
	}
	if len(outages) != 1 {
		t.Fatalf("%d outage reports", len(outages))
	}
	from := start.Add(RunningEvery).UTC().Format(timeFormat)
	if outages[0].From != from {
		t.Fatalf("from %s want %s", outages[0].From, from)
	}
	b, _ := json.Marshal(outages[0])
	if len(b) == 0 || outages[0].To <= outages[0].From {
		t.Fatal("bad outage", string(b))
	}
}

func TestOutageKeptWhileRequestCarryingItFails(t *testing.T) {
	r := newRig(t)
	r.running = true
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if n >= 2 && n <= 4 {
			return res(hubapi.Unreachable, 500)
		}
		return ok("", nil)
	}
	r.advance(6 * time.Minute)
	var seen []string
	for _, q := range r.hub.reqs {
		if q.Outage != nil {
			seen = append(seen, q.Outage.From)
		}
	}
	if len(seen) != 2 || seen[0] != seen[1] {
		t.Fatalf("%v", seen)
	}
}

func TestRetry30ThenSlower(t *testing.T) {
	r := newRig(t)
	r.running = true
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if n == 1 {
			return ok("", nil)
		}
		return res(hubapi.Unreachable, 500)
	}
	r.advance(TickEvery) // request 1
	r.advance(RunningEvery)
	// failures begin at t=60 s; the next five retries are 30 s apart (rounded to ticks), then 120 s
	var times []time.Time
	r2 := newRig(t)
	r2.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		return res(hubapi.Unreachable, 500)
	}
	for i := 0; i < 200; i++ {
		before := len(r2.hub.reqs)
		r2.s.Tick(context.Background())
		if len(r2.hub.reqs) > before {
			times = append(times, r2.now)
		}
		r2.now = r2.now.Add(TickEvery)
	}
	for i := 1; i <= FastRetries && i < len(times); i++ {
		if d := times[i].Sub(times[i-1]); d != RetryFast {
			t.Fatalf("retry %d after %v", i, d)
		}
	}
	if d := times[FastRetries+1].Sub(times[FastRetries]); d != RetrySlow {
		t.Fatalf("slow retry after %v", d)
	}
}

func TestUnauthorizedRevokesAndPersists(t *testing.T) {
	r := newRig(t)
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return res(hubapi.Unlinked, 401) }
	r.advance(TickEvery)
	if r.s.Status() != Revoked || !r.store.Snapshot().Revoked {
		t.Fatal("not revoked")
	}
	if n := r.advance(time.Hour); n != 0 {
		t.Fatal("kept sending after 401")
	}
	if New(Deps{Store: r.store}).Status() != Revoked {
		t.Fatal("a new run forgot the revocation")
	}
}

func TestRejectedMarksUploadedAndKeepsSchedule(t *testing.T) {
	r := newRig(t)
	r.snap = fileSnap("1-1", "Thrall")
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return res(hubapi.Rejected, 400) }
	r.advance(2 * IdleEvery)
	if r.store.Snapshot().Uploaded["WTF/A.lua"] != "1-1" {
		t.Fatal("not marked uploaded")
	}
	if len(r.hub.reqs) != 2 || len(r.hub.reqs[1].Characters) != 0 {
		t.Fatalf("%d requests", len(r.hub.reqs))
	}
	if r.s.Status() != Healthy {
		t.Fatal("400 means the hub was reached")
	}
}

func TestThrottleWaits300AndIsNeitherFailureNorOutage(t *testing.T) {
	r := newRig(t)
	r.running = true
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if n <= 3 {
			return res(hubapi.Throttled, 429)
		}
		return ok("", nil)
	}
	if n := r.advance(ThrottleWait - TickEvery); n != 1 {
		t.Fatalf("%d requests inside the wait", n)
	}
	r.advance(15 * time.Minute)
	for _, q := range r.hub.reqs {
		if q.Outage != nil {
			t.Fatal("429 produced an outage")
		}
	}
	if r.s.Status() != Healthy {
		t.Fatal(r.s.Status())
	}
}

func TestRosterEtagAndWrite(t *testing.T) {
	r := newRig(t)
	roster := &wire.Roster{Members: []wire.RosterMember{{Name: "Anna"}}}
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if q.RosterEtag == "e1" {
			return ok("e1", nil)
		}
		return ok("e1", roster)
	}
	r.advance(TickEvery)
	if r.hub.reqs[0].RosterEtag != "" || len(r.written) != 1 || r.store.Snapshot().RosterEtag != "e1" {
		t.Fatal("first roster not stored")
	}
	r.advance(IdleEvery)
	if r.hub.reqs[1].RosterEtag != "" {
		t.Fatal("etag sent while Data.lua does not exist")
	}
	r.rosterOn = true
	r.advance(IdleEvery)
	if got := r.hub.reqs[len(r.hub.reqs)-1].RosterEtag; got != "e1" {
		t.Fatalf("etag %q", got)
	}
}

func TestFailedDataWriteKeepsOldEtag(t *testing.T) {
	r := newRig(t)
	r.writeErr = context.DeadlineExceeded
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		return ok("e2", &wire.Roster{})
	}
	r.advance(TickEvery)
	if r.store.Snapshot().RosterEtag != "" {
		t.Fatal("etag stored although the file was not written")
	}
}

func TestWakeFromSleepForgivesFailures(t *testing.T) {
	r := newRig(t)
	r.running = true
	r.advance(2 * TickEvery)
	r.now = r.now.Add(2 * time.Hour) // the PC slept
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return res(hubapi.Unreachable, 0) }
	r.advance(ResumeGrace)
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return ok("", nil) }
	r.advance(5 * time.Minute)
	for _, q := range r.hub.reqs {
		if q.Outage != nil {
			t.Fatal("outage from the first minute after a wake")
		}
	}
	if r.s.Status() != Healthy {
		t.Fatal(r.s.Status())
	}
}

func TestRunStopsOnRequest(t *testing.T) {
	r := newRig(t)
	ticks := 0
	err := r.s.Run(context.Background(), func(context.Context, time.Duration) error { r.now = r.now.Add(TickEvery); ticks++; return nil }, func() bool { return ticks >= 3 })
	if err != nil || ticks != 3 || len(r.hub.reqs) != 1 {
		t.Fatal(err, ticks, len(r.hub.reqs))
	}
}
