package state

import "testing"

func TestHistogramCloneIsIndependent(t *testing.T) {
	h := NewHistogram()
	h.Add(-42)
	h.Add(-200) // underflow
	h.Add(5)    // overflow

	cp := h.Clone()

	// Mutating the original must not touch the clone.
	h.Add(-42)
	h.Add(-200)
	h.Add(5)

	if cp.Bins[int(-42+80)] != 1 {
		t.Errorf("clone bin changed with original: got %d, want 1", cp.Bins[int(-42+80)])
	}
	if cp.Underflow != 1 {
		t.Errorf("clone underflow changed: got %d, want 1", cp.Underflow)
	}
	if cp.Overflow != 1 {
		t.Errorf("clone overflow changed: got %d, want 1", cp.Overflow)
	}
}
