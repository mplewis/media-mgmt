package lib

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{0, "0:00"},
		{30, "0:30"},
		{60, "1:00"},
		{90, "1:30"},
		{90.7, "1:30"},
		{3599, "59:59"},
		{3600, "1:00:00"},
		{7975, "2:12:55"},   // 2*3600 + 12*60 + 55
		{36303, "10:05:03"}, // 10*3600 + 5*60 + 3
	}

	for _, tt := range tests {
		result := FormatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}
