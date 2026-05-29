package ui

import (
	"sync/atomic"
	"testing"
	"time"
)

// F12 must not render the PNG on the UI goroutine: a full 1920x1080
// render + PNG encode + file write blocks the Gio event loop, freezing
// the window (especially on Windows) so the app looks hung. snapshot()
// must capture inputs synchronously then dispatch the heavy write to a
// background goroutine, and ignore overlapping presses (single-flight).
func TestSnapshotDispatchNonBlockingAndSingleFlight(t *testing.T) {
	a := New()

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var calls int32
	a.snapshotWriter = func(dir string, d SnapshotData) (string, error) {
		atomic.AddInt32(&calls, 1)
		started <- struct{}{}
		<-release // hold the "render" open
		return "snap.png", nil
	}

	// snapshot() itself must return promptly even though the write blocks.
	done := make(chan struct{})
	go func() { a.snapshot(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("snapshot() blocked on the render; must dispatch async")
	}

	<-started // first render is in flight

	// A second F12 while the first is still rendering must be ignored.
	a.snapshot()

	close(release) // let the first render finish

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !a.snapshotBusy.Load() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected single-flight (1 render), got %d", got)
	}
}
