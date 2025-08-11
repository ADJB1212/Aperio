package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Bar renders a single-line progress bar with Unicode block characters,
// including partial cell resolution using left-to-right fractional blocks.
//
// Example output:
//
//	[██████████▊                    ]  36% (36/100)
//
// It writes updates using a carriage return prefix so the line is updated in place.
// Call Finish() to print a trailing newline when complete.
type Bar struct {
	// Out is the writer to render to (defaults to os.Stderr if nil).
	Out io.Writer

	// Width is the number of visual cells inside the brackets.
	// Defaults to 40 when zero or negative.
	Width int

	// ShowPercent controls whether to print a percentage after the bar.
	ShowPercent bool

	// ShowCounts controls whether to print "(current/total)" after the percentage.
	ShowCounts bool

	// LeftBracket and RightBracket customize the surrounding delimiters.
	// Defaults: "[" and "]".
	LeftBracket  string
	RightBracket string

	// Fill is the full-cell block (default: '█').
	Fill rune

	// PartialRunes are 1/8th-resolution left-to-right fractional blocks.
	// Defaults: ▏ ▎ ▍ ▌ ▋ ▊ ▉
	PartialRunes []rune

	// Optional label appended after percentage/counts (e.g., file name).
	Label string

	// internal
	lastLen int
}

// New creates a progress bar configured with sensible defaults.
func New(out io.Writer, width int) *Bar {
	if out == nil {
		out = os.Stderr
	}
	if width <= 0 {
		width = 40
	}
	return &Bar{
		Out:          out,
		Width:        width,
		ShowPercent:  true,
		ShowCounts:   true,
		LeftBracket:  "[",
		RightBracket: "]",
		Fill:         '█',
		PartialRunes: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
	}
}

// SetLabel sets an optional label to be printed after the suffix (percentage/counts).
func (b *Bar) SetLabel(s string) { b.Label = s }

// Render updates the progress bar to represent current/total.
// If total <= 0, it is treated as 0 to avoid division by zero.
func (b *Bar) Render(current, total int) {
	if b.Out == nil {
		b.Out = os.Stderr
	}
	width := b.Width
	if width <= 0 {
		width = 40
	}
	if total < 0 {
		total = 0
	}
	if current < 0 {
		current = 0
	}
	if total > 0 && current > total {
		current = total
	}

	frac := 0.0
	if total > 0 {
		frac = float64(current) / float64(total)
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
	}

	cells := float64(width) * frac
	full := int(cells)
	rem := cells - float64(full)

	// Determine partial cell (1..7), clamp to available width.
	partialIdx := int(rem * 8.0)
	if partialIdx > 7 {
		partialIdx = 7
	}
	// When fraction is exactly an integer multiple of cell, skip partial.
	hasPartial := partialIdx > 0 && full < width

	// Build the bar
	var sb strings.Builder
	sb.Grow(2 + width + 32) // brackets + cells + suffix buffer

	sb.WriteString("\r")
	sb.WriteString(b.LeftBracket)
	if full > 0 {
		sb.WriteString(strings.Repeat(string(b.Fill), full))
	}
	if hasPartial {
		// partialIdx in range 1..7 maps to PartialRunes[0..6]
		idx := partialIdx - 1
		if idx >= len(b.PartialRunes) {
			idx = len(b.PartialRunes) - 1
		}
		if idx >= 0 {
			sb.WriteRune(b.PartialRunes[idx])
			full++
		}
	}
	if full < width {
		sb.WriteString(strings.Repeat(" ", width-full))
	}
	sb.WriteString(b.RightBracket)
	sb.WriteByte(' ')

	// Suffix: percentage and counts
	if b.ShowPercent {
		pct := 0
		if total > 0 {
			pct = int(frac*100.0 + 0.5) // round to nearest
		}
		sb.WriteString(fmt.Sprintf("%3d%%", pct))
		if b.ShowCounts {
			sb.WriteByte(' ')
		}
	}
	if b.ShowCounts {
		sb.WriteString(fmt.Sprintf("(%d/%d)", current, total))
	}

	if b.Label != "" {
		sb.WriteByte(' ')
		sb.WriteString(b.Label)
	}

	// If the new line is shorter than the previous render, pad with spaces to clear.
	line := sb.String()
	if pad := b.lastLen - (len(line) - 1); pad > 0 { // exclude the leading \r from length
		line += strings.Repeat(" ", pad)
	}

	_, _ = io.WriteString(b.Out, line)
	b.lastLen = len(line) - 1 // minus the leading \r
}

// Finish finalizes the progress bar by writing a trailing newline.
func (b *Bar) Finish() {
	if b.Out == nil {
		b.Out = os.Stderr
	}
	_, _ = io.WriteString(b.Out, "\n")
	b.lastLen = 0
}
