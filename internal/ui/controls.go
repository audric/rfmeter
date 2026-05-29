package ui

import (
	"fmt"
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

// Rate is the meter sample-rate setting.
type Rate int

const (
	RateS0 Rate = iota // slow
	RateS1             // medium (default)
	RateS2             // fast
)

// Command returns the wire string for the rate ("S0"/"S1"/"S2").
func (r Rate) Command() string {
	return [...]string{"S0", "S1", "S2"}[r]
}

// Controls holds the persistent state of the left-column widgets.
type Controls struct {
	Th *material.Theme

	PageBtns    [8]widget.Clickable
	Selected    byte    // 'A'..'H'
	PageFreqs   [8]int     // saved freq MHz per band; 0 = unknown
	PageOffsets [8]float64 // saved offset dB per band
	PageKnown   [8]bool    // whether the corresponding band's config is known

	RateBtns [3]widget.Clickable
	Rate     Rate

	FreqInput   widget.Editor
	OffsetInput widget.Editor
	ApplyBtn    widget.Clickable

	LogBtn    widget.Clickable
	LogActive bool
	LogPath   string
	LogRows   int64

	Connected bool
	PortLabel string

	// Callbacks set by App.
	OnPageSelect func(letter byte)
	OnRateSelect func(r Rate)
	OnApply      func(letter byte, freq int, offsetDB float64)
	OnLogToggle  func()
}

// Factory-default band frequencies (from the meter's docs / observed Read reply).
var defaultBandFreqsMHz = [8]int{100, 200, 300, 400, 1000, 2000, 5000, 6000}

// NewControls initialises Controls with default page A and rate S1.
func NewControls(th *material.Theme) *Controls {
	c := &Controls{Th: th, Selected: 'A', Rate: RateS1, PortLabel: "(no device)"}
	c.FreqInput.SingleLine = true
	c.FreqInput.Filter = "0123456789"
	c.OffsetInput.SingleLine = true
	c.OffsetInput.Filter = "0123456789+-."
	for i, f := range defaultBandFreqsMHz {
		c.PageFreqs[i] = f
		c.PageOffsets[i] = 0.0
		c.PageKnown[i] = true
	}
	return c
}

// Layout draws the entire controls column and dispatches click events.
func (c *Controls) Layout(gtx layout.Context) layout.Dimensions {
	for i := range c.PageBtns {
		for c.PageBtns[i].Clicked(gtx) {
			c.Selected = 'A' + byte(i)
			if c.OnPageSelect != nil {
				c.OnPageSelect(c.Selected)
			}
		}
	}
	for i := range c.RateBtns {
		for c.RateBtns[i].Clicked(gtx) {
			c.Rate = Rate(i)
			if c.OnRateSelect != nil {
				c.OnRateSelect(c.Rate)
			}
		}
	}
	for c.ApplyBtn.Clicked(gtx) {
		freq, _ := parseInt(c.FreqInput.Text())
		off, _ := parseFloat(c.OffsetInput.Text())
		if c.OnApply != nil {
			c.OnApply(c.Selected, freq, off)
		}
	}
	for c.LogBtn.Clicked(gtx) {
		if c.OnLogToggle != nil {
			c.OnLogToggle()
		}
	}
	inset := layout.UniformInset(unit.Dp(8))
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx,
			layout.Rigid(c.section("Connection")),
			layout.Rigid(c.statusRow()),
			layout.Rigid(spacer(12)),

			layout.Rigid(c.section("Bands")),
			layout.Rigid(c.pageGrid()),
			layout.Rigid(spacer(12)),

			layout.Rigid(c.section("Edit page")),
			layout.Rigid(c.labeledEditor("Freq (MHz)", &c.FreqInput, "e.g. 2400")),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(c.labeledEditor("Offset (dB)", &c.OffsetInput, "e.g. +10.0")),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(c.btn(&c.ApplyBtn, fmt.Sprintf("Apply to %s", selectedFkey(c.Selected)))),
			layout.Rigid(spacer(12)),

			layout.Rigid(c.section("Sample rate")),
			layout.Rigid(c.rateRow()),
			layout.Rigid(spacer(12)),

			layout.Rigid(c.section("Logging (F11)")),
			layout.Rigid(c.btn(&c.LogBtn, logLabel(c.LogActive))),
			layout.Rigid(material.Caption(c.Th, c.logStatus()).Layout),
		)
	})
}

func (c *Controls) section(title string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return material.H6(c.Th, title).Layout(gtx)
	}
}

