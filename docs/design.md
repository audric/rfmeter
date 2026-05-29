# USB RF Power Meter — Desktop App Design

Date: 2026-05-29
Target platform: Linux (x86_64), single statically-linked binary.

## Purpose

A native Linux desktop "cockpit" for the Chinese USB RF Power Meter V3
(STM32-based, enumerates as USB VID `0483` / PID `5740`, `STM32 Virtual
COM Port`, exposed at `/dev/ttyACM*`). The meter streams ASCII frames
of the form `a±XXXXXXXX[uwm]A` over CDC ACM at 921600 baud.

The CLI prototype `rfmeter.py` already parses the protocol and dumps
CSV. This app replaces it with an interactive GUI that shows live
power, a rolling time-series plot, a histogram of recent readings,
and per-page calibration controls — in a single compiled Go binary,
no Python, no JS.

## Goals & Non-Goals

**Goals**

- Single statically-linked Linux binary (~15-20 MB).
- Auto-detect the meter on startup; auto-reconnect on unplug.
- Live numeric readout (dBm + linear power) at the meter's rate.
- Rolling time-series plot of the last 60 s (configurable later).
- Histogram of recent samples in 1 dB bins.
- Read/edit the 8 calibration pages (A-H: frequency + offset).
- Change sample rate (S0/S1/S2 commands).
- Start/stop CSV logging on user command.
- PNG snapshot of the cockpit.
- Attenuator helper (compute required attenuation for a target signal).
- Function-key shortcuts for all common actions.

**Non-goals**

- Not building for Windows or macOS in v1 (no blockers, just out of scope).
- No FFT / spectrum view — the meter outputs a single scalar power
  reading, not spectral data.
- No 9th calibration page; firmware only exposes 8 (we verified by
  reading config — page I never appears in the `Read` reply).
- No firmware flashing / DFU.

## High-level architecture

Three independent layers, each testable in isolation, communicating
through bounded Go channels:

```
                  +----------------+
                  |  meter (lib)   |     pure serial I/O
                  +----------------+
                       |   ^
                  framesCh   cmdCh
                       v   |
                  +----------------+
                  |  state (lib)   |     ring buffer, stats, CSV log
                  +----------------+
                       |   ^
                  redrawCh   userActions
                       v   |
                  +----------------+
                  |  ui (Gio)      |     widgets, layout, input
                  +----------------+
```

- `meter` exposes `Open`, `ReadFrames`, `SendCmd`, `ReadConfig`. It has
  no awareness of the UI and is reusable for a future CLI/headless mode.
- `state` owns the ring buffer of recent samples, rolling stats, and
  the CSV writer. It receives `Frame`s on `framesCh`, updates its
  internal state, and emits coalesced redraw signals to the UI.
- `ui` paints the cockpit, dispatches input (mouse, keyboard, function
  keys) into `userActions`, which the controller layer routes to
  `cmdCh` (for serial commands), to `state` (for log start/stop), or
  to internal UI state (for modals, pause).

## Package layout

```
cmd/rfmeter/main.go                  // entrypoint, wires the layers
internal/meter/serial.go             // open, enumerate, reconnect
internal/meter/frame.go              // parser + Frame type
internal/meter/config.go             // ReadConfig, SetPage command builder
internal/state/buffer.go             // ring buffer
internal/state/stats.go              // rolling min/max/mean + histogram
internal/state/csvlog.go             // CSV writer
internal/ui/app.go                   // root layout, event loop
internal/ui/readout.go               // big numeric widget
internal/ui/plot.go                  // time-series widget
internal/ui/histogram.go             // histogram widget
internal/ui/controls.go              // page buttons, rate radio, log btn
internal/ui/modal_help.go            // F1
internal/ui/modal_attenuator.go      // F2
internal/ui/keys.go                  // function-key dispatch
internal/ui/snapshot.go              // PNG export (uses fogleman/gg)
```

