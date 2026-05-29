package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"

	"rfmeter/internal/state"
)

const (
	histMarginLeft   = 48
	histMarginRight  = 8
	histMarginTop    = 4
	histMarginBottom = 20
)

// HistogramView renders an 80-bin histogram of recent samples.
type HistogramView struct{}

// Layout draws the histogram. meanDBm is overlaid as a vertical line. th is
// used for axis labels (may be nil).
func (HistogramView) Layout(gtx layout.Context, th *material.Theme, h *state.Histogram, meanDBm float64) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X < 80 {
		size.X = 80
	}
	if size.Y < 20 {
		size.Y = 20
	}
	outerBg := color.NRGBA{R: 0x18, G: 0x18, B: 0x1c, A: 0xff}
	paint.FillShape(gtx.Ops, outerBg, clip.Rect{Max: size}.Op())

	canvas := image.Rect(histMarginLeft, histMarginTop, size.X-histMarginRight, size.Y-histMarginBottom)
	cw := canvas.Dx()
	ch := canvas.Dy()
	if cw < 10 || ch < 10 {
		return layout.Dimensions{Size: size}
	}

	bg := color.NRGBA{R: 0x05, G: 0x05, B: 0x0a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Min: canvas.Min, Max: canvas.Max}.Op())
	border := color.NRGBA{R: 0x60, G: 0x60, B: 0x70, A: 0xff}
	drawBorder(gtx, canvas, border)

	labelCol := color.NRGBA{R: 0xa0, G: 0xa0, B: 0xa8, A: 0xff}
	if h != nil {
		maxLog := 0.0
		for _, c := range h.Bins {
			l := math.Log1p(float64(c))
			if l > maxLog {
				maxLog = l
			}
		}
		if maxLog > 0 {
			barW := float64(cw) / float64(state.HistogramBins)
			bar := color.NRGBA{R: 0x40, G: 0xa0, B: 0xff, A: 0xff}
			for i, c := range h.Bins {
				hgt := math.Log1p(float64(c)) / maxLog * float64(ch-2)
				x0 := canvas.Min.X + int(float64(i)*barW)
				x1 := canvas.Min.X + int(float64(i+1)*barW) - 1
				if x1 < x0 {
					x1 = x0
				}
				y0 := canvas.Max.Y - int(hgt)
				rect := clip.Rect{Min: image.Pt(x0, y0), Max: image.Pt(x1, canvas.Max.Y)}
				paint.FillShape(gtx.Ops, bar, rect.Op())
			}
		}
		if meanDBm >= -80 && meanDBm < 0 {
			x := canvas.Min.X + int((meanDBm+80)/80.0*float64(cw))
			line := color.NRGBA{R: 0xff, G: 0xa0, B: 0x40, A: 0xff}
			rect := clip.Rect{Min: image.Pt(x, canvas.Min.Y), Max: image.Pt(x+2, canvas.Max.Y)}
			paint.FillShape(gtx.Ops, line, rect.Op())
		}
	}

	// X-axis labels every 10 dB: -80, -70, ..., 0.
	gridCol := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x32, A: 0xff}
	const ticks = 8 // 9 labels (0..8)
	for i := 0; i <= ticks; i++ {
		dbm := -80.0 + 10.0*float64(i)
		frac := float64(i) / float64(ticks)
		x := canvas.Min.X + int(frac*float64(cw-1))
		paint.FillShape(gtx.Ops, gridCol,
			clip.Rect{Min: image.Pt(x, canvas.Min.Y), Max: image.Pt(x+1, canvas.Max.Y)}.Op())
		drawLabel(gtx, th, fmt.Sprintf("%.0f", dbm), labelCol, image.Pt(x-10, canvas.Max.Y+2), 10)
	}
	return layout.Dimensions{Size: size}
}
