package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"runtime"
	"testing"
)

func TestKeyRoundTripAndMode(t *testing.T) {
	d := Dir(t.TempDir())
	_, k, _ := ed25519.GenerateKey(rand.Reader)
	if err := d.SaveKey(k); err != nil {
		t.Fatal(err)
	}
	got, err := d.LoadKey()
	if err != nil || !k.Equal(got) {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		st, _ := os.Stat(d.Path(KeyFile))
		if st.Mode().Perm() != 0o600 {
			t.Fatal(st.Mode())
		}
	}
}

func TestConfigMissingIsEmptyAndRoundTrips(t *testing.T) {
	d := Dir(t.TempDir())
	c, err := d.Load()
	if err != nil || c.Hub != "" {
		t.Fatal(c, err)
	}
	c = Config{Hub: "https://h", Device: "0123456789abcdef", Pending: &Pending{Code: "HKRT-BCDF", ExpiresUnix: 5}}
	if err := d.Save(c); err != nil {
		t.Fatal(err)
	}
	got, _ := d.Load()
	if got.Hub != c.Hub || got.Pending == nil || got.Pending.Code != "HKRT-BCDF" {
		t.Fatal(got)
	}
}

func TestSeqMonotonicAcrossReopen(t *testing.T) {
	d := Dir(t.TempDir())
	s, _ := d.OpenStore()
	for want := uint64(1); want <= 3; want++ {
		if n, err := s.Reserve(); err != nil || n != want {
			t.Fatal(n, err)
		}
	}
	s2, _ := d.OpenStore()
	if n, _ := s2.Reserve(); n != 4 {
		t.Fatal(n)
	}
	_ = s2.Adopt(40)
	if n, _ := s2.Reserve(); n != 41 {
		t.Fatal(n)
	}
	_ = s2.Reset()
	if n, _ := s2.Reserve(); n != 1 {
		t.Fatal(n)
	}
}