## Dependencies

- `gioui.org` — UI toolkit (immediate-mode, no GTK/Qt dep).
- `gioui.org/x/component` — only if needed for modals; else hand-rolled.
- `go.bug.st/serial` — serial port open + `enumerator.GetDetailedPortsList()`
  for VID/PID-based meter detection.
- `github.com/fogleman/gg` — off-screen 2D rendering for PNG snapshot.

Standard library: `encoding/csv`, `image/png`, `regexp` (only for the
streaming parser fast path; main parsing is a byte-state machine).

## Protocol (recap, authoritative)

| Direction | Bytes | Meaning |
|---|---|---|
| meter → host | `a±DDDLLLLL[uwm]A` (12 ASCII chars, no separator) | streaming power frame |
| meter → host | `R<freq><±off.o>` × N | reply to `Read` (N is up to 8 on this firmware) |
| host → meter | `S0\r\n` / `S1\r\n` / `S2\r\n` | sample rate slow / medium / fast |
| host → meter | `<A-H><freq><±off.o>\r\n` (e.g. `A2400+10.0`) | set page calibration |
| host → meter | `<A-H>...<A-H>...\r\n` concatenated | set all pages in one shot |
| host → meter | `Read\r\n` | dump current page config |

Frame fields: sign (1 char), dBm × 10 (3 digits), linear power × 100 (5
digits), unit char (`u`=µW, `m`=mW, `w`=W), trailing `A`.
Example: `a-39200011uA` = -39.2 dBm, 0.11 µW.

The firmware echoes recent host writes back into the stream
intermittently (we observed this with `R\nVER\nver` and stray `A`
bytes interleaved). The parser MUST tolerate garbage between frames
without dropping valid frames.

## Components — detail

### meter.Frame

```go
type Frame struct {
    T       time.Time
    DBm     float64 // -99.9 .. +99.9
    LinearW float64 // power in watts, computed from raw + unit
    Unit    byte    // 'u', 'm', 'w' as reported
    Raw     []byte  // the 12-byte original (for debugging)
}
```

### meter.Open / auto-detect

```go
func DetectPorts() ([]PortInfo, error) // VID=0483 PID=5740 matches
func Open(path string) (*Port, error)  // 921600 8N1
```

`DetectPorts` returns ports whose USB VID/PID match the meter, along
with serial numbers. If exactly one match, the UI auto-connects on
startup. Otherwise the user picks from a dropdown.

### meter.ReadFrames

```go
func (p *Port) ReadFrames(ctx context.Context, out chan<- Frame) error
```

Runs in its own goroutine. Reads bytes into a 64-byte scratch buffer,
feeds a byte-state machine. The machine scans for `a`, then validates
the next 11 bytes match `[+-]\d{3}\d{5}[uwm]A`; on success it emits a
`Frame{T: time.Now(), ...}` to `out`. On any byte mismatch mid-frame
it discards one byte and resumes scanning. Garbage between frames is
silently dropped. Returns on `ctx.Done()` or fatal port error.

### meter.SendCmd / ReadConfig

```go
func (p *Port) SendCmd(cmd string) error          // appends "\r\n"
func (p *Port) ReadConfig(ctx) ([]Page, error)    // sends "Read", parses reply
```

`ReadConfig` waits up to 2 s for the reply, runs `frame.SubReply`
(strips streaming `a...uA` frames from the buffer first), then matches
`(\d{4})([+-]\d{2}\.\d)` repeatedly. Returns up to 8 `Page{Letter,
FreqMHz, OffsetDB}` entries.

### state.Buffer

A bounded ring of `Frame`s sized for 60 s at the highest expected rate
(say 200 Hz → 12 000 entries, ~250 KB). The UI plot reads the tail
through a snapshot method that copies under a `sync.RWMutex`. Writes
are non-blocking; reads at 60 Hz max.

### state.Stats

