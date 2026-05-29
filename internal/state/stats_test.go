package state

import (
	"math"
	"testing"
	"time"

	"rfmeter/internal/meter"
)

func framesAt(base time.Time, dbms []float64) []meter.Frame {
	out := make([]meter.Frame, len(dbms))
	for i, d := range dbms {
		out[i] = meter.Frame{T: base.Add(time.Duration(i) * time.Second), DBm: d}
	}
	return out
}

func TestComputeStats_Empty(t *testing.T) {
	s := ComputeStats(nil)
	if s.N != 0 {
		t.Errorf("N=%d, want 0", s.N)
	}
}

func TestComputeStats_MinMaxMean(t *testing.T) {
	now := time.Now()
	frames := framesAt(now, []float64{-10, -20, -30, -40})
	s := ComputeStats(frames)
	if s.N != 4 {
		t.Errorf("N=%d, want 4", s.N)
	}
	if s.MinDBm != -40 || s.MaxDBm != -10 {
		t.Errorf("min=%v max=%v, want -40 / -10", s.MinDBm, s.MaxDBm)
	}
	if math.Abs(s.MeanDBm-(-25)) > 1e-9 {
		t.Errorf("mean=%v, want -25", s.MeanDBm)
	}
}

func TestHistogram_Bins(t *testing.T) {
	h := NewHistogram()
	h.Add(-79.5) // bin 0
	h.Add(-0.5)  // bin 79
	h.Add(+1.0)  // overflow
	h.Add(-100)  // underflow
	h.Add(-40.0) // bin 40
	if h.Bins[0] != 1 || h.Bins[79] != 1 || h.Bins[40] != 1 {
		t.Errorf("bins wrong: 0=%d 40=%d 79=%d", h.Bins[0], h.Bins[40], h.Bins[79])
	}
	if h.Overflow != 1 || h.Underflow != 1 {
		t.Errorf("over=%d under=%d, want 1 / 1", h.Overflow, h.Underflow)
	}
}
