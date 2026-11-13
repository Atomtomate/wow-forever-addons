package hubapi_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubfake"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

type memSeq struct{ n uint64 }

func (m *memSeq) Reserve() (uint64, error) { m.n++; return m.n, nil }
func (m *memSeq) Adopt(l uint64) error     { m.n = l; return nil }

func seedKey() ed25519.PrivateKey {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	return ed25519.NewKeyFromSeed(seed)
}

func TestSignVector(t *testing.T) {
	k := seedKey()
	if got := base64.StdEncoding.EncodeToString(k.Public().(ed25519.PublicKey)); got != "A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg=" {
		t.Fatal(got)
	}
	sig := hubapi.Sign(k, "POST", "/api/companion/sync", "0123456789abcdef", 7, []byte(`{"v":1,"game_running":false}`))
	want := "/ugHnkaZUd3KLobU7MuwmJfVZHMDdseFQ0WO5IjfJh0xvHvEWIB8dSWXhpVvPBcxaPCGbtg1srH1wV8UNAsUCw=="
	if sig != want {
		t.Fatalf("signature %s", sig)
	}
}

func linked(t *testing.T, h *hubfake.Hub) *hubapi.Client {
	t.Helper()
	k := seedKey()
	out, res := hubapi.StartLink(context.Background(), hubapi.NewHTTPClient(time.Second), h.URL, "t", "pc", k.Public().(ed25519.PublicKey))
	if res.Outcome != hubapi.OK {
		t.Fatal(res)
	}
	h.Approve(out.UserCode, "Anna")
	return &hubapi.Client{Origin: h.URL, Device: out.Device, Key: k, Seq: &memSeq{}, HTTP: hubapi.NewHTTPClient(time.Second)}
}

func TestStatusTable(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   hubapi.Outcome
	}{
		{200, `{"ok":true,"member":"Anna"}`, hubapi.OK},
		{200, `not json`, hubapi.Unreachable},
		{200, `{"ok":false}`, hubapi.Unreachable},
		{400, `{"ok":false,"error":"x"}`, hubapi.Rejected},
		{413, `{"ok":false}`, hubapi.Rejected},
		{401, `{"ok":false}`, hubapi.Unlinked},
		{429, ``, hubapi.Throttled},
		{500, `{"ok":false}`, hubapi.Unreachable},
		{404, ``, hubapi.Unreachable},
	}
	for _, c := range cases {
		h := hubfake.New(t)
		cl := linked(t, h)
		h.Override = func(string) (int, string, bool) { return c.status, c.body, true }
		_, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1})
		if res.Outcome != c.want {
			t.Errorf("%d %q: %v want %v", c.status, c.body, res.Outcome, c.want)
		}
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	for _, code := range []int{302, 307} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, code)
		}))
		cl := &hubapi.Client{Origin: srv.URL, Device: "0123456789abcdef", Key: seedKey(), Seq: &memSeq{}, HTTP: hubapi.NewHTTPClient(time.Second)}
		_, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1})
		srv.Close()
		if res.Outcome != hubapi.Unreachable || hits.Load() != 0 {
			t.Fatalf("%d: %v hits %d", code, res.Outcome, hits.Load())
		}
	}
}

func TestConflictResendsOnceThenFails(t *testing.T) {
	h := hubfake.New(t)
	cl := linked(t, h)
	cl.Seq = &memSeq{} // behind the hub: first answer is 409
	h.FirstDevice().LastSeq = 40
	_, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1})
	if res.Outcome != hubapi.OK || h.Count("/api/companion/sync") != 2 {
		t.Fatalf("%v, %d requests", res.Outcome, h.Count("/api/companion/sync"))
	}
	// a hub that always says 409 gets exactly two tries
	h2 := hubfake.New(t)
	cl2 := linked(t, h2)
	h2.Override = func(string) (int, string, bool) { return 409, `{"ok":false,"seq":3}`, true }
	_, res = cl2.Sync(context.Background(), wire.SyncRequest{V: 1})
	if res.Outcome != hubapi.Unreachable || h2.Count("/api/companion/sync") != 2 {
		t.Fatalf("%v, %d requests", res.Outcome, h2.Count("/api/companion/sync"))
	}
}

func TestSeqStoredBeforeSend(t *testing.T) {
	seq := &memSeq{}
	cl := &hubapi.Client{Origin: "http://127.0.0.1:1", Device: "0123456789abcdef", Key: seedKey(), Seq: seq, HTTP: hubapi.NewHTTPClient(time.Second)}
	_, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1})
	if res.Outcome != hubapi.Unreachable || seq.n != 1 {
		t.Fatalf("%v seq %d", res.Outcome, seq.n)
	}
}

func TestSyncParsesRosterAndShape(t *testing.T) {
	h := hubfake.New(t)
	cl := linked(t, h)
	h.Roster = `{"members":[{"name":"Anna","characters":[{"name":"T","class":"Shaman","level":23,"main":true}]}]}`
	out, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1})
	if res.Outcome != hubapi.OK || out.Roster == nil || out.Roster.Members[0].Characters[0].Level != 23 || out.Member != "Anna" {
		t.Fatalf("%+v %v", out, res.Outcome)
	}
	h.Roster = `{"members":"nope"}`
	if _, res = cl.Sync(context.Background(), wire.SyncRequest{V: 1}); res.Outcome != hubapi.Unreachable {
		t.Fatal(res.Outcome)
	}
}

func TestUnlinkedAfterRemove(t *testing.T) {
	h := hubfake.New(t)
	cl := linked(t, h)
	h.Remove(cl.Device)
	if _, res := cl.Sync(context.Background(), wire.SyncRequest{V: 1}); res.Outcome != hubapi.Unlinked {
		t.Fatal(res.Outcome)
	}
}

func TestStartLinkRefusals(t *testing.T) {
	h := hubfake.New(t)
	pub := seedKey().Public().(ed25519.PublicKey)
	if _, res := hubapi.StartLink(context.Background(), hubapi.NewHTTPClient(time.Second), h.URL, "", "pc", pub); res.Outcome != hubapi.OK {
		t.Fatal(res)
	}
	if _, res := hubapi.StartLink(context.Background(), hubapi.NewHTTPClient(time.Second), h.URL, "", "pc", pub); res.Outcome != hubapi.Rejected {
		t.Fatal("known key accepted", res.Outcome)
	}
	h.Override = func(string) (int, string, bool) { return 429, `{"ok":false}`, true }
	if _, res := hubapi.StartLink(context.Background(), hubapi.NewHTTPClient(time.Second), h.URL, "", "pc", pub); res.Outcome != hubapi.Throttled {
		t.Fatal(res.Outcome)
	}
}

func TestOriginRules(t *testing.T) {
	good := map[string]string{"https://hub.example": "https://hub.example", "https://hub.example:8443/": "https://hub.example:8443", "http://127.0.0.1:3000": "http://127.0.0.1:3000", "http://localhost": "http://localhost", "http://[::1]:80": "http://[::1]:80"}
	for in, want := range good {
		if got, err := hubapi.ParseOrigin(in); err != nil || got != want {
			t.Errorf("%s: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", "hub.example", "http://hub.example", "ftp://x", "https://u:p@hub.example", "https://hub.example/path", "https://hub.example/?q=1", "https://hub.example/#f", "https://hub.example?"} {
		if got, err := hubapi.ParseOrigin(in); err == nil {
			t.Errorf("%q accepted as %q", in, got)
		}
	}
}
