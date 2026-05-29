package meter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Meter USB identifiers (STM32 Virtual COM Port).
const (
	MeterVID = "0483"
	MeterPID = "5740"
)

// PortInfo identifies a candidate serial port.
type PortInfo struct {
	Device   string // e.g. "/dev/ttyACM0"
	SerialNo string
	IsMeter  bool // VID/PID matches
}

// DetectPorts enumerates USB serial ports and flags ones matching the meter.
func DetectPorts() ([]PortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	var out []PortInfo
	for _, p := range ports {
		if !p.IsUSB {
			continue
		}
		vid := strings.ToLower(p.VID)
		pid := strings.ToLower(p.PID)
		out = append(out, PortInfo{
			Device:   p.Name,
			SerialNo: p.SerialNumber,
			IsMeter:  vid == strings.ToLower(MeterVID) && pid == strings.ToLower(MeterPID),
		})
	}
	return out, nil
}

// Port is an opened meter connection.
type Port struct {
	dev string
	rw  serial.Port
	mu  sync.Mutex // serializes SendCmd writes
}

// Open opens dev at 921600 8N1.
func Open(dev string) (*Port, error) {
	mode := &serial.Mode{
		BaudRate: 921600,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	p, err := serial.Open(dev, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dev, err)
	}
	return &Port{dev: dev, rw: p}, nil
}

// Close releases the port.
func (p *Port) Close() error { return p.rw.Close() }

// SendCmd writes cmd + "\r\n".
func (p *Port) SendCmd(cmd string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.rw.Write([]byte(cmd + "\r\n"))
	return err
}

// ReadFrames consumes bytes and emits Frame values until ctx is done or
// a read error occurs. Returns nil on clean ctx cancel, error on I/O failure.
func (p *Port) ReadFrames(ctx context.Context, out chan<- Frame) error {
	br := bufio.NewReaderSize(p.rw, 4096)
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 256)
	_ = p.rw.SetReadTimeout(200 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		n, err := br.Read(tmp)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return io.EOF
			}
			if strings.Contains(err.Error(), "timeout") {
				continue
			}
			return fmt.Errorf("read %s: %w", p.dev, err)
		}
		if n == 0 {
			continue
		}
		buf = append(buf, tmp[:n]...)
		frames, tail := Extract(buf)
		buf = tail
		now := time.Now()
		for _, f := range frames {
			f.T = now
			select {
			case out <- f:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// ReadConfig sends "Read" and parses the reply.
func (p *Port) ReadConfig(ctx context.Context) ([]Page, error) {
	if err := p.SendCmd("Read"); err != nil {
		return nil, err
	}
	_ = p.rw.SetReadTimeout(200 * time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	var buf []byte
	tmp := make([]byte, 256)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		n, err := p.rw.Read(tmp)
		if err != nil && !strings.Contains(err.Error(), "timeout") {
			return nil, err
		}
		buf = append(buf, tmp[:n]...)
	}
	pages := ParseConfigReply(buf)
	if len(pages) == 0 {
		return nil, errors.New("no pages parsed from reply")
	}
	return pages, nil
}
