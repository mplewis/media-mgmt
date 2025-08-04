package lib

import "fmt"

// FormatDuration formats a duration in seconds to a human-readable string.
// Returns format like "2:12:55" for hours:minutes:seconds or "12:55" for minutes:seconds
func FormatDuration(seconds float64) string {
	totalSeconds := int(seconds)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	secs := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}
