package lib

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"log/slog"
	"fmt"
)

type VideoInfo struct {
	Path     string
	IsHDR    bool
	Width    int
	Height   int
	Duration float64
}

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
	duration := parseDuration(outputStr)

	return &VideoInfo{
		Path:     filePath,
		IsHDR:    isHDR,
		Duration: duration,
	}, nil
}

func parseDuration(ffprobeOutput string) float64 {
	// Try to extract duration from format section first
	durationRegex := regexp.MustCompile(`"duration"\s*:\s*"([^"]+)"`)
	matches := durationRegex.FindStringSubmatch(ffprobeOutput)
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return duration
		}
	}

	// Fallback to a default if we can't parse
	slog.Warn("Could not parse video duration from ffprobe output, using default")
	return 3600.0 // 1 hour default
}

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