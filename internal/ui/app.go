package ui

import (
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"rfmeter/internal/meter"
	"rfmeter/internal/state"
)

// App is the root UI controller.
type App struct {
	Theme          *material.Theme
	Read           Reading
	Plot           *Plot
	Hist           *state.Histogram
	Controls       *Controls
	Keys           *KeyHandler
	Attenuator     *Attenuator
	Menu           *Menu
	ShowHelp       bool
	ShowAttenuator bool
	// FrameSource returns frames for the plot. Set by Controller; defaults to fakeFrames.
	FrameSource func() []meter.Frame
	fakeFrames  []meter.Frame

	// snapshotWriter renders+encodes the PNG. Indirected for testing;
	// defaults to Snapshot.
	snapshotWriter func(dir string, d SnapshotData) (string, error)
	// snapshotBusy is true while a snapshot is rendering off-thread, so
	// repeated F12 presses are ignored rather than piling up goroutines.
	snapshotBusy atomic.Bool
	// toast holds a transient on-screen status message (e.g. snapshot
	// progress). Read on the UI goroutine, written from the writer goroutine.
	toast atomic.Pointer[toastMsg]
}

// New returns an App with default theme and synthetic demo data.
func New() *App {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	// Dark palette.
	th.Palette.Bg = color.NRGBA{R: 0x18, G: 0x18, B: 0x1c, A: 0xff}
	th.Palette.Fg = color.NRGBA{R: 0xe6, G: 0xe6, B: 0xe6, A: 0xff}
	th.Palette.ContrastBg = color.NRGBA{R: 0x30, G: 0x50, B: 0x80, A: 0xff}
	th.Palette.ContrastFg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	a := &App{
		Theme:          th,
		Read:           Reading{DBm: -42.3, WattsW: 5.9e-8, Valid: true},
		Plot:           NewPlot(),
		Hist:           state.NewHistogram(),
		Controls:       NewControls(th),
		snapshotWriter: Snapshot,
	}
	now := time.Now()
	for i := 0; i < 600; i++ {
		dbm := -45 + 3*math.Sin(float64(i)/15)
		a.fakeFrames = append(a.fakeFrames, meter.Frame{
			T:   now.Add(time.Duration(i-600) * 100 * time.Millisecond),
			DBm: dbm,
		})
		a.Hist.Add(dbm)
	}
	// Default debug callbacks; real wiring replaces these in Controller (task 19).
	a.Controls.OnPageSelect = func(letter byte) { log.Printf("page select: %c", letter) }
	a.Controls.OnRateSelect = func(r Rate) { log.Printf("rate select: %s", r.Command()) }
	a.Controls.OnApply = func(letter byte, freq int, off float64) {
		log.Printf("apply: %s", meter.BuildSetPageCmd(letter, freq, off))
	}
	a.Controls.OnLogToggle = func() { log.Printf("log toggle") }

	a.FrameSource = func() []meter.Frame { return a.fakeFrames }
	// OnExit is wired in Run, where the window handle is available.
	a.Menu = &Menu{OnAbout: func() { a.ShowHelp = true }}
	a.Attenuator = NewAttenuator(th)
	a.Attenuator.OnApply = func(rec float64) {
		freq, _ := parseInt(a.Controls.FreqInput.Text())
		cmd := meter.BuildSetPageCmd(a.Controls.Selected, freq, rec)
		log.Printf("attenuator apply: %.1f dB → page %c (cmd %s)", rec, a.Controls.Selected, cmd)
		a.ShowAttenuator = false
	}
	a.Keys = &KeyHandler{
		OnHelp:       func() { a.ShowHelp = true },
		OnAttenuator: func() { a.ShowAttenuator = true },
		OnSelectPage: func(l byte) { a.Controls.Selected = l; log.Printf("F: page %c", l) },
		OnToggleLog:  func() { log.Printf("F11: toggle log") },
		OnSnapshot:   a.snapshot,
		OnEscape: func() {
			a.ShowHelp = false
			a.ShowAttenuator = false
			a.Menu.close()
		},
		OnSpace: func() { a.Plot.Paused = !a.Plot.Paused },
	}
	return a
}

// snapshot captures the current cockpit state and renders the PNG on a
// background goroutine. F12 runs on the Gio event-loop goroutine, and a
// full 1920x1080 render + PNG encode + file write takes long enough
// (notably on Windows) to freeze the window if done inline. We capture
// the inputs synchronously here — cloning the histogram so the writer
// touches only immutable data — then hand off to a goroutine. Overlapping
// presses are ignored while one render is in flight.
func (a *App) snapshot() {
	if !a.snapshotBusy.CompareAndSwap(false, true) {
		return
	}
	cwd, _ := os.Getwd()
	frames := a.FrameSource()
	var hist *state.Histogram
	if a.Hist != nil {
		hist = a.Hist.Clone()
	}
	d := SnapshotData{
		Reading: a.Read,
		Frames:  frames,
		Hist:    hist,
		Stats:   state.ComputeStats(frames),
		WindowS: a.Plot.WindowSec,
		Page:    a.Controls.Selected,
	}
	a.toast.Store(&toastMsg{text: "Saving snapshot…"})
	go func() {
		defer a.snapshotBusy.Store(false)
		path, err := a.snapshotWriter(cwd, d)
		if err != nil {
			log.Printf("snapshot: %v", err)
			a.toast.Store(&toastMsg{text: "Snapshot failed", until: time.Now().Add(4 * time.Second)})
			return
		}
		log.Printf("snapshot saved: %s", path)
		a.toast.Store(&toastMsg{text: "Saved " + filepath.Base(path), until: time.Now().Add(3 * time.Second)})
	}()
}

// Run drives the Gio event loop. Returns when the window is closed.
func (a *App) Run(w *app.Window) error {
	a.Menu.OnExit = func() { w.Perform(system.ActionClose) }
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			for {
				ev, ok := gtx.Event(a.Keys.Filters()...)
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok {
					a.Keys.Handle(ke)
				}
			}
			paint.Fill(gtx.Ops, color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xff})
			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.Menu.Bar(gtx, a.Theme)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(280))
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return a.Controls.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return (&Readout{Th: a.Theme}).Layout(gtx, a.Read)
								}),
								layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
									return a.Plot.Layout(gtx, a.Theme, a.FrameSource())
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									mean := 0.0
									if a.Hist != nil {
										mean = state.ComputeStats(a.FrameSource()).MeanDBm
									}
									return HistogramView{}.Layout(gtx, a.Theme, a.Hist, mean)
								}),
							)
						}),
					)
				}),
			)
			if a.ShowHelp {
				HelpModal(gtx, a.Theme)
			}
			if a.ShowAttenuator {
				a.Attenuator.Layout(gtx, a.Controls.Selected)
			}
			a.Menu.Overlay(gtx, a.Theme)
			a.drawToast(gtx, a.Theme)
			e.Frame(gtx.Ops)
		}
	}
}

// DefaultWindowOptions returns the window sizing options.
func DefaultWindowOptions() []app.Option {
	return []app.Option{
		app.Title("RF Meter"),
		app.Size(unit.Dp(900), unit.Dp(600)),
		app.MinSize(unit.Dp(640), unit.Dp(400)),
	}
}
