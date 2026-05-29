package main

import (
	"log"
	"os"
	"time"

	"gioui.org/app"

	"rfmeter/internal/meter"
	"rfmeter/internal/ui"
)

func main() {
	go func() {
		a := ui.New()
		c := ui.NewController(a)
		a.FrameSource = func() []meter.Frame { return c.FramesSince(60 * time.Second) }
		w := new(app.Window)
		w.Option(ui.DefaultWindowOptions()...)
		// Drive redraws at ~20 Hz so the live plot scrolls smoothly.
		go func() {
			t := time.NewTicker(50 * time.Millisecond)
			for range t.C {
				w.Invalidate()
			}
		}()
		if err := a.Run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
