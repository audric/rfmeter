package meter

import "time"

// FrameLen is the fixed byte length of a streaming frame.
const FrameLen = 12

// Frame is a single power-meter reading.
type Frame struct {
	T       time.Time // host-side receive timestamp
	DBm     float64   // -99.9 .. +99.9
	LinearW float64   // power in watts
	Unit    byte      // 'u', 'm', 'w' as reported
	Raw     [FrameLen]byte
}

// ParseFrame parses a single 12-byte frame. Returns (frame, true) on success.
// The timestamp field is zero; caller fills it.
func ParseFrame(b []byte) (Frame, bool) {
	var f Frame
	if len(b) != FrameLen {
		return f, false
	}
	if b[0] != 'a' || b[11] != 'A' {
		return f, false
	}
	var sign float64
	switch b[1] {
	case '+':
		sign = +1
	case '-':
		sign = -1
	default:
		return f, false
	}
	for i := 2; i <= 9; i++ {
		if b[i] < '0' || b[i] > '9' {
			return f, false
		}
	}
	dbmTimes10 := int(b[2]-'0')*100 + int(b[3]-'0')*10 + int(b[4]-'0')
	linearTimes100 := int(b[5]-'0')*10000 + int(b[6]-'0')*1000 + int(b[7]-'0')*100 + int(b[8]-'0')*10 + int(b[9]-'0')
	var unitW float64
	switch b[10] {
	case 'u':
		unitW = 1e-6
	case 'm':
		unitW = 1e-3
	case 'w':
		unitW = 1.0
	default:
		return f, false
	}
	f.DBm = sign * float64(dbmTimes10) / 10.0
	f.LinearW = float64(linearTimes100) / 100.0 * unitW
	f.Unit = b[10]
	copy(f.Raw[:], b)
	return f, true
}

// Extract walks b, emitting every valid 12-byte frame found in order.
// Returns the unconsumed suffix (which may contain the start of an
// in-progress frame the caller should prepend to the next read).
func Extract(b []byte) (frames []Frame, tail []byte) {
	i := 0
	for i+FrameLen <= len(b) {
		if b[i] != 'a' {
			i++
			continue
		}
		f, ok := ParseFrame(b[i : i+FrameLen])
		if ok {
			frames = append(frames, f)
			i += FrameLen
			continue
		}
		i++
	}
	keep := len(b) - i
	if keep > FrameLen-1 {
		keep = FrameLen - 1
		i = len(b) - keep
	}
	tail = append(tail, b[i:]...)
	return frames, tail
}
