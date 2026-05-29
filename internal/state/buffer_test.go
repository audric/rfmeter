package state

import (
	"testing"
	"time"

	"rfmeter/internal/meter"
)

func mkFrame(dbm float64, t time.Time) meter.Frame {
	return meter.Frame{T: t, DBm: dbm, LinearW: 0, Unit: 'u'}
}

func TestBuffer_Append_BelowCap(t *testing.T) {
	b := NewBuffer(4)
	now := time.Now()
	for i := 0; i < 3; i++ {
		b.Append(mkFrame(float64(-i), now.Add(time.Duration(i)*time.Second)))
	}
	snap := b.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("len=%d, want 3", len(snap))
	}
	if snap[0].DBm != 0 || snap[2].DBm != -2 {
		t.Errorf("snap order wrong: %+v", snap)
	}
}

func TestBuffer_Append_Overflow(t *testing.T) {
	b := NewBuffer(3)
	now := time.Now()
	for i := 0; i < 5; i++ {
		b.Append(mkFrame(float64(-i), now.Add(time.Duration(i)*time.Second)))
	}
	snap := b.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("len=%d, want 3 (ring kept only last 3)", len(snap))
	}
	if snap[0].DBm != -2 || snap[2].DBm != -4 {
		t.Errorf("ring order wrong: %+v", snap)
	}
}

func TestBuffer_Since(t *testing.T) {
	b := NewBuffer(10)
	now := time.Now()
	for i := 0; i < 10; i++ {
		b.Append(mkFrame(float64(i), now.Add(time.Duration(i)*time.Second)))
	}
	since := b.Since(now.Add(6 * time.Second))
	if len(since) != 4 {
		t.Fatalf("len(Since)=%d, want 4", len(since))
	}
	if since[0].DBm != 6 {
		t.Errorf("first since DBm = %v, want 6", since[0].DBm)
	}
}
