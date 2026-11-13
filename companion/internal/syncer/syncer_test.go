package syncer

import (
	"context"
	"strings"
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
	return wire.SyncResponse{RosterEtag: etag, Roster: roster}, hubapi.Result{Outcome: hubapi.OK, Status: 200}
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
	fileHash string // what is in Data.lua now; "" means no file
	written  []wire.Roster
	writeErr error
	logs     []string
}

func newRig(t *testing.T) *rig {
	r := &rig{hub: &stubHub{}, now: time.Date(2026, 11, 13, 19, 0, 0, 0, time.UTC)}
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return ok("", nil) }
	r.store, _ = config.Dir(t.TempDir()).OpenStore()
	r.s = New(Deps{Hub: r.hub, Now: func() time.Time { return r.now }, Running: func() bool { return r.running },
		Read: func() savedvars.Snapshot { return r.snap }, Store: r.store, RosterHash: func() string { return r.fileHash },
		WriteRoster: func(ro wire.Roster, _ time.Time) error {
			if r.writeErr != nil {
				return r.writeErr
			}
			r.written = append(r.written, ro)
			r.fileHash = "hash-of-" + ro.Members[0].Name
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
	if outages[0].To <= outages[0].From {
		t.Fatal("outage ends before it starts")
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

func TestRetry30ThenSlowerButNeverSlowerThanNormal(t *testing.T) {
	retries := func(running bool) []time.Duration {
		r := newRig(t)
		r.running = running
		r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return res(hubapi.Unreachable, 500) }
		var times []time.Time
		for i := 0; i < 300; i++ {
			before := len(r.hub.reqs)
			r.s.Tick(context.Background())
			if len(r.hub.reqs) > before {
				times = append(times, r.now)
			}
			r.now = r.now.Add(TickEvery)
		}
		var gaps []time.Duration
		for i := 1; i < len(times); i++ {
			gaps = append(gaps, times[i].Sub(times[i-1]))
		}
		return gaps
	}
	idle := retries(false)
	for i := 0; i < FastRetries; i++ {
		if idle[i] != RetryFast {
			t.Fatalf("retry %d after %v", i+1, idle[i])
		}
	}
	if idle[FastRetries] != RetrySlow {
		t.Fatalf("slow retry after %v", idle[FastRetries])
	}
	running := retries(true)
	if running[FastRetries] != RunningEvery {
		t.Fatalf("while the game runs the slow retry came after %v, not the normal %v", running[FastRetries], RunningEvery)
	}
}

func TestUnauthorizedRevokesAndPersists(t *testing.T) {
	r := newRig(t)
	r.hub.next = func(int, wire.SyncRequest) (wire.SyncResponse, hubapi.Result) { return res(hubapi.Unlinked, 401) }
	r.advance(TickEvery)
	if !r.s.Revoked() || !r.store.Snapshot().Revoked {
		t.Fatal("not revoked")
	}
	if n := r.advance(time.Hour); n != 0 {
		t.Fatal("kept sending after 401")
	}
	if !New(Deps{Store: r.store}).Revoked() {
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
	if r.s.LastOutcome() != hubapi.Rejected || r.s.fails != 0 {
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
	if r.s.LastOutcome() != hubapi.OK || r.s.fails != 0 {
		t.Fatal("not healthy after the throttle")
	}
}

func TestRosterEtagSentOnlyWhileTheFileHoldsWhatWeWrote(t *testing.T) {
	r := newRig(t)
	roster := &wire.Roster{Members: []wire.RosterMember{{Name: "Anna"}}}
	r.hub.next = func(n int, q wire.SyncRequest) (wire.SyncResponse, hubapi.Result) {
		if q.RosterEtag == "e1" {
			return ok("e1", nil)
		}
		return ok("e1", roster)
	}
	last := func() string { return r.hub.reqs[len(r.hub.reqs)-1].RosterEtag }
	r.advance(TickEvery)
	if r.hub.reqs[0].RosterEtag != "" || len(r.written) != 1 || r.store.Snapshot().RosterEtag != "e1" {
		t.Fatal("first roster not stored")
	}
	r.advance(IdleEvery)
	if last() != "e1" || len(r.written) != 1 {
		t.Fatalf("etag %q while the file is as written", last())
	}
	r.fileHash = "" // the file is gone (the game's folder was reset)
	r.advance(IdleEvery)
	if last() != "" || len(r.written) != 2 {
		t.Fatalf("etag %q sent without the file, %d writes", last(), len(r.written))
	}
	r.fileHash = "the empty first file of another game folder"
	r.advance(IdleEvery)
	if last() != "" {
		t.Fatalf("etag %q sent for a file we did not write", last())
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
	if r.s.LastOutcome() != hubapi.OK || r.s.fails != 0 {
		t.Fatal("not healthy after the wake")
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

func TestUnreadableFileIsLoggedOncePerRewrite(t *testing.T) {
	r := newRig(t)
	bad := func(fp string) savedvars.Snapshot {
		return savedvars.Snapshot{Files: []savedvars.File{{Path: "WTF/A.lua", Fingerprint: fp, Problem: "cut off"}}}
	}
	r.snap = bad("1-1")
	r.advance(3 * TickEvery)
	r.snap = bad("2-2")
	r.advance(3 * TickEvery)
	n := 0
	for _, l := range r.logs {
		if strings.HasPrefix(l, "cannot use") {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("%d lines for two rewrites: %v", n, r.logs)
	}
}
