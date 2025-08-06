package lib

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	durationRegex = regexp.MustCompile(`"duration"\s*:\s*"([^"]+)"`)
)

// VideoInfo contains metadata about a video file extracted from ffprobe.
type VideoInfo struct {
	Path     string  // Full path to the video file
	IsHDR    bool    // Whether the video contains HDR content
	Width    int     // Video width in pixels
	Height   int     // Video height in pixels
	Duration float64 // Duration in seconds
}

// GetVideoInfo extracts video metadata from a file using ffprobe.
// Returns VideoInfo with duration and HDR detection, or an error if ffprobe fails.
func GetVideoInfo(filePath string) (*VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	outputStr := string(output)
	isHDR := DetectHDR(outputStr)
	duration, err := parseDuration(outputStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video duration: %w", err)
	}

	return &VideoInfo{
		Path:     filePath,
		IsHDR:    isHDR,
		Duration: duration,
	}, nil
}

// parseDuration extracts the duration from ffprobe JSON output.
// Returns the duration in seconds, or an error if parsing fails.
func parseDuration(ffprobeOutput string) (float64, error) {
	// Try to extract duration from format section first
	matches := durationRegex.FindStringSubmatch(ffprobeOutput)
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return duration, nil
		}
	}

	return 0, fmt.Errorf("could not parse video duration from ffprobe output")
}

// DetectHDR analyzes ffprobe output to determine if video contains HDR content.
// Checks for various HDR indicators including color primaries, transfer functions, and pixel formats.
// Returns true if any HDR indicators are found (case-insensitive), false otherwise.
func DetectHDR(ffprobeOutput string) bool {
	hdrIndicators := []string{
		"bt2020",
		"smpte2084",
		"arib-std-b67",
		"color_primaries=bt2020",
		"color_transfer=smpte2084",
		"yuv420p10le",
		"yuv422p10le",
		"yuv444p10le",
	}

	output := strings.ToLower(ffprobeOutput)
	for _, indicator := range hdrIndicators {
		if strings.Contains(output, indicator) {
			return true
		}
	}
	return false
}