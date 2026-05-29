package state

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"rfmeter/internal/meter"
)

// CsvLog appends meter frames to a CSV file.
type CsvLog struct {
	mu   sync.Mutex
	f    *os.File
	w    *csv.Writer
	path string
	rows int64
}

// NewCsvLog returns an inactive log.
func NewCsvLog() *CsvLog { return &CsvLog{} }

// Start creates rfmeter_YYYYMMDD_HHMMSS.csv in dir.
// Idempotent: if already started, returns the existing path.
func (l *CsvLog) Start(dir string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		return l.path, nil
	}
	name := fmt.Sprintf("rfmeter_%s.csv", time.Now().Format("20060102_150405"))
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"time_iso", "dbm", "linear_w", "unit", "page"}); err != nil {
		_ = f.Close()
		return "", err
	}
	w.Flush()
	l.f = f
	l.w = w
	l.path = path
	l.rows = 0
	return path, nil
}

// Write appends one frame.
func (l *CsvLog) Write(fr meter.Frame, page byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w == nil {
		return nil
	}
	row := []string{
		fr.T.UTC().Format(time.RFC3339),
		strconv.FormatFloat(fr.DBm, 'f', -1, 64),
		strconv.FormatFloat(fr.LinearW, 'g', -1, 64),
		string(fr.Unit),
		string(page),
	}
	if err := l.w.Write(row); err != nil {
		return err
	}
	l.rows++
	return nil
}

// Stop flushes and closes.
func (l *CsvLog) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	l.w.Flush()
	err := l.f.Close()
	l.f = nil
	l.w = nil
	return err
}

// Status returns the current log state.
func (l *CsvLog) Status() (active bool, path string, rows int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f != nil, l.path, l.rows
}
