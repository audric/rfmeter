package meter

import (
	"fmt"
	"regexp"
	"strconv"
)

// Page is one of the meter's calibration entries A..H.
type Page struct {
	Letter   byte    // 'A'..'H'
	FreqMHz  int     // 0..9999
	OffsetDB float64 // -99.9..+99.9
}

var (
	streamingFrame = regexp.MustCompile(`a[+-]\d{8}[umw]A`)
	pageEntry      = regexp.MustCompile(`(\d{4})([+-]\d{2}\.\d)`)
)

// ParseConfigReply extracts up-to-8 pages from a Read reply.
// Streaming frames are stripped first; remaining garbage (R, \n, VER, ver, A)
// is ignored as long as the freq+offset tuples remain intact.
func ParseConfigReply(b []byte) []Page {
	cleaned := streamingFrame.ReplaceAll(b, nil)
	matches := pageEntry.FindAllSubmatch(cleaned, -1)
	var pages []Page
	for i, m := range matches {
		if i >= 8 {
			break
		}
		freq, _ := strconv.Atoi(string(m[1]))
		off, _ := strconv.ParseFloat(string(m[2]), 64)
		pages = append(pages, Page{
			Letter:   'A' + byte(i),
			FreqMHz:  freq,
			OffsetDB: off,
		})
	}
	return pages
}

// BuildSetPageCmd formats a "set page" command (e.g. "A2400+10.0").
// Caller appends "\r\n".
func BuildSetPageCmd(letter byte, freqMHz int, offsetDB float64) string {
	sign := '+'
	off := offsetDB
	if off < 0 {
		sign = '-'
		off = -off
	}
	return fmt.Sprintf("%c%04d%c%04.1f", letter, freqMHz, sign, off)
}
