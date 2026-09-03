package simreplay

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ProgressFunc reports simulation progress; gameTime is in seconds, maxSec the target.
type ProgressFunc func(gameTime, maxSec float64)

// TerminalProgress draws an ASCII progress bar to a terminal stream.
type TerminalProgress struct {
	Label   string
	Out     io.Writer
	BarWide int
	lastPct int
}

// NewTerminalProgress creates a stderr progress reporter.
func NewTerminalProgress(label string) *TerminalProgress {
	return &TerminalProgress{
		Label:   label,
		Out:     os.Stderr,
		BarWide: 40,
		lastPct: -1,
	}
}

// Update redraws the bar when the rounded percent changes.
func (p *TerminalProgress) Update(gameTime, maxSec float64) {
	if p == nil || p.Out == nil || maxSec <= 0 {
		return
	}
	frac := gameTime / maxSec
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	pct := int(frac*100 + 0.5)
	if pct == p.lastPct {
		return
	}
	p.lastPct = pct
	w := p.BarWide
	if w < 10 {
		w = 10
	}
	filled := pct * w / 100
	if filled > w {
		filled = w
	}
	bar := strings.Repeat("=", filled) + strings.Repeat("-", w-filled)
	label := p.Label
	if label != "" {
		label += " "
	}
	fmt.Fprintf(p.Out, "\r%s[%s] %3d%% (%s / %s)",
		label, bar, pct, formatProgressTime(gameTime), formatProgressTime(maxSec))
}

// Finish prints a newline after the bar.
func (p *TerminalProgress) Finish() {
	if p != nil && p.Out != nil {
		fmt.Fprintln(p.Out)
	}
}

func formatProgressTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
