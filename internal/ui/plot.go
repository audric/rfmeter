package ui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"rfmeter/internal/meter"
)

// Plot renders dBm vs time for the last WindowSec seconds.
type Plot struct {
	WindowSec  float64 // visible window, default 60
	yMin, yMax float64
	yInit      bool
	Paused     bool
}

// NewPlot returns a Plot with sane defaults.
func NewPlot() *Plot { return &Plot{WindowSec: 60} }

// Layout draws the plot. Frames must be sorted by T ascending.
// th may be nil; if provided, the "PAUSED" overlay text is rendered when paused.
// margins reserved for axis labels.
const (
	plotMarginLeft   = 48
	plotMarginRight  = 8
	plotMarginTop    = 6
	plotMarginBottom = 20
)

func (p *Plot) Layout(gtx layout.Context, th *material.Theme, frames []meter.Frame) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X < 100 {
		size.X = 100
	}
	if size.Y < 50 {
		size.Y = 50
	}
	// Outer background fills the whole widget (axis-label gutters get app bg).
	outerBg := color.NRGBA{R: 0x18, G: 0x18, B: 0x1c, A: 0xff}
	paint.FillShape(gtx.Ops, outerBg, clip.Rect{Max: size}.Op())

	canvas := image.Rect(plotMarginLeft, plotMarginTop, size.X-plotMarginRight, size.Y-plotMarginBottom)
	cw := canvas.Dx()
	ch := canvas.Dy()
	if cw < 10 || ch < 10 {
		return layout.Dimensions{Size: size}
	}

	// Canvas background.
	bg := color.NRGBA{R: 0x05, G: 0x05, B: 0x0a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Min: canvas.Min, Max: canvas.Max}.Op())
	// Border.
	border := color.NRGBA{R: 0x60, G: 0x60, B: 0x70, A: 0xff}
	drawBorder(gtx, canvas, border)

	if len(frames) > 0 && !p.Paused {
		p.adjustYRange(frames)
	}
	if !p.yInit {
		p.yMin, p.yMax = -60, 0
		p.yInit = true
	}

	// Gridlines + Y-axis labels.
	gridCol := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x32, A: 0xff}
	labelCol := color.NRGBA{R: 0xa0, G: 0xa0, B: 0xa8, A: 0xff}
	const yTicks = 5
	for i := 0; i <= yTicks; i++ {
		frac := float64(i) / float64(yTicks)
		dbm := p.yMax - frac*(p.yMax-p.yMin)
		y := canvas.Min.Y + int(frac*float64(ch-1))
		paint.FillShape(gtx.Ops, gridCol, clip.Rect{
			Min: image.Pt(canvas.Min.X, y), Max: image.Pt(canvas.Max.X, y+1),
		}.Op())
		drawLabel(gtx, th, fmt.Sprintf("%.0f", dbm), labelCol, image.Pt(2, y-7), 11)
	}

	if len(frames) > 0 {
		// Adaptive window: if data spans < WindowSec, scale to actual span.
		tNow := frames[len(frames)-1].T
		tFirst := frames[0].T
		span := tNow.Sub(tFirst).Seconds()
		if span < 0.1 {
			span = 0.1
		}
		windowS := p.WindowSec
		if span < windowS {
			windowS = span
		}
		tStart := tNow.Add(-time.Duration(windowS * float64(time.Second)))
		xSpan := float64(tNow.Sub(tStart).Nanoseconds())
		if xSpan > 0 {
			type bucket struct {
				minY, maxY float64
				has        bool
			}
			buckets := make([]bucket, cw)
			for _, f := range frames {
				if f.T.Before(tStart) {
					continue
				}
				xn := float64(f.T.Sub(tStart).Nanoseconds()) / xSpan
				bi := int(xn * float64(cw-1))
				if bi < 0 || bi >= cw {
					continue
				}
				if !buckets[bi].has {
					buckets[bi] = bucket{minY: f.DBm, maxY: f.DBm, has: true}
					continue
				}
				if f.DBm < buckets[bi].minY {
					buckets[bi].minY = f.DBm
				}
				if f.DBm > buckets[bi].maxY {
					buckets[bi].maxY = f.DBm
				}
			}
			line := color.NRGBA{R: 0x60, G: 0xff, B: 0xa0, A: 0xff}
			for x := 0; x < cw; x++ {
				if !buckets[x].has {
					continue
				}
				y1 := dbmToY(buckets[x].minY, p.yMin, p.yMax, ch)
				y2 := dbmToY(buckets[x].maxY, p.yMin, p.yMax, ch)
				if y1 > y2 {
					y1, y2 = y2, y1
				}
				if y2-y1 < 3 {
					mid := (y1 + y2) / 2
					y1, y2 = mid-1, mid+2
				}
				xEnd := x + 2
				if xEnd > cw {
					xEnd = cw
				}
				rect := clip.Rect{
					Min: image.Pt(canvas.Min.X+x, canvas.Min.Y+y1),
					Max: image.Pt(canvas.Min.X+xEnd, canvas.Min.Y+y2+1),
				}
				paint.FillShape(gtx.Ops, line, rect.Op())
			}
			// X-axis labels.
			const xTicks = 4
			for i := 0; i <= xTicks; i++ {
				frac := float64(i) / float64(xTicks)
				secsAgo := windowS * (1 - frac)
				x := canvas.Min.X + int(frac*float64(cw-1))
				lbl := fmt.Sprintf("-%.0fs", secsAgo)
				if secsAgo < 0.05 {
					lbl = "now"
				}
				drawLabel(gtx, th, lbl, labelCol, image.Pt(x-12, canvas.Max.Y+2), 11)
			}
		}
	}

	if p.Paused {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x10},
			clip.Rect{Min: canvas.Min, Max: canvas.Max}.Op())
		if th != nil {
			layout.Center.Layout(gtx, material.H4(th, "PAUSED").Layout)
		}
	}
	return layout.Dimensions{Size: size}
}

