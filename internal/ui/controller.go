package ui

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"rfmeter/internal/meter"
	"rfmeter/internal/state"
)

// Controller owns the serial port, buffer, log, and pumps updates into App.
type Controller struct {
	App  *App
	Buf  *state.Buffer
	Hist *state.Histogram
	Log  *state.CsvLog

	mu     sync.Mutex
	port   *meter.Port
	cancel context.CancelFunc
	pages  []meter.Page // last-known config (length up to 8)
}

// NewController wires storage to the App and starts auto-detect.
func NewController(a *App) *Controller {
	c := &Controller{
		App:  a,
		Buf:  state.NewBuffer(12000),
		Hist: state.NewHistogram(),
		Log:  state.NewCsvLog(),
	}
	// Replace synthetic data with real frames.
	a.Hist = c.Hist
	a.fakeFrames = nil

	a.Controls.OnLogToggle = c.toggleLog
	a.Controls.OnApply = c.applyPage
	a.Controls.OnRateSelect = c.setRate
	a.Controls.OnPageSelect = c.selectBand
	a.Keys.OnSelectPage = c.selectBand // F-keys go through the same path as button clicks
	a.Keys.OnToggleLog = c.toggleLog
	a.Attenuator.OnApply = func(rec float64) {
		freq, _ := parseInt(a.Controls.FreqInput.Text())
		c.applyPage(a.Controls.Selected, freq, rec)
		a.ShowAttenuator = false
	}

	go c.detectLoop()
	return c
}

// FramesSince returns frames newer than now-d. Used as App.FrameSource.
func (c *Controller) FramesSince(d time.Duration) []meter.Frame {
	return c.Buf.Since(time.Now().Add(-d))
}

func (c *Controller) detectLoop() {
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		c.mu.Lock()
		connected := c.port != nil
		c.mu.Unlock()
		if !connected {
			ports, _ := meter.DetectPorts()
			for _, p := range ports {
				if p.IsMeter {
					c.connect(p.Device)
					break
				}
			}
		}
		<-tick.C
	}
}

func (c *Controller) connect(dev string) {
	p, err := meter.Open(dev)
	if err != nil {
		log.Printf("open %s: %v", dev, err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.port = p
	c.cancel = cancel
	c.App.Controls.Connected = true
	c.App.Controls.PortLabel = dev
	c.mu.Unlock()
	log.Printf("connected: %s", dev)

	frames := make(chan meter.Frame, 256)
	go func() {
		err := p.ReadFrames(ctx, frames)
		log.Printf("ReadFrames exit: %v", err)
		close(frames)
		c.mu.Lock()
		c.port = nil
		c.App.Controls.Connected = false
		c.App.Controls.PortLabel = "(disconnected)"
		c.mu.Unlock()
	}()

	// Fetch page config in the background so the status line can show MHz.
	go func() {
		pctx, pcancel := context.WithTimeout(ctx, 3*time.Second)
		defer pcancel()
		pages, err := p.ReadConfig(pctx)
		if err != nil {
			log.Printf("ReadConfig: %v", err)
			return
		}
		c.mu.Lock()
		c.pages = pages
		c.mu.Unlock()
		c.updateReadCtx()
	}()

	go func() {
		for f := range frames {
			c.Buf.Append(f)
			c.Hist.Add(f.DBm)
			r := Reading{DBm: f.DBm, WattsW: f.LinearW, Valid: true}
			c.fillReadCtx(&r)
			c.App.Read = r
			if active, _, _ := c.Log.Status(); active {
				_ = c.Log.Write(f, c.App.Controls.Selected)
				_, _, rows := c.Log.Status()
				c.App.Controls.LogRows = rows
			}
		}
	}()
}

// fillReadCtx populates Page/FreqMHz/Offset/Rate from current Controls + cached pages.
func (c *Controller) fillReadCtx(r *Reading) {
	r.Page = c.App.Controls.Selected
	r.Rate = c.App.Controls.Rate
	c.mu.Lock()
	for _, p := range c.pages {
		if p.Letter == r.Page {
			r.FreqMHz = p.FreqMHz
			r.OffsetDB = p.OffsetDB
			r.OffsetKnown = true
			break
		}
	}
	// Also keep PageFreqs/Offsets in sync for the band buttons.
	for _, p := range c.pages {
		idx := int(p.Letter - 'A')
		if idx >= 0 && idx < 8 {
			c.App.Controls.PageFreqs[idx] = p.FreqMHz
			c.App.Controls.PageOffsets[idx] = p.OffsetDB
			c.App.Controls.PageKnown[idx] = true
		}
	}
	c.mu.Unlock()
}

// updateReadCtx refreshes only the context fields on App.Read.
func (c *Controller) updateReadCtx() {
	r := c.App.Read
	c.fillReadCtx(&r)
	c.App.Read = r
}

func (c *Controller) toggleLog() {
	if active, _, _ := c.Log.Status(); active {
		_ = c.Log.Stop()
		c.App.Controls.LogActive = false
		return
	}
	dir, _ := os.Getwd()
	path, err := c.Log.Start(dir)
	if err != nil {
		log.Printf("log start: %v", err)
		return
	}
	c.App.Controls.LogActive = true
	c.App.Controls.LogPath = path
}

// selectBand updates the host's selected band AND tells the device to switch
// its active calibration to that band by re-sending the page's stored config.
// (The protocol has no separate "switch page" command; re-applying the
// page's contents triggers the meter to use it.)
func (c *Controller) selectBand(letter byte) {
	c.App.Controls.Selected = letter
	idx := int(letter - 'A')
	if idx < 0 || idx >= 8 {
		return
	}
	freq := c.App.Controls.PageFreqs[idx]
	off := c.App.Controls.PageOffsets[idx]
	c.updateReadCtx()
	if freq <= 0 {
		// No cached config to re-send; can't tell the device to switch.
		return
	}
	c.applyPage(letter, freq, off)
}

func (c *Controller) applyPage(letter byte, freq int, off float64) {
	c.mu.Lock()
	p := c.port
	c.mu.Unlock()
	if p == nil {
		log.Printf("apply: no port")
		return
	}
	cmd := meter.BuildSetPageCmd(letter, freq, off)
	if err := p.SendCmd(cmd); err != nil {
		log.Printf("send %q: %v", cmd, err)
		return
	}
	c.applyPageInternal(letter, freq, off)
}

func (c *Controller) setRate(r Rate) {
	c.mu.Lock()
	p := c.port
	c.mu.Unlock()
	c.updateReadCtx()
	if p == nil {
		return
	}
	if err := p.SendCmd(r.Command()); err != nil {
		log.Printf("rate: %v", err)
	}
}

func (c *Controller) applyPageInternal(letter byte, freq int, off float64) {
	// After Apply, refresh cached pages so the status line shows the new freq.
	c.mu.Lock()
	updated := false
	for i := range c.pages {
		if c.pages[i].Letter == letter {
			c.pages[i].FreqMHz = freq
			c.pages[i].OffsetDB = off
			updated = true
			break
		}
	}
	if !updated {
		c.pages = append(c.pages, meter.Page{Letter: letter, FreqMHz: freq, OffsetDB: off})
	}
	c.mu.Unlock()
	c.updateReadCtx()
}