Rolling min/max/mean over the last N seconds (default 60). Histogram:
fixed bins from -80 to 0 dBm in 1 dB steps (80 bins) + one overflow
bin. Updated incrementally on each frame.

### state.CsvLog

```go
type CsvLog struct { ... }
func (l *CsvLog) Start(dir string) (path string, err error)
func (l *CsvLog) Write(f Frame, page byte) error
func (l *CsvLog) Stop() error
```

Header: `time_iso,dbm,linear_w,unit,page`. Filename:
`rfmeter_YYYYMMDD_HHMMSS.csv`. `Write` is called from the state
goroutine for every frame. `fsync` every 1 s on a ticker (so a crash
loses at most 1 s of data). `Stop` flushes and closes.

### ui.App

Owns the layout. The root is a horizontal split:

- Left column, fixed 280 px: controls panel.
- Right column, flex: visuals panel (readout / plot / histogram /
  stats), vertically stacked with 4 / 4 / 3 / 1 weights respectively.

At most one modal is active at a time, rendered on top of the
cockpit. Esc dismisses. The cockpit continues updating behind the
modal (so you can watch readings while reading help).

### ui.Plot (custom widget)

- X axis: time, last 60 s, right-aligned at "now".
- Y axis: dBm, auto-ranges with hysteresis — only re-fits when the
  newest sample is outside `[ymin-margin, ymax+margin]` with margin =
  10% of current range. Prevents jitter when noise floor wanders.
- Downsamples to ≤1 sample per horizontal pixel when zoomed out by
  taking min/max per bucket and drawing as a thin vertical band — keeps
  the polyline cheap and preserves spikes.
- Pause (Space): freezes both axes and stops scrolling, but logging
  and stats keep running. Visible "PAUSED" overlay.

### ui.Histogram (custom widget)

80 bars, fixed -80..0 dBm range. Bar heights = log(count) for
readability (one big spike doesn't flatten the rest). A vertical
line marks the rolling mean.

### ui.Readout

Two text lines, large:
- Line 1: signed dBm to 1 decimal, ~64 pt.
- Line 2: linear power with auto units (pW / nW / µW / mW / W), ~32 pt.

### ui.Controls

- Port dropdown + Connect/Disconnect + status dot (green / amber
  reconnecting / red).
- Auto-reconnect checkbox (default on).
- 8 page buttons in a 4×2 grid; selected page highlighted.
- Sample rate radio: S0 / S1 / S2.
- Frequency + offset text inputs + "Apply to <page>" button.
- "Read config" button (re-runs `ReadConfig`).
- Logging: "Start log" / "Stop log" + filename + row count.

### ui.Snapshot (PNG export)

F12 triggers. Renders an 1920×1080 image with `gg`:
- Top: big readout text.
- Middle: time-series plot (same logic as on-screen plot, but redrawn
  off-screen at the snapshot dimensions).
- Bottom: histogram + stats text block.

Saved as `rfmeter_snapshot_YYYYMMDD_HHMMSS.png` in CWD. A toast at the
bottom of the window shows the path for ~3 s.

## Function-key map

| Key | Action |
|---|---|
| F1  | Help modal (protocol notes, control reference, this keymap) |
| F2  | Attenuator helper modal |
| F3..F10 | Select page A..H (see page-selection semantics below) |
| F11 | Toggle CSV log |
| F12 | Export PNG snapshot |
| Esc | Close current modal |
| Space | Toggle pause on the live plot |

### Page-selection semantics

The documented protocol exposes commands to **set** a page's freq +
offset but **not** to switch the meter's active page (the physical
S1/S2 buttons on the device do that). Consequently, "select page A"
in the UI is a host-side action only:

- The current page indicator updates to A.
- The freq + offset edit fields fill with page A's stored values.
- The next `Apply` writes to page A.
- The CSV log's `page` column reflects which page the host is
  currently focused on.

The meter's actual active calibration may differ from the UI's
selection. We surface this caveat in the help modal.

