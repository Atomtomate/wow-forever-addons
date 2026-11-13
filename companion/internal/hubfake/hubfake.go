// Package hubfake is an httptest hub that checks requests the way docs/CONTRACT.md §3.2 says.
// It builds the signing string itself and imports nothing from the code it tests.
package hubfake

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Device is one PC the fake hub knows.
type Device struct {
	ID      string
	Key     ed25519.PublicKey
	LastSeq uint64
	Linked  bool
	Member  string
	Code    string
}

// Hub is the fake.
type Hub struct {
	*httptest.Server
	mu sync.Mutex
	// Roster and Etag are what sync answers with.
	Roster string
	Etag   string
	// Override, when it returns true, answers instead of the normal handler (after the request
	// is counted).
	Override func(path string) (status int, body string, ok bool)
	// Interval is sent as "interval" on link; zero means 1.
	Interval int
	// Devices by id.
	Devices map[string]*Device
	// Syncs holds the decoded body of every sync that passed the signature checks.
	Syncs []map[string]any
	// Requests counts every request by path.
	Requests map[string]int
	nextID   int
}

// New starts a fake hub and stops it with the test.
func New(t testing.TB) *Hub {
	h := &Hub{Devices: map[string]*Device{}, Requests: map[string]int{}, Roster: `{"members":[]}`, Etag: "0000000000000001"}
	h.Server = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.Close)
	return h
}

// Approve links the pending device that shows code; it returns false when there is none.
func (h *Hub) Approve(code, member string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, d := range h.Devices {
		if d.Code == code && !d.Linked {
			d.Linked, d.Member = true, member
			return true
		}
	}
	return false
}

// Remove deletes a device, as the member's Remove on the profile does.
func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Devices, id)
}

// Count returns how many requests reached path.
func (h *Hub) Count(path string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Requests[path]
}

// SyncBodies returns a copy of the sync bodies received.
func (h *Hub) SyncBodies() []map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]map[string]any(nil), h.Syncs...)
}

// FirstDevice returns any one device, for tests with a single PC.
func (h *Hub) FirstDevice() *Device {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, d := range h.Devices {
		return d
	}
	return nil
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func (h *Hub) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Requests[r.URL.Path]++
	if h.Override != nil {
		if st, b, ok := h.Override(r.URL.Path); ok {
			if st >= 300 && st < 400 {
				w.Header().Set("Location", "/elsewhere")
			}
			reply(w, st, b)
			return
		}
	}
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/companion/") {
		reply(w, 404, `{"ok":false,"error":"not found"}`)
		return
	}
	if r.URL.Path == "/api/companion/link" {
		h.link(w, body)
		return
	}
	d := h.verify(w, r, body)
	if d == nil {
		return
	}
	switch r.URL.Path {
	case "/api/companion/link/status":
		if d.Linked {
			reply(w, 200, `{"ok":true,"linked":true,"member":`+quote(d.Member)+`}`)
		} else {
			reply(w, 200, `{"ok":true,"linked":false}`)
		}
	case "/api/companion/unlink":
		delete(h.Devices, d.ID)
		reply(w, 200, `{"ok":true}`)
	case "/api/companion/sync":
		h.sync(w, d, body)
	default:
		reply(w, 404, `{"ok":false,"error":"not found"}`)
	}
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }

func (h *Hub) link(w http.ResponseWriter, body []byte) {
	var in struct {
		V          int    `json:"v"`
		DeviceName string `json:"device_name"`
		PublicKey  string `json:"public_key"`
	}
	if json.Unmarshal(body, &in) != nil || in.V != 1 {
		reply(w, 400, `{"ok":false,"error":"bad body"}`)
		return
	}
	key, err := base64.StdEncoding.DecodeString(in.PublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		reply(w, 400, `{"ok":false,"error":"bad key"}`)
		return
	}
	for _, d := range h.Devices {
		if string(d.Key) == string(key) {
			reply(w, 400, `{"ok":false,"error":"key already known"}`)
			return
		}
	}
	h.nextID++
	id := "00000000000000" + strconv.Itoa(10+h.nextID)
	code := "BCDF-" + string("GHJKLMNPQRSTVWXZ"[h.nextID%16]) + "BCD"
	h.Devices[id] = &Device{ID: id, Key: key, Code: code}
	iv := h.Interval
	if iv == 0 {
		iv = 1
	}
	reply(w, 200, `{"ok":true,"device":"`+id+`","user_code":"`+code+`","expires_in":600,"interval":`+strconv.Itoa(iv)+`}`)
}

// verify runs the §3.2 checks in the contract's order and answers when one fails.
func (h *Hub) verify(w http.ResponseWriter, r *http.Request, body []byte) *Device {
	id, seqS, sigS := r.Header.Get("X-FF-Device"), r.Header.Get("X-FF-Seq"), r.Header.Get("X-FF-Signature")
	seq, err := strconv.ParseUint(seqS, 10, 64)
	bad := err != nil || seqS != strconv.FormatUint(seq, 10) || seq == 0 || len(seqS) > 15 || len(id) != 16
	d := h.Devices[id]
	sig, serr := base64.StdEncoding.DecodeString(sigS)
	if bad || d == nil || serr != nil || len(sig) != ed25519.SignatureSize {
		reply(w, 401, `{"ok":false,"error":"unauthorized"}`)
		return nil
	}
	sum := sha256.Sum256(body)
	msg := "FF1\n" + r.Method + "\n" + r.URL.Path + "\n" + id + "\n" + seqS + "\n" + hex.EncodeToString(sum[:])
	if !ed25519.Verify(d.Key, []byte(msg), sig) {
		reply(w, 401, `{"ok":false,"error":"unauthorized"}`)
		return nil
	}
	if seq <= d.LastSeq {
		reply(w, 409, `{"ok":false,"error":"stale sequence","seq":`+strconv.FormatUint(d.LastSeq, 10)+`}`)
		return nil
	}
	d.LastSeq = seq
	return d
}

func (h *Hub) sync(w http.ResponseWriter, d *Device, body []byte) {
	if !d.Linked {
		reply(w, 401, `{"ok":false,"error":"unauthorized"}`)
		return
	}
	var m map[string]any
	if json.Unmarshal(body, &m) != nil || m["v"] != float64(1) {
		reply(w, 400, `{"ok":false,"error":"unsupported version"}`)
		return
	}
	h.Syncs = append(h.Syncs, m)
	out := `{"ok":true,"member":` + quote(d.Member) + `,"server_time":"2026-11-13T20:00:00Z","roster_etag":"` + h.Etag + `"`
	if chars, ok := m["characters"].([]any); ok {
		out += `,"accepted":` + strconv.Itoa(len(chars)) + `,"ignored":[]`
	}
	if m["roster_etag"] != h.Etag {
		out += `,"roster":` + h.Roster
	}
	reply(w, 200, out+`}`)
}
