package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Bar struct {
	Out          io.Writer
	Width        int
	ShowPercent  bool
	ShowCounts   bool
	LeftBracket  string
	RightBracket string
	Fill         rune
	PartialRunes []rune
	Label        string
	lastLen      int
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
	partialIdx := min(int(rem*8.0), 7)
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
