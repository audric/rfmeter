package ui

import "testing"

// snapshotFace must return a scalable embedded font whose metrics grow
// with the requested size. The old code loaded fonts from hardcoded
// Linux paths and, when they were absent (e.g. on Windows), silently
// fell back to gg's fixed 13px bitmap font — which ignores the size
// argument and renders tiny, wrong-typeface text in the PNG snapshot.
func TestSnapshotFaceScalesWithSize(t *testing.T) {
	small := snapshotFace(20)
	big := snapshotFace(96)
	if small == nil || big == nil {
		t.Fatal("snapshotFace returned nil")
	}
	hSmall := small.Metrics().Height
	hBig := big.Metrics().Height
	if hBig <= hSmall {
		t.Fatalf("font does not scale with size: height(20)=%v height(96)=%v "+
			"(likely the fixed bitmap fallback)", hSmall, hBig)
	}
}