// drawBorder draws a 1-px border around the rect.
func drawBorder(gtx layout.Context, r image.Rectangle, col color.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Min.X, r.Min.Y), Max: image.Pt(r.Max.X, r.Min.Y+1)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Min.X, r.Max.Y-1), Max: image.Pt(r.Max.X, r.Max.Y)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Min.X, r.Min.Y), Max: image.Pt(r.Min.X+1, r.Max.Y)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Max.X-1, r.Min.Y), Max: image.Pt(r.Max.X, r.Max.Y)}.Op())
}

// drawLabel renders a tiny text label at the given offset.
func drawLabel(gtx layout.Context, th *material.Theme, txt string, col color.NRGBA, off image.Point, sp float32) {
	if th == nil {
		return
	}
	defer op.Offset(off).Push(gtx.Ops).Pop()
	lbl := material.Label(th, unit.Sp(sp), txt)
	lbl.Color = col
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Pt(80, 16)
	lbl.Layout(gtx)
}

func dbmToY(dbm, ymin, ymax float64, h int) int {
	frac := (dbm - ymin) / (ymax - ymin)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	return h - int(frac*float64(h-1)) - 1
}

func (p *Plot) adjustYRange(frames []meter.Frame) {
	curMin, curMax := frames[0].DBm, frames[0].DBm
	for _, f := range frames {
		if f.DBm < curMin {
			curMin = f.DBm
		}
		if f.DBm > curMax {
			curMax = f.DBm
		}
	}
	curMin -= 1
	curMax += 1
	if !p.yInit {
		p.yMin, p.yMax, p.yInit = curMin, curMax, true
		return
	}
	margin := (p.yMax - p.yMin) * 0.1
	if curMin < p.yMin-margin || curMax > p.yMax+margin {
		p.yMin, p.yMax = curMin, curMax
	}
}
