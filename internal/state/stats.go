package state

import "rfmeter/internal/meter"

// Stats are rolling-window aggregates.
type Stats struct {
	N              int
	MinDBm, MaxDBm float64
	MeanDBm        float64
}

// ComputeStats computes Stats over the provided frames (already window-filtered).
func ComputeStats(frames []meter.Frame) Stats {
	var s Stats
	if len(frames) == 0 {
		return s
	}
	s.MinDBm = frames[0].DBm
	s.MaxDBm = frames[0].DBm
	sum := 0.0
	for _, f := range frames {
		if f.DBm < s.MinDBm {
			s.MinDBm = f.DBm
		}
		if f.DBm > s.MaxDBm {
			s.MaxDBm = f.DBm
		}
		sum += f.DBm
	}
	s.N = len(frames)
	s.MeanDBm = sum / float64(s.N)
	return s
}

// HistogramBins covers -80..0 dBm in 1 dB steps.
const HistogramBins = 80

// Histogram counts samples per 1 dB bin plus under/over.
type Histogram struct {
	Bins      [HistogramBins]int
	Underflow int // dBm < -80
	Overflow  int // dBm >= 0
}

// NewHistogram returns an empty histogram.
func NewHistogram() *Histogram { return &Histogram{} }

// Add increments the bin for dbm.
func (h *Histogram) Add(dbm float64) {
	if dbm < -80 {
		h.Underflow++
		return
	}
	if dbm >= 0 {
		h.Overflow++
		return
	}
	bin := int(dbm + 80) // -80→0, -0.5→79, -79.5→0
	if bin < 0 {
		bin = 0
	}
	if bin >= HistogramBins {
		bin = HistogramBins - 1
	}
	h.Bins[bin]++
}

// Reset clears all bins.
func (h *Histogram) Reset() { *h = Histogram{} }
