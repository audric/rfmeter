package meter

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func TestParseFrame_GoodFrames(t *testing.T) {
	cases := []struct {
		in    string
		dbm   float64
		watts float64
		unit  byte
	}{
		{"a-39200011uA", -39.2, 0.11e-6, 'u'},
		{"a+25838194mA", +25.8, 381.94e-3, 'm'},
		{"a-48400001uA", -48.4, 0.01e-6, 'u'},
		{"a+00000000uA", 0.0, 0.0, 'u'},
		{"a+99999999wA", +99.9, 999.99, 'w'},
	}
	for _, c := range cases {
		f, ok := ParseFrame([]byte(c.in))
		if !ok {
			t.Fatalf("ParseFrame(%q) = !ok, want ok", c.in)
		}
		if !almostEqual(f.DBm, c.dbm) {
			t.Errorf("%q DBm = %v, want %v", c.in, f.DBm, c.dbm)
		}
		if !almostEqual(f.LinearW, c.watts) {
			t.Errorf("%q LinearW = %v, want %v", c.in, f.LinearW, c.watts)
		}
		if f.Unit != c.unit {
			t.Errorf("%q Unit = %c, want %c", c.in, f.Unit, c.unit)
		}
	}
}

func TestParseFrame_Rejects(t *testing.T) {
	bad := []string{
		"",
		"a-39200011uX", // not 'A' tail
		"a-39200011xA", // bad unit
		"b-39200011uA", // not 'a' head
		"a*39200011uA", // bad sign
		"a-3920abcd1uA", // non-digit body
		"a-39200011uAa", // wrong length (13)
		"a-3920001uA",   // wrong length (11)
	}
	for _, s := range bad {
		if _, ok := ParseFrame([]byte(s)); ok {
			t.Errorf("ParseFrame(%q) = ok, want !ok", s)
		}
	}
}

func TestExtract_BackToBack(t *testing.T) {
	in := []byte("a-39200011uAa+25838194mA")
	frames, tail := Extract(in)
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if len(tail) != 0 {
		t.Errorf("tail = %q, want empty", tail)
	}
}

func TestExtract_GarbageBetween(t *testing.T) {
	// Real captured bytes: frames separated by echoed commands.
	in := []byte("a-46200002uAR0100+00.00200+00.00300+00.00400+00.0R\nVER\nver1000+00.02000+00.05000+00.06000+00.0Aa-43900004uA")
	frames, _ := Extract(in)
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 (first and last)", len(frames))
	}
	if frames[0].DBm != -46.2 {
		t.Errorf("frame 0 DBm = %v, want -46.2", frames[0].DBm)
	}
	if frames[1].DBm != -43.9 {
		t.Errorf("frame 1 DBm = %v, want -43.9", frames[1].DBm)
	}
}

func TestExtract_PartialAtEnd(t *testing.T) {
	in := []byte("a-39200011uAa+258")
	frames, tail := Extract(in)
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	if string(tail) != "a+258" {
		t.Errorf("tail = %q, want %q", tail, "a+258")
	}
}

func TestExtract_NoFrames(t *testing.T) {
	in := []byte("garbage no frames here")
	frames, tail := Extract(in)
	if len(frames) != 0 {
		t.Errorf("got %d frames, want 0", len(frames))
	}
	if len(tail) > len(in) {
		t.Errorf("tail longer than input")
	}
}
