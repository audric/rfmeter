package ui

import (
	"fmt"
	"image/color"
	"strconv"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

)

// Attenuator holds the persistent state of the F2 modal.
type Attenuator struct {
	Th        *material.Theme
	SignalIn  widget.Editor
	MaxIn     widget.Editor
	MarginIn  widget.Editor
	ApplyBtn  widget.Clickable
	OnApply   func(attenuationDB float64)
}

// NewAttenuator returns a modal with defaults: max=0, margin=6.
func NewAttenuator(th *material.Theme) *Attenuator {
	a := &Attenuator{Th: th}
	a.SignalIn.SingleLine = true
	a.SignalIn.Filter = "0123456789+-."
	a.MaxIn.SingleLine = true
	a.MaxIn.Filter = "0123456789+-."
	a.MaxIn.SetText("0")
	a.MarginIn.SingleLine = true
	a.MarginIn.Filter = "0123456789."
	a.MarginIn.SetText("6")
	return a
}

// Compute returns (required-dB, applicable, message).
func (a *Attenuator) Compute() (recommended float64, ok bool, msg string) {
	sig, err1 := strconv.ParseFloat(a.SignalIn.Text(), 64)
	mx, err2 := strconv.ParseFloat(a.MaxIn.Text(), 64)
	mg, err3 := strconv.ParseFloat(a.MarginIn.Text(), 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false, "enter numeric values"
	}
	req := sig - mx + mg
	if req <= 0 {
		return req, false, fmt.Sprintf("no attenuator needed (signal already %.1f dB below max+margin)", -req)
	}
	return req, true, fmt.Sprintf("Need ≥ %.1f dB attenuation. After pad, meter sees ≈ %.1f dBm.", req, sig-req)
}

// Layout renders the modal. currentPage labels the Apply button.
func (a *Attenuator) Layout(gtx layout.Context, currentPage byte) layout.Dimensions {
	size := gtx.Constraints.Max
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0xc0}, clip.Rect{Max: size}.Op())

	for a.ApplyBtn.Clicked(gtx) {
		if rec, ok, _ := a.Compute(); ok && a.OnApply != nil {
			a.OnApply(rec)
		}
	}

	_, _, msg := a.Compute()
	fkey := "F?"
	if currentPage >= 'A' && currentPage <= 'H' {
		fkey = fmt.Sprintf("F%d", int(currentPage-'A')+3)
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return modalPanel(gtx, a.Th, "Attenuator helper", func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(a.row("Expected signal (dBm)", &a.SignalIn)),
				layout.Rigid(a.row("Meter max safe input (dBm)", &a.MaxIn)),
				layout.Rigid(a.row("Safety margin (dB)", &a.MarginIn)),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(material.Body1(a.Th, msg).Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(material.Button(a.Th, &a.ApplyBtn, fmt.Sprintf("Apply to %s", fkey)).Layout),
				layout.Rigid(material.Caption(a.Th, "Esc to close.").Layout),
			)
		})
	})
}

func (a *Attenuator) row(label string, e *widget.Editor) layout.Widget {
	e.MaxLen = 6 // e.g. "-50.0"
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(material.Body2(a.Th, label).Layout),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(90))
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				ed := material.Editor(a.Th, e, "0")
				ed.HintColor = color.NRGBA{R: 0x70, G: 0x70, B: 0x78, A: 0xff}
				return editorBox(gtx, ed.Layout)
			}),
		)
	}
}
