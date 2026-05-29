package meter

import "testing"

func TestParseConfigReply_RealBytes(t *testing.T) {
	in := []byte("a-45800002uAR0100+00.00200+00.00300+00.00400+00.0R\nVER\nver1000+00.02000+00.05000+00.06000+00.0Aa-43900004uA")
	pages := ParseConfigReply(in)
	want := []Page{
		{Letter: 'A', FreqMHz: 100, OffsetDB: 0.0},
		{Letter: 'B', FreqMHz: 200, OffsetDB: 0.0},
		{Letter: 'C', FreqMHz: 300, OffsetDB: 0.0},
		{Letter: 'D', FreqMHz: 400, OffsetDB: 0.0},
		{Letter: 'E', FreqMHz: 1000, OffsetDB: 0.0},
		{Letter: 'F', FreqMHz: 2000, OffsetDB: 0.0},
		{Letter: 'G', FreqMHz: 5000, OffsetDB: 0.0},
		{Letter: 'H', FreqMHz: 6000, OffsetDB: 0.0},
	}
	if len(pages) != len(want) {
		t.Fatalf("got %d pages, want %d: %+v", len(pages), len(want), pages)
	}
	for i, p := range pages {
		if p != want[i] {
			t.Errorf("page %d = %+v, want %+v", i, p, want[i])
		}
	}
}

func TestBuildSetPageCmd(t *testing.T) {
	cases := []struct {
		letter byte
		freq   int
		offset float64
		want   string
	}{
		{'A', 2400, 10.0, "A2400+10.0"},
		{'B', 100, -3.5, "B0100-03.5"},
		{'H', 9999, 0.0, "H9999+00.0"},
	}
	for _, c := range cases {
		got := BuildSetPageCmd(c.letter, c.freq, c.offset)
		if got != c.want {
			t.Errorf("BuildSetPageCmd(%c, %d, %v) = %q, want %q", c.letter, c.freq, c.offset, got, c.want)
		}
	}
}
