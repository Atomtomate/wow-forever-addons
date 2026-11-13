package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReplacesAndLeavesNoTemp(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "f")
	for _, want := range []string{"a", "b"} {
		if err := Write(p, []byte(want), 0o755, 0o644); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(p); string(b) != want {
			t.Fatal(string(b))
		}
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Fatal("temp left")
	}
}
