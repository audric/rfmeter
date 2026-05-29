package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rfmeter/internal/meter"
)

func TestCsvLog_WritesHeaderAndRows(t *testing.T) {
	dir := t.TempDir()
	l := NewCsvLog()
	path, err := l.Start(dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(path), "rfmeter_") || !strings.HasSuffix(path, ".csv") {
		t.Errorf("filename %q does not match pattern", path)
	}
	now := time.Date(2026, 5, 29, 10, 30, 0, 0, time.UTC)
	if err := l.Write(meter.Frame{T: now, DBm: -40.5, LinearW: 1e-7, Unit: 'u'}, 'A'); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := l.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "time_iso,dbm,linear_w,unit,page\n") {
		t.Errorf("missing header: %q", content)
	}
	if !strings.Contains(content, "2026-05-29T10:30:00Z,-40.5,1e-07,u,A\n") {
		t.Errorf("missing row: %q", content)
	}
}

func TestCsvLog_StopWithoutStart(t *testing.T) {
	l := NewCsvLog()
	if err := l.Stop(); err != nil {
		t.Errorf("Stop on never-started: %v, want nil", err)
	}
}
