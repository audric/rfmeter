package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// menuBarHeight is the fixed height of the top menu strip.
const menuBarHeight = unit.Dp(32)

// Menu is the top menu bar with a dropdown (About, Exit). It is drawn by
// Gio itself — Gio has no native OS menu bar — so it looks and behaves
// identically on Linux and Windows.
type Menu struct {
	OnAbout func()
	OnExit  func()

	open    bool
	menuBtn widget.Clickable
	scrim   widget.Clickable // full-window catcher to dismiss on outside click
	about   widget.Clickable
	exit    widget.Clickable
}

func (m *Menu) toggle() { m.open = !m.open }
func (m *Menu) close()  { m.open = false }

func (m *Menu) selectAbout() {
	m.open = false
	if m.OnAbout != nil {
		m.OnAbout()
	}
}

func (m *Menu) selectExit() {
	m.open = false
	if m.OnExit != nil {
		m.OnExit()
	}
}

// Bar draws the top menu strip and returns its dimensions.
func (m *Menu) Bar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if m.menuBtn.Clicked(gtx) {
		m.toggle()
	}
	h := gtx.Dp(menuBarHeight)
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = h, h
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x22, G: 0x22, B: 0x2a, A: 0xff},
		clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
	return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return m.menuBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, "Menu  ▾")
				lbl.Color = color.NRGBA{R: 0xe6, G: 0xe6, B: 0xe6, A: 0xff}
				return lbl.Layout(gtx)
			})
		})
	})
}

// Overlay draws the dropdown (and a dismiss scrim) on top of everything
// when the menu is open. Call it late in the frame, after the cockpit.
func (m *Menu) Overlay(gtx layout.Context, th *material.Theme) {
	if m.scrim.Clicked(gtx) {
		m.close()
	}
	if m.about.Clicked(gtx) {
		m.selectAbout()
	}
	if m.exit.Clicked(gtx) {
		m.selectExit()
	}
	if !m.open {
		return
	}
	// Transparent full-window catcher: a click anywhere outside the
	// dropdown items lands here and closes the menu.
	m.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	// Dropdown anchored just below the bar, near the left edge.
	off := op.Offset(image.Pt(gtx.Dp(unit.Dp(8)), gtx.Dp(menuBarHeight))).Push(gtx.Ops)
	defer off.Pop()
	m.dropdown(gtx, th)
}

func (m *Menu) dropdown(gtx layout.Context, th *material.Theme) layout.Dimensions {
	w := gtx.Dp(unit.Dp(160))
	gtx.Constraints.Min = image.Pt(w, 0)
	gtx.Constraints.Max.X = w
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(menuItem(th, &m.about, "About")),
		layout.Rigid(menuItem(th, &m.exit, "Exit")),
	)
	call := macro.Stop()
	panel := color.NRGBA{R: 0x28, G: 0x28, B: 0x32, A: 0xff}
	border := color.NRGBA{R: 0x70, G: 0x70, B: 0x80, A: 0xff}
	paint.FillShape(gtx.Ops, panel, clip.Rect{Max: dims.Size}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(dims.Size.X, 1)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, dims.Size.Y-1), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(1, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, border, clip.Rect{Min: image.Pt(dims.Size.X-1, 0), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return dims
}

func menuItem(th *material.Theme, btn *widget.Clickable, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X // full-width hit target
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, label)
				lbl.Color = color.NRGBA{R: 0xe6, G: 0xe6, B: 0xe6, A: 0xff}
				return lbl.Layout(gtx)
			})
		})
	}
}
