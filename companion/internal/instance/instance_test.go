package instance

import (
	"path/filepath"
	"testing"
)

func TestSecondAcquireFailsUntilRelease(t *testing.T) {
	p := filepath.Join(t.TempDir(), "run.lock")
	l, err := TryAcquire(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TryAcquire(p); err != ErrHeld {
		t.Fatalf("second acquire: %v", err)
	}
	l.Release()
	l2, err := TryAcquire(p)
	if err != nil {
		t.Fatal(err)
	}
	l2.Release()
}

func TestStopRequestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stop.request")
	if StopRequested(p) {
		t.Fatal("requested before asked")
	}
	if err := RequestStop(p); err != nil || !StopRequested(p) {
		t.Fatal(err)
	}
	ClearStop(p)
	if StopRequested(p) {
		t.Fatal("not cleared")
	}
}
