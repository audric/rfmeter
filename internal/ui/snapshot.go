package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"

	"rfmeter/internal/meter"
	"rfmeter/internal/state"
)

// SnapshotData is everything needed to render a PNG offline.
type SnapshotData struct {
	Reading Reading
	Frames  []meter.Frame
	Stats   state.Stats
	Hist    *state.Histogram
	WindowS float64
	Page    byte
}

// snapshotFont is the Go Bold typeface, embedded in the binary. Using
// the same family the live UI renders with (gofont) keeps the PNG
// snapshot consistent with the on-screen cockpit, and embedding it
// means rendering never depends on system font files — the old code
// loaded DejaVu from hardcoded Linux paths and fell back to gg's tiny
// fixed bitmap font wherever those were absent (e.g. Windows).
var snapshotFont = mustParseFont(gobold.TTF)

func mustParseFont(ttf []byte) *truetype.Font {
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic("snapshot: parse embedded font: " + err.Error())
	}
	return f
}

// snapshotFace returns a scalable font face at the given point size.
func snapshotFace(size float64) font.Face {
	return truetype.NewFace(snapshotFont, &truetype.Options{Size: size})
}

func loadFont(dc *gg.Context, size float64) {
	dc.SetFontFace(snapshotFace(size))
}

// Snapshot renders an image and saves it to dir as
// rfmeter_snapshot_YYYYMMDD_HHMMSS.png. Returns the file path.
func Snapshot(dir string, d SnapshotData) (string, error) {
	const W, H = 1920, 1080
	dc := gg.NewContext(W, H)
	dc.SetRGB(0.05, 0.05, 0.05)
	dc.Clear()
	dc.SetRGB(0.9, 0.9, 0.9)

	loadFont(dc, 96)
	if d.Reading.Valid {
		dc.DrawStringAnchored(fmt.Sprintf("%+.1f dBm", d.Reading.DBm), W/2, 100, 0.5, 0.5)
		loadFont(dc, 56)
		dc.DrawStringAnchored(formatPower(d.Reading.WattsW), W/2, 200, 0.5, 0.5)
	}

	plotY0, plotY1 := 280, 720
	dc.SetRGB(0.06, 0.06, 0.06)
	dc.DrawRectangle(60, float64(plotY0), W-120, float64(plotY1-plotY0))
	dc.Fill()
	if len(d.Frames) > 0 {
		drawSeries(dc, d.Frames, 60, plotY0, W-120, plotY1-plotY0, d.WindowS)
	}

	histY0, histY1 := 760, 1000
	dc.SetRGB(0.06, 0.06, 0.06)
	dc.DrawRectangle(60, float64(histY0), W-120, float64(histY1-histY0))
	dc.Fill()
	if d.Hist != nil {
		drawHist(dc, d.Hist, d.Stats.MeanDBm, 60, histY0, W-120, histY1-histY0)
	}

	loadFont(dc, 28)
	dc.SetRGB(0.9, 0.9, 0.9)
	dc.DrawString(fmt.Sprintf(
		"min=%+.1f max=%+.1f mean=%+.1f  n=%d  page=%c  taken %s",
		d.Stats.MinDBm, d.Stats.MaxDBm, d.Stats.MeanDBm,
		d.Stats.N, d.Page, time.Now().Format(time.RFC3339)),
		80, 1050)

	name := fmt.Sprintf("rfmeter_snapshot_%s.png", time.Now().Format("20060102_150405"))
	path := filepath.Join(dir, name)
	if err := dc.SavePNG(path); err != nil {
		return "", err
	}
	return path, nil
}

