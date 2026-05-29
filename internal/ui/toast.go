package ui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// toastMsg is a transient status pill. A zero until means it stays
// visible indefinitely (e.g. while a snapshot is still rendering).
type toastMsg struct {
	text  string
	until time.Time
}

// toastText returns the message to show now, or ok=false if none.
func (a *App) toastText(now time.Time) (string, bool) {
	ts := a.toast.Load()
	if ts == nil {
		return "", false
	}
	if !ts.until.IsZero() && now.After(ts.until) {
		return "", false
	}
	return ts.text, true
}

// drawToast paints the current toast (if any) near the bottom-centre.
func (a *App) drawToast(gtx layout.Context, th *material.Theme) {
	text, ok := a.toastText(time.Now())
	if !ok {
		return
	}
	layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			macro := op.Record(gtx.Ops)
			lbl := material.Body1(th, text)
			lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			dims := layout.UniformInset(unit.Dp(12)).Layout(gtx, lbl.Layout)
			call := macro.Stop()
			// Background pill, recorded-then-replayed so it sits behind the text.
			r := gtx.Dp(unit.Dp(8))
			bg := color.NRGBA{R: 0x28, G: 0x28, B: 0x32, A: 0xf0}
			rr := clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SW: r, SE: r}
			paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
			call.Add(gtx.Ops)
			return dims
		})
	})
}
