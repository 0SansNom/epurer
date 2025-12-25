package utils

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

// FormatBytes converts bytes to human-readable format (e.g., "1.2 GB")
func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}
	return humanize.Bytes(uint64(bytes))
}

// FormatCount formats numbers with commas (e.g., "1,234")
func FormatCount(count int) string {
	return humanize.Comma(int64(count))
}

// FormatDuration formats a duration in human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// FormatPercentage formats a percentage with 1 decimal place
func FormatPercentage(value, total int64) string {
	if total == 0 {
		return "0%"
	}
	pct := float64(value) / float64(total) * 100
	return fmt.Sprintf("%.1f%%", pct)
}