func drawSeries(dc *gg.Context, frames []meter.Frame, x0, y0, w, h int, windowS float64) {
	tNow := frames[len(frames)-1].T
	tFirst := frames[0].T
	span := tNow.Sub(tFirst).Seconds()
	if span < 0.1 {
		span = 0.1
	}
	if span < windowS {
		windowS = span
	}
	tStart := tNow.Add(-time.Duration(windowS * float64(time.Second)))
	xSpan := tNow.Sub(tStart).Seconds()
	if xSpan <= 0 {
		return
	}
	ymin, ymax := math.Inf(1), math.Inf(-1)
	for _, f := range frames {
		if f.DBm < ymin {
			ymin = f.DBm
		}
		if f.DBm > ymax {
			ymax = f.DBm
		}
	}
	if ymax == ymin {
		ymax = ymin + 1
	}
	// Y-axis gridlines + labels (5 ticks).
	loadFont(dc, 20)
	dc.SetRGB(0.16, 0.16, 0.20)
	for i := 0; i <= 5; i++ {
		frac := float64(i) / 5.0
		py := float64(y0) + frac*float64(h)
		dc.DrawLine(float64(x0), py, float64(x0+w), py)
		dc.Stroke()
	}
	dc.SetRGB(0.7, 0.7, 0.75)
	for i := 0; i <= 5; i++ {
		frac := float64(i) / 5.0
		dbm := ymax - frac*(ymax-ymin)
		py := float64(y0) + frac*float64(h)
		dc.DrawStringAnchored(fmt.Sprintf("%.1f", dbm), float64(x0)-4, py, 1.0, 0.5)
	}
	// X-axis labels (5 ticks).
	for i := 0; i <= 4; i++ {
		frac := float64(i) / 4.0
		secsAgo := windowS * (1 - frac)
		px := float64(x0) + frac*float64(w)
		lbl := fmt.Sprintf("-%.0fs", secsAgo)
		if secsAgo < 0.05 {
			lbl = "now"
		}
		dc.DrawStringAnchored(lbl, px, float64(y0+h)+4, 0.5, 0.0)
	}
	// Polyline.
	dc.SetRGB(0.4, 1.0, 0.6)
	dc.SetLineWidth(2)
	first := true
	for _, f := range frames {
		if f.T.Before(tStart) {
			continue
		}
		fx := f.T.Sub(tStart).Seconds() / xSpan
		fy := (f.DBm - ymin) / (ymax - ymin)
		px := float64(x0) + fx*float64(w)
		py := float64(y0+h) - fy*float64(h)
		if first {
			dc.MoveTo(px, py)
			first = false
		} else {
			dc.LineTo(px, py)
		}
	}
	dc.Stroke()
}

func drawHist(dc *gg.Context, h *state.Histogram, meanDBm float64, x0, y0, w, hgt int) {
	maxLog := 0.0
	for _, c := range h.Bins {
		l := math.Log1p(float64(c))
		if l > maxLog {
			maxLog = l
		}
	}
	if maxLog > 0 {
		barW := float64(w) / float64(state.HistogramBins)
		dc.SetRGB(0.25, 0.6, 1.0)
		for i, c := range h.Bins {
			barH := math.Log1p(float64(c)) / maxLog * float64(hgt-4)
			dc.DrawRectangle(float64(x0)+float64(i)*barW, float64(y0+hgt)-barH, barW-1, barH)
			dc.Fill()
		}
	}
	if meanDBm >= -80 && meanDBm < 0 {
		mx := float64(x0) + (meanDBm+80)/80.0*float64(w)
		dc.SetRGB(1.0, 0.6, 0.25)
		dc.DrawRectangle(mx, float64(y0), 2, float64(hgt))
		dc.Fill()
	}
	// X-axis labels every 10 dB.
	loadFont(dc, 20)
	dc.SetRGB(0.7, 0.7, 0.75)
	const ticks = 8
	for i := 0; i <= ticks; i++ {
		dbm := -80.0 + 10.0*float64(i)
		frac := float64(i) / float64(ticks)
		px := float64(x0) + frac*float64(w)
		dc.DrawStringAnchored(fmt.Sprintf("%.0f", dbm), px, float64(y0+hgt)+4, 0.5, 0.0)
	}
}
