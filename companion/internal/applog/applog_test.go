package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogRotatesOnce(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.log")
	l := New(p, 100, func() time.Time { return time.Unix(0, 0) })
	for i := 0; i < 20; i++ {
		l.Printf("line %d with some padding to fill the file", i)
	}
	cur, _ := os.ReadFile(p)
	old, err := os.ReadFile(p + ".1")
	if err != nil || len(cur) == 0 || len(cur) > 300 {
		t.Fatalf("current %d, old %v", len(cur), err)
	}
	if !strings.HasPrefix(string(old), "1970-01-01T00:00:00Z line") {
		t.Fatal(string(old))
	}
	if _, err := os.Stat(p + ".2"); err == nil {
		t.Fatal("more than one rotation")
	}
}
