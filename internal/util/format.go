package util

import (
	"fmt"
	"strings"
)

// HumanBytes returns a human-readable, IEC (base-1024) formatted size.
//
// Examples:
//
//	HumanBytes(0)            => "0 B"
//	HumanBytes(1023)         => "1023 B"
//	HumanBytes(1024)         => "1 KiB"
//	HumanBytes(10*1024)      => "10 KiB"
//	HumanBytes(1536)         => "1.50 KiB"
//	HumanBytes(5*1024*1024)  => "5 MiB"
//
// Exact multiples are rendered without decimals for compactness.
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

// CommaInt formats an integer with thousands separators.
//
// Examples:
//
//	CommaInt(0)         => "0"
//	CommaInt(12)        => "12"
//	CommaInt(1234)      => "1,234"
//	CommaInt(-9876543)  => "-9,876,543"
func CommaInt(n int) string {
	return commaSigned(fmt.Sprintf("%d", n))
}

// CommaInt64 formats a 64-bit integer with thousands separators.
//
// Examples:
//
//	CommaInt64(1234567890)  => "1,234,567,890"
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
