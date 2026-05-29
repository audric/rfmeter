package ui

import (
	"fmt"
	"math"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Reading is the live numeric state shown by Readout.
type Reading struct {
	DBm    float64
	WattsW float64
	Valid  bool
	// Context fields for the status line.
	Page         byte // 'A'..'H'
	FreqMHz      int  // 0 if unknown
	OffsetDB     float64
	OffsetKnown  bool
	Rate         Rate
}

// Readout is a stateless widget; its data lives in Reading.
type Readout struct {
	Th *material.Theme
}

// formatPower picks pW/nW/uW/mW/W based on magnitude.
func formatPower(w float64) string {
	if w <= 0 || math.IsNaN(w) {
		return "— W"
	}
	switch {
	case w < 1e-9:
		return fmt.Sprintf("%.2f pW", w*1e12)
	case w < 1e-6:
		return fmt.Sprintf("%.2f nW", w*1e9)
	case w < 1e-3:
		return fmt.Sprintf("%.2f µW", w*1e6)
	case w < 1.0:
		return fmt.Sprintf("%.2f mW", w*1e3)
	default:
		return fmt.Sprintf("%.2f W", w)
	}
}

// Layout draws the readout. Returns the natural (compact) size — caller uses
// layout.Rigid so the surrounding column gives the plot more vertical space.
func (rd *Readout) Layout(gtx layout.Context, r Reading) layout.Dimensions {
	if !r.Valid {
		return material.H3(rd.Th, "no signal").Layout(gtx)
	}
	w := gtx.Constraints.Max.X
	dbmPx := float32(w) / 14
	if dbmPx < 24 {
		dbmPx = 24
	}
	if dbmPx > 64 {
		dbmPx = 64
	}
	dbmLbl := material.H2(rd.Th, fmt.Sprintf("%+.1f dBm", r.DBm))
	dbmLbl.TextSize = unit.Sp(dbmPx)
	pwrLbl := material.H4(rd.Th, formatPower(r.WattsW))
	pwrLbl.TextSize = unit.Sp(dbmPx / 1.8)
	status := formatStatus(r)
	statusLbl := material.Body2(rd.Th, status)
	statusLbl.TextSize = unit.Sp(dbmPx / 3)
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// SpaceSides on a Horizontal flex with one Rigid child puts equal
		// leftover space on both sides — i.e. centers it horizontally.
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceSides, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(dbmLbl.Layout),
					layout.Rigid(pwrLbl.Layout),
					layout.Rigid(statusLbl.Layout),
				)
			}),
		)
	})
}

func formatStatus(r Reading) string {
	fkey := "F?"
	if r.Page >= 'A' && r.Page <= 'H' {
		fkey = fmt.Sprintf("F%d", int(r.Page-'A')+3)
	}
	freq := "— MHz"
	if r.FreqMHz > 0 {
		freq = fmt.Sprintf("%d MHz", r.FreqMHz)
	}
	off := "— dB"
	if r.OffsetKnown {
		off = fmt.Sprintf("%+.1f dB", r.OffsetDB)
	}
	return fmt.Sprintf("%s  ·  %s  ·  off %s  ·  %s", fkey, freq, off, r.Rate.Command())
}
