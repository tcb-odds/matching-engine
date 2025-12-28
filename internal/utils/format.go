package utils

import (
	"fmt"
	"time"
)

func FormatDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	d %= time.Minute
	seconds := d / time.Second

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 || result != "" {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 || result != "" {
		result += fmt.Sprintf("%dm ", minutes)
	}
	result += fmt.Sprintf("%ds", seconds)

	return result
}
