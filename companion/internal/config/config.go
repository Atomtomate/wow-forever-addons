// Package config keeps what the companion remembers between runs, in one folder it is given:
// the link (config.json), the moving parts (state.json) and the private key (device.key).
package config

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/atomicfile"
)

// File names inside the folder.
const (
	ConfigFile = "config.json"
	StateFile  = "state.json"
	KeyFile    = "device.key"
	LogFile    = "companion.log"
	LockFile   = "run.lock"
	StopFile   = "stop.request"
)

// Dir is the config folder.
type Dir string

// Path returns the path of a file in the folder.
func (d Dir) Path(name string) string { return filepath.Join(string(d), name) }

// Pending is a link the member has not approved yet.
type Pending struct {
	Code        string `json:"code"`
	ExpiresUnix int64  `json:"expires"`
	IntervalSec int    `json:"interval"`
}

// defaultInterval is how often to ask when the hub named none.
const defaultInterval = 5 * time.Second

// Expires is when the hub forgets the link.
func (p Pending) Expires() time.Time { return time.Unix(p.ExpiresUnix, 0) }

// Interval is how often to ask whether the member has approved.
func (p Pending) Interval() time.Duration {
	if p.IntervalSec <= 0 {
		return defaultInterval
	}
	return time.Duration(p.IntervalSec) * time.Second
}

// Config is the link between this PC and a hub.
type Config struct {
	V          int      `json:"v"`
	Hub        string   `json:"hub"`
	Device     string   `json:"device"`
	DeviceName string   `json:"device_name"`
	Member     string   `json:"member,omitempty"`
	Game       string   `json:"game"`
	Pending    *Pending `json:"pending,omitempty"`
}

// State is what changes while the companion runs.
type State struct {
	V          int               `json:"v"`
	Seq        uint64            `json:"seq"`
	Uploaded   map[string]string `json:"uploaded,omitempty"`
	RosterEtag string            `json:"roster_etag,omitempty"`
	// RosterHash is the hash of the Data.lua the companion wrote when it stored RosterEtag; the
	// etag is sent only while the file still has it.
	RosterHash string `json:"roster_hash,omitempty"`
	Revoked    bool   `json:"revoked,omitempty"`
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return atomicfile.Write(path, b, 0o700, 0o600)
}

// Load reads config.json; a missing file is an empty Config.
func (d Dir) Load() (Config, error) {
	var c Config
	err := readJSON(d.Path(ConfigFile), &c)
	return c, err
}

// Save writes config.json.
func (d Dir) Save(c Config) error { c.V = 1; return writeJSON(d.Path(ConfigFile), c) }

// SaveKey stores the private key as PKCS8 PEM, readable by this user only.
func (d Dir) SaveKey(k ed25519.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return err
	}
	return atomicfile.Write(d.Path(KeyFile), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o700, 0o600)
}

// LoadKey reads the private key.
func (d Dir) LoadKey() (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(d.Path(KeyFile))
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, errors.New("device.key is not a PEM file")
	}
	k, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	ek, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("device.key is not an Ed25519 key")
	}
	return ek, nil
}

// Store is state.json held in memory and written through on every change. It is the
// hubapi.SeqStore.
type Store struct {
	dir Dir
	mu  sync.Mutex
	s   State
}

// OpenStore reads state.json; a missing file is an empty State.
func (d Dir) OpenStore() (*Store, error) {
	st := &Store{dir: d}
	if err := readJSON(d.Path(StateFile), &st.s); err != nil {
		return nil, err
	}
	return st, nil
}

// Snapshot returns a copy of the state.
func (s *Store) Snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.s
	c.Uploaded = maps.Clone(s.s.Uploaded)
	if c.Uploaded == nil {
		c.Uploaded = map[string]string{}
	}
	return c
}

// Update changes the state and writes it.
func (s *Store) Update(f func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(&s.s)
	s.s.V = 1
	return writeJSON(s.dir.Path(StateFile), s.s)
}

// Reserve implements hubapi.SeqStore: the next number is on disk before it is used.
func (s *Store) Reserve() (uint64, error) {
	var n uint64
	err := s.Update(func(st *State) { st.Seq++; n = st.Seq })
	return n, err
}

// Adopt implements hubapi.SeqStore.
func (s *Store) Adopt(last uint64) error { return s.Update(func(st *State) { st.Seq = last }) }

// Reset starts the state of a new link: a new device has its own sequence and nothing uploaded.
func (s *Store) Reset() error { return s.Update(func(st *State) { *st = State{} }) }
