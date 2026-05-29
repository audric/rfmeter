package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

const helpBody = `USB RF Power Meter V3

This window shows live power, a 60 s rolling plot, a histogram of
recent samples, and per-band calibration controls.

Band selection (F3-F10 or the band buttons) is host-side only — the
meter's active calibration is chosen by its physical S1/S2 buttons.

Keyboard shortcuts:
  F1     Help (this dialog)
  F2     Attenuator helper
  F3-F10 Select band
  F11    Toggle CSV log
  F12    PNG snapshot
  Esc    Close modal
  Space  Pause / resume plot

Press Esc to close.`

// HelpModal is drawn on top of the cockpit when shown.
func HelpModal(gtx layout.Context, th *material.Theme) layout.Dimensions {
	size := gtx.Constraints.Max
	// Dim the cockpit behind.
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0xc0}, clip.Rect{Max: size}.Op())
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return modalPanel(gtx, th, "Help", func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, helpBody).Layout(gtx)
		})
	})
}

// modalPanel draws a solid card with title + body inset.
func modalPanel(gtx layout.Context, th *material.Theme, title string, body layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(material.H4(th, title).Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(body),
		)
	})
	call := macro.Stop()
	// Draw the card background first, then replay the recorded content.
	panel := color.NRGBA{R: 0x28, G: 0x28, B: 0x32, A: 0xff}
	border := color.NRGBA{R: 0x70, G: 0x70, B: 0x80, A: 0xff}
	rect := clip.Rect{Max: dims.Size}
	paint.FillShape(gtx.Ops, panel, rect.Op())
	// Border (1 px on each side).
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(dims.Size.X, 1)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, dims.Size.Y-1), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(1, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(dims.Size.X-1, 0), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return dims
}
