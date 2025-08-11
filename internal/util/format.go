package util

import (
	"fmt"
	"strings"
)

func HumanBytes(bytes int64) string {
	const unit = int64(1024)
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	div := unit
	exp := 0
	for exp < len(units)-1 && bytes >= div*unit {
		div *= unit
		exp++
	}

	v := float64(bytes) / float64(div)
	if bytes%div == 0 {
		return fmt.Sprintf("%.0f %s", v, units[exp])
	}
	return fmt.Sprintf("%.2f %s", v, units[exp])
}

func CommaInt(n int) string {
	return commaSigned(fmt.Sprintf("%d", n))
}

func CommaInt64(n int64) string {
	return commaSigned(fmt.Sprintf("%d", n))
}

func commaSigned(s string) string {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	out := commaUnsigned(s)
	if neg {
		return "-" + out
	}
	return out
}

func commaUnsigned(s string) string {
	if len(s) <= 3 {
		return s
	}
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	var b strings.Builder
	b.Grow(len(s) + (len(s)-1)/3)

	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func Colorize(s string, fg int, bg int) string {
	var parts []string
	if fg >= 0 {
		parts = append(parts, fmt.Sprintf("38;5;%d", fg))
	}
	if bg >= 0 {
		parts = append(parts, fmt.Sprintf("48;5;%d", bg))
	}
	if len(parts) == 0 {
		return s
	}
	return "\x1b[" + strings.Join(parts, ";") + "m" + s + "\x1b[0m"
}