### F2 — attenuator helper

Inputs:
- Expected signal power: numeric entry + unit toggle (dBm / W).
- Meter max safe input: pre-filled at +0 dBm, editable.
- Safety margin: pre-filled at 6 dB, editable.

Outputs:
- Required attenuation in dB = `signal_dBm − max_dBm + margin`.
- "Resulting reading will be ≈ X dBm" preview.
- "Apply to current page" button → writes the computed attenuation as
  the page offset via the `<page><freq>±XX.X` command (using the
  current page's freq, the new offset). After write, re-runs
  `ReadConfig` to confirm.

If the user enters a signal below the meter's noise floor (~-50 dBm
without attenuator) the helper warns instead of recommending a
negative-attenuation pad.

## Auto-detect & auto-reconnect

- **Startup**: `DetectPorts` runs immediately. If 1 match, open and
  start streaming. If >1, populate dropdown, leave disconnected. If 0,
  show "No meter found — plug it in" in the status area.
- **Disconnect detection**: the `ReadFrames` goroutine returns on
  `io.EOF` or a `syscall.ENODEV`. The controller catches the return,
  sets status to "reconnecting" if the checkbox is on, and starts a
  1 s ticker that re-runs `DetectPorts` + open. The ticker cancels on
  successful open or on user "Disconnect".
- The on-screen status dot reflects: `disconnected` (red),
  `reconnecting` (amber, pulsing), `streaming` (green).

## Threading model

- 1 goroutine: serial reader (`ReadFrames`), produces to `framesCh`.
- 1 goroutine: state updater, consumes `framesCh`, updates buffer +
  stats + CSV log, emits to `redrawCh` (rate-limited to 60 Hz via a
  ticker).
- 1 goroutine: serial writer, drains `cmdCh`, writes commands.
- 1 goroutine: Gio main loop, reads `redrawCh` to invalidate, dispatches
  input events to `userActionsCh`.

Channels are bounded (8-256) to backpressure naturally; the state
goroutine drops `redrawCh` sends when the UI hasn't consumed yet (it
only needs the latest "redraw please" anyway).

## Build & distribution

```bash
go mod init github.com/audric/rfmeter   # if not already
go mod tidy
go build -o rfmeter ./cmd/rfmeter        # ~15-20 MB single binary
./rfmeter                                # runs immediately, no install
```

Requires the user to be in the `dialout` group (already is in this
environment). Optionally we ship a `.desktop` file for the GNOME app
grid; not required for v1.

No CGo dependencies — Gio uses Wayland/X11/EGL directly. The binary
runs on any glibc Linux from the last few years.

## Testing

- **Frame parser**: table-driven tests with good frames, partial
  frames at buffer boundaries, garbage prefixes/suffixes, the exact
  `R\nVER\nver` interleaved sequence we observed.
- **ReadConfig**: fed the captured byte stream from the live device,
  must return the 8 known pages even with embedded echo garbage.
- **Stats**: ring buffer rolloff at the 60 s boundary, histogram bin
  edges, log-scaled bar heights.
- **CSV log**: header + rows correct, fsync interval, clean Stop.
- **UI**: not unit-tested. Manual smoke test against the live meter
  before shipping.

## Open questions (to revisit during implementation)

- Should pause apply to histogram too, or only the time-series? Tentative
  answer: time-series only (histogram is already an aggregate).
- Should F11 prompt for a filename or always auto-name? Tentative
  answer: auto-name, since the use case is "hit a key, start logging".
  Add a "Save as…" menu item later if anyone asks.

## What this design intentionally omits

- Network export (Prometheus, MQTT). Add later if needed.
- Multiple-meter support. The state and channels assume one device.
- A "history" mode for replaying CSV logs. The CLI can already do this;
  no need to duplicate in the GUI for v1.
- Theming / dark-light toggle. Gio default look is fine.
- Internationalisation. English only.
