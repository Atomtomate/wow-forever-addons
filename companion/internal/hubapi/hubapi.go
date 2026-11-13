// Package hubapi is the only code that sends anything to the hub: it signs requests as
// docs/CONTRACT.md §3.2 says, spends sequence numbers, and reduces every answer to one Outcome.
package hubapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// Outcome is what a request came to, in the terms the contract's status table uses.
type Outcome int

// The outcomes of a request.
const (
	OK          Outcome = iota // 200 with "ok": true
	Rejected                   // 400 or 413: the hub read it and refused the body
	Unlinked                   // 401: the device is unknown, removed or not linked
	Throttled                  // 429: something in front of the hub asks for less
	Unreachable                // anything else, including a redirect or a malformed 200
)

// Result is the reduced answer of one request.
type Result struct {
	Outcome Outcome
	Status  int
	Error   string // the hub's one-sentence error, when it sent one
	Body    []byte
}

// SeqStore hands out sequence numbers and keeps them across restarts.
type SeqStore interface {
	// Reserve stores and returns the next number before the request that uses it is sent.
	Reserve() (uint64, error)
	// Adopt continues from a number the hub reported as the last it accepted.
	Adopt(last uint64) error
}

// Client talks to one hub as one device.
type Client struct {
	Origin    string
	Device    string
	Key       ed25519.PrivateKey
	Seq       SeqStore
	HTTP      *http.Client
	UserAgent string
}

// NewHTTPClient returns a client that never follows a redirect: the path a request is signed
// with must be the path the hub saw.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// ParseOrigin checks and normalises the hub's address: https, or http on a loopback host,
// with no user, path, query or fragment.
func ParseOrigin(s string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Host == "" || u.Opaque != "" {
		return "", errors.New("that is not an address like https://example.org")
	}
	if u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", errors.New("the address must be only the site's origin, like https://example.org")
	}
	switch u.Scheme {
	case "https":
	case "http":
		h := u.Hostname()
		if ip := net.ParseIP(h); h != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("http is only accepted for this computer; use https")
		}
	default:
		return "", errors.New("the address must start with https://")
	}
	return u.Scheme + "://" + u.Host, nil
}

// SigningString is the byte string §3.2 signs.
func SigningString(method, path, device string, seq uint64, body []byte) []byte {
	sum := sha256.Sum256(body)
	return []byte("FF1\n" + method + "\n" + path + "\n" + device + "\n" + strconv.FormatUint(seq, 10) + "\n" + hex.EncodeToString(sum[:]))
}

// Sign returns the base64 signature for a request.
func Sign(key ed25519.PrivateKey, method, path, device string, seq uint64, body []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(key, SigningString(method, path, device, seq, body)))
}

func (c *Client) do(ctx context.Context, path string, body []byte, seq uint64) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Origin+path, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.Key != nil {
		req.Header.Set("X-FF-Device", c.Device)
		req.Header.Set("X-FF-Seq", strconv.FormatUint(seq, 10))
		req.Header.Set("X-FF-Signature", Sign(c.Key, http.MethodPost, path, c.Device, seq, body))
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, wire.MaxAnswerBytes+1))
	if err != nil || len(b) > wire.MaxAnswerBytes {
		return Result{}, errors.New("answer unreadable or too large")
	}
	return reduce(resp.StatusCode, b), nil
}

func reduce(status int, body []byte) Result {
	r := Result{Status: status, Body: body, Outcome: Unreachable}
	var env struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	parsed := json.Unmarshal(body, &env) == nil
	if parsed {
		r.Error = env.Error
	}
	switch status {
	case 200:
		if parsed && env.OK {
			r.Outcome = OK
		}
	case 400, 413:
		r.Outcome = Rejected
	case 401:
		r.Outcome = Unlinked
	case 429:
		r.Outcome = Throttled
	}
	return r
}

// Post sends one signed request. It spends a new sequence number for each attempt; on 409 it
// continues from the number the hub names and sends once more, and a second 409 is Unreachable.
func (c *Client) Post(ctx context.Context, path string, payload any) Result {
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{Outcome: Unreachable, Error: err.Error()}
	}
	for attempt := 0; attempt < 2; attempt++ {
		seq, err := c.Seq.Reserve()
		if err != nil {
			return Result{Outcome: Unreachable, Error: err.Error()}
		}
		res, err := c.do(ctx, path, body, seq)
		if err != nil {
			return Result{Outcome: Unreachable, Error: err.Error()}
		}
		if res.Status != 409 {
			return res
		}
		var conflict struct {
			Seq uint64 `json:"seq"`
		}
		if json.Unmarshal(res.Body, &conflict) != nil || c.Seq.Adopt(conflict.Seq) != nil {
			break
		}
	}
	return Result{Outcome: Unreachable, Status: 409, Error: "stale sequence"}
}

// StartLink asks the hub for a short code; it is the one request that is not signed.
func StartLink(ctx context.Context, hc *http.Client, origin, userAgent, deviceName string, pub ed25519.PublicKey) (wire.LinkResponse, Result) {
	c := &Client{Origin: origin, HTTP: hc, UserAgent: userAgent}
	body, _ := json.Marshal(wire.LinkRequest{V: wire.Version, DeviceName: deviceName, PublicKey: base64.StdEncoding.EncodeToString(pub)})
	var out wire.LinkResponse
	res, err := c.do(ctx, "/api/companion/link", body, 0)
	if err != nil {
		return out, Result{Outcome: Unreachable, Error: err.Error()}
	}
	if res.Outcome == OK && (json.Unmarshal(res.Body, &out) != nil || out.Device == "" || out.UserCode == "") {
		res.Outcome = Unreachable
	}
	return out, res
}

// LinkStatus asks whether the member has approved this device yet.
func (c *Client) LinkStatus(ctx context.Context) (wire.StatusResponse, Result) {
	res := c.Post(ctx, "/api/companion/link/status", wire.VersionOnly{V: wire.Version})
	var out wire.StatusResponse
	if res.Outcome == OK && (json.Unmarshal(res.Body, &out) != nil || (out.Linked && out.Member == "")) {
		res.Outcome = Unreachable
	}
	return out, res
}

// Unlink removes this device on the hub; a 401 means it is already gone.
func (c *Client) Unlink(ctx context.Context) Result {
	return c.Post(ctx, "/api/companion/unlink", wire.VersionOnly{V: wire.Version})
}

// Sync uploads and fetches the member list. A 200 whose member list has the wrong shape is
// Unreachable; entries out of bounds are skipped (wire.ParseRoster).
func (c *Client) Sync(ctx context.Context, req wire.SyncRequest) (wire.SyncResponse, Result) {
	res := c.Post(ctx, "/api/companion/sync", req)
	var out wire.SyncResponse
	if res.Outcome != OK {
		return out, res
	}
	var raw struct {
		Member     string          `json:"member"`
		RosterEtag string          `json:"roster_etag"`
		Roster     json.RawMessage `json:"roster"`
	}
	if err := json.Unmarshal(res.Body, &raw); err != nil {
		res.Outcome = Unreachable
		return out, res
	}
	out = wire.SyncResponse{OK: true, Member: raw.Member, RosterEtag: raw.RosterEtag}
	if len(raw.Roster) > 0 && string(raw.Roster) != "null" {
		r, err := wire.ParseRoster(raw.Roster)
		if err != nil {
			res.Outcome, res.Error = Unreachable, fmt.Sprint(err)
			return wire.SyncResponse{}, res
		}
		out.Roster = &r
	}
	return out, res
}
