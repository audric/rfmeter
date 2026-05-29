package state

import (
	"sync"
	"time"

	"rfmeter/internal/meter"
)

// Buffer is a thread-safe ring of recent Frames.
type Buffer struct {
	mu   sync.RWMutex
	cap  int
	data []meter.Frame
	head int  // next write index
	full bool // wrapped around at least once
}

// NewBuffer constructs a ring with the given capacity.
func NewBuffer(capacity int) *Buffer {
	if capacity < 1 {
		capacity = 1
	}
	return &Buffer{cap: capacity, data: make([]meter.Frame, capacity)}
}

// Append adds f, overwriting the oldest entry once full.
func (b *Buffer) Append(f meter.Frame) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[b.head] = f
	b.head = (b.head + 1) % b.cap
	if b.head == 0 {
		b.full = true
	}
}

// Snapshot returns a chronologically-ordered copy of all current entries.
func (b *Buffer) Snapshot() []meter.Frame {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !b.full {
		out := make([]meter.Frame, b.head)
		copy(out, b.data[:b.head])
		return out
	}
	out := make([]meter.Frame, b.cap)
	copy(out, b.data[b.head:])
	copy(out[b.cap-b.head:], b.data[:b.head])
	return out
}

// Since returns frames with T >= cutoff, in chronological order.
func (b *Buffer) Since(cutoff time.Time) []meter.Frame {
	all := b.Snapshot()
	i := len(all)
	for i > 0 && !all[i-1].T.Before(cutoff) {
		i--
	}
	return all[i:]
}
