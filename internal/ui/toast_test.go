package ui

import (
	"strings"
	"testing"
	"time"
)

func TestToastTextVisibility(t *testing.T) {
	a := New()

	// No toast set yet.
	if _, ok := a.toastText(time.Now()); ok {
		t.Fatal("expected no toast initially")
	}

	now := time.Now()
	// Sticky toast (zero expiry) shows regardless of time.
	a.toast.Store(&toastMsg{text: "Saving snapshot…"})
	if txt, ok := a.toastText(now.Add(time.Hour)); !ok || txt != "Saving snapshot…" {
		t.Fatalf("sticky toast should always show: got %q ok=%v", txt, ok)
	}

	// Expiring toast: visible before, hidden after.
	a.toast.Store(&toastMsg{text: "Saved x.png", until: now.Add(3 * time.Second)})
	if _, ok := a.toastText(now.Add(time.Second)); !ok {
		t.Fatal("toast should be visible before expiry")
	}
	if _, ok := a.toastText(now.Add(4 * time.Second)); ok {
		t.Fatal("toast should be hidden after expiry")
	}
}

// A snapshot should announce itself: a sticky "saving" toast while the
// render runs, then a transient "saved <file>" toast on success.
func TestSnapshotUpdatesToast(t *testing.T) {
	a := New()
	release := make(chan struct{})
	a.snapshotWriter = func(dir string, d SnapshotData) (string, error) {
		<-release
		return "/tmp/rfmeter_snapshot_20260529_180000.png", nil
	}

	a.snapshot()
	if txt, ok := a.toastText(time.Now()); !ok || !strings.Contains(strings.ToLower(txt), "saving") {
		t.Fatalf("expected a 'saving' toast during render, got %q ok=%v", txt, ok)
	}

	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && a.snapshotBusy.Load() {
		time.Sleep(5 * time.Millisecond)
	}

	txt, ok := a.toastText(time.Now())
	if !ok || !strings.Contains(txt, "rfmeter_snapshot_20260529_180000.png") {
		t.Fatalf("expected a 'saved <file>' toast, got %q ok=%v", txt, ok)
	}
}
