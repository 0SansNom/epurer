package utils

import (
	"testing"
	"time"
)

// =============================================================================
// FormatBytes Tests
// =============================================================================

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Negative bytes", -100, "0 B"},
		{"Small bytes", 500, "500 B"},
		{"One kilobyte", 1024, "1.0 kB"},
		{"Kilobytes", 1536, "1.5 kB"},
		{"One megabyte", 1024 * 1024, "1.0 MB"},
		{"Megabytes", 1572864, "1.6 MB"},
		{"One gigabyte", 1024 * 1024 * 1024, "1.1 GB"},
		{"Gigabytes", 5368709120, "5.4 GB"},
		{"Terabyte", 1099511627776, "1.1 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// FormatCount Tests
// =============================================================================

func TestFormatCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{"Zero", 0, "0"},
		{"Small number", 42, "42"},
		{"Hundreds", 999, "999"},
		{"Thousands", 1234, "1,234"},
		{"Ten thousands", 12345, "12,345"},
		{"Hundred thousands", 123456, "123,456"},
		{"Millions", 1234567, "1,234,567"},
		{"Large number", 1234567890, "1,234,567,890"},
		{"Negative", -1234, "-1,234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCount(tt.count)
			if result != tt.expected {
				t.Errorf("FormatCount(%d) = %q, want %q", tt.count, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// FormatDuration Tests
// =============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Milliseconds", 500 * time.Millisecond, "500ms"},
		{"Sub-millisecond", 100 * time.Microsecond, "0ms"},
		{"One second", 1 * time.Second, "1.0s"},
		{"Seconds", 5500 * time.Millisecond, "5.5s"},
		{"Under a minute", 45 * time.Second, "45.0s"},
		{"One minute", 60 * time.Second, "1.0m"},
		{"Minutes", 90 * time.Second, "1.5m"},
		{"Several minutes", 5 * time.Minute, "5.0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// FormatPercentage Tests
// =============================================================================

func TestFormatPercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		total    int64
		expected string
	}{
		{"Zero total", 50, 0, "0%"},
		{"Zero value", 0, 100, "0.0%"},
		{"Half", 50, 100, "50.0%"},
		{"Full", 100, 100, "100.0%"},
		{"Quarter", 25, 100, "25.0%"},
		{"Third", 1, 3, "33.3%"},
		{"Two thirds", 2, 3, "66.7%"},
		{"Over 100%", 150, 100, "150.0%"},
		{"Small percentage", 1, 1000, "0.1%"},
		{"Large numbers", 5000000, 10000000, "50.0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPercentage(tt.value, tt.total)
			if result != tt.expected {
				t.Errorf("FormatPercentage(%d, %d) = %q, want %q", tt.value, tt.total, result, tt.expected)
			}
		})
	}
}