// statusRow renders a colored dot next to the port label.
func (c *Controls) statusRow() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Dp(unit.Dp(10))
				col := color.NRGBA{R: 0xc0, G: 0x40, B: 0x40, A: 0xff}
				if c.Connected {
					col = color.NRGBA{R: 0x40, G: 0xc0, B: 0x60, A: 0xff}
				}
				paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
				return layout.Dimensions{Size: image.Pt(sz, sz)}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(material.Body2(c.Th, c.PortLabel).Layout),
		)
	}
}

func (c *Controls) btn(b *widget.Clickable, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx,
			material.Button(c.Th, b, label).Layout)
	}
}

func (c *Controls) labeledEditor(label string, e *widget.Editor, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.Body2(c.Th, label).Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(c.Th, e, hint)
				ed.HintColor = color.NRGBA{R: 0x70, G: 0x70, B: 0x78, A: 0xff}
				return editorBox(gtx, ed.Layout)
			}),
		)
	}
}

// editorBox wraps a widget in a bordered, slightly-different-bg panel so it's
// visible against the dark theme.
func editorBox(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(6)).Layout(gtx, w)
	call := macro.Stop()
	bgCol := color.NRGBA{R: 0x0c, G: 0x0c, B: 0x12, A: 0xff}
	borderCol := color.NRGBA{R: 0x60, G: 0x60, B: 0x70, A: 0xff}
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: dims.Size}.Op())
	paint.FillShape(gtx.Ops, borderCol, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(dims.Size.X, 1)}.Op())
	paint.FillShape(gtx.Ops, borderCol, clip.Rect{Min: image.Pt(0, dims.Size.Y-1), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, borderCol, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(1, dims.Size.Y)}.Op())
	paint.FillShape(gtx.Ops, borderCol, clip.Rect{Min: image.Pt(dims.Size.X-1, 0), Max: image.Pt(dims.Size.X, dims.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return dims
}

func (c *Controls) pageGrid() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		row := func(start int) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{}
				for i := start; i < start+4; i++ {
					i := i
					letter := byte('A') + byte(i)
					fkey := fmt.Sprintf("F%d", i+3) // F3..F10
					freq, off := "—", ""
					if c.PageKnown[i] {
						freq = fmt.Sprintf("%d MHz", c.PageFreqs[i])
						off = fmt.Sprintf("%+.1f dB", c.PageOffsets[i])
					}
					if i > start {
						children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout))
					}
					children = append(children, layout.Flexed(1, c.bandBtn(&c.PageBtns[i], fkey, freq, off, c.Selected == letter)))
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
			}
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(row(0)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(row(4)),
		)
	}
}

// bandBtn renders a three-line band button: F-key on top, freq + offset below.
func (c *Controls) bandBtn(b *widget.Clickable, fkey, freq, off string, selected bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		style := material.ButtonLayout(c.Th, b)
		if selected {
			style.Background = color.NRGBA{R: 0x50, G: 0x80, B: 0xc0, A: 0xff}
		}
		return style.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				top := material.Body1(c.Th, fkey)
				top.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				mid := material.Caption(c.Th, freq)
				mid.Color = color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd8, A: 0xff}
				bot := material.Caption(c.Th, off)
				bot.Color = color.NRGBA{R: 0xb0, G: 0xb0, B: 0xb8, A: 0xff}
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(top.Layout),
					layout.Rigid(mid.Layout),
					layout.Rigid(bot.Layout),
				)
			})
		})
	}
}

func (c *Controls) rateRow() layout.Widget {
	labels := []string{"S0", "S1", "S2"}
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{}
		for i, l := range labels {
			i, l := i, l
			lbl := l
			if Rate(i) == c.Rate {
				lbl = "[" + lbl + "]"
			}
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout))
			}
			children = append(children, layout.Flexed(1, c.btn(&c.RateBtns[i], lbl)))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	}
}

func (c *Controls) logStatus() string {
	if !c.LogActive {
		return "log: stopped"
	}
	return fmt.Sprintf("log: %s (%d rows)", c.LogPath, c.LogRows)
}

func selectedFkey(letter byte) string {
	if letter < 'A' || letter > 'H' {
		return "F?"
	}
	return fmt.Sprintf("F%d", int(letter-'A')+3)
}

func logLabel(active bool) string {
	if active {
		return "Stop log"
	}
	return "Start log"
}

func spacer(dp int) layout.Widget {
	return layout.Spacer{Height: unit.Dp(float32(dp))}.Layout
}

// parseInt and parseFloat are used by Task 14 wiring.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
