package handbrake

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkipInfo contains metadata about why a file was skipped during transcoding.
// Stored as JSON in .skip files to prevent re-processing files that don't meet savings criteria.
type SkipInfo struct {
	Reason             string    `json:"reason"`               // Reason for skipping (e.g., "insufficient_savings")
	Quality            int       `json:"quality"`              // Quality setting used for estimation
	Encoder            string    `json:"encoder"`              // Encoder that would have been used
	Timestamp          time.Time `json:"timestamp"`            // When the skip decision was made
	OriginalSizeBytes  int64     `json:"original_size_bytes"`  // Original file size in bytes
	EstimatedSizeBytes int64     `json:"estimated_size_bytes"` // Estimated output size in bytes
	RequiredSizeBytes  int64     `json:"required_size_bytes"`  // Minimum size required to meet savings threshold
}

// checkSkipFile determines if a skip file exists for the given input file.
// Returns true if a .skip file is found, indicating the file should be skipped.
func (t *HandBrakeTranscoder) checkSkipFile(filePath string) bool {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	_, err := os.Stat(skipPath)
	return err == nil
}

// checkSizeSavings estimates output size and determines if the file should be skipped.
// Performs size estimation and compares against the minimum savings threshold.
// Returns true if the file should be skipped (insufficient savings), false to proceed.
func (t *HandBrakeTranscoder) checkSizeSavings(ctx context.Context, filePath string, originalFileSize int64, videoInfo *lib.VideoInfo, hasVideoToolbox bool) (bool, error) {
	slog.Info("Estimating output size", "file", filepath.Base(filePath))

	estimatedSize, err := t.estimateOutputSize(ctx, filePath, videoInfo, hasVideoToolbox)
	if err != nil {
		return false, err
	}

	sizeRatio := float64(estimatedSize) / float64(originalFileSize)
	
	if sizeRatio > t.MaxSizeRatio {
		encoder := t.selectEncoder(videoInfo, hasVideoToolbox)

		slog.Info("Skipping file, insufficient space savings",
			"file", filepath.Base(filePath),
			"size_ratio", fmt.Sprintf("%.1f%%", sizeRatio*100),
			"max_size_ratio", fmt.Sprintf("%.1f%%", t.MaxSizeRatio*100))
		if err := t.createSkipFile(filePath, "insufficient_savings", originalFileSize, estimatedSize, encoder); err != nil {
			slog.Warn("Failed to create skip file", "file", filePath, "error", err)
		}
		return true, nil
	}

	slog.Info("Size estimation passed threshold",
		"file", filepath.Base(filePath),
		"size_ratio", fmt.Sprintf("%.1f%%", sizeRatio*100),
		"max_size_ratio", fmt.Sprintf("%.1f%%", t.MaxSizeRatio*100))
	return false, nil
}

// createSkipFile generates a .skip file with metadata about why the file was skipped.
// Creates a JSON file containing size estimates, encoder settings, and skip reasons.
// This prevents re-processing the file in future runs.
func (t *HandBrakeTranscoder) createSkipFile(filePath string, reason string, originalSize, estimatedSize int64, encoder string) error {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	requiredSize := int64(float64(originalSize) * t.MaxSizeRatio)
	skipInfo := SkipInfo{
		Reason:             reason,
		Quality:            t.Quality,
		Encoder:            encoder,
		Timestamp:          time.Now(),
		OriginalSizeBytes:  originalSize,
		EstimatedSizeBytes: estimatedSize,
		RequiredSizeBytes:  requiredSize,
	}

	data, err := json.MarshalIndent(skipInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skip info: %w", err)
	}
	if err := os.WriteFile(skipPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skip file: %w", err)
	}
	return nil
}

// estimateOutputSize calculates approximate output file size by encoding test segments.
// Encodes 3 segments of 10 seconds each at 25%, 50%, and 75% through the video.
// Averages the results and extrapolates to the full video duration.
func (t *HandBrakeTranscoder) estimateOutputSize(ctx context.Context, inputPath string, videoInfo *lib.VideoInfo, hasVideoToolbox bool) (int64, error) {
	segmentDuration := 10.0                  // seconds
	positions := []float64{0.25, 0.50, 0.75} // 25%, 50%, 75% through video

	var totalSize int64
	var successfulSegments int

	for i, pos := range positions {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		startTime := videoInfo.Duration * pos
		testOutputPath := fmt.Sprintf("%s.size-test-%d.mkv", inputPath, i+1)

		defer func(path string) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				slog.Warn("Failed to clean up test file", "file", path, "error", err)
			}
		}(testOutputPath)

		segmentSize, err := t.encodeSegment(ctx, inputPath, testOutputPath, startTime, segmentDuration, videoInfo, hasVideoToolbox)
		if err != nil {
			slog.Warn("Failed to encode test segment", "segment", i+1, "error", err)
			continue
		}

		totalSize += segmentSize
		successfulSegments++
	}

	if successfulSegments == 0 {
		return 0, fmt.Errorf("failed to encode any test segments")
	}

	avgBytesPerSecond := float64(totalSize) / (float64(successfulSegments) * segmentDuration)
	estimatedSize := int64(avgBytesPerSecond * videoInfo.Duration)

	slog.Debug("Size estimation",
		"successful_segments", successfulSegments,
		"avg_bytes_per_second", int64(avgBytesPerSecond),
		"estimated_size_bytes", estimatedSize)

	return estimatedSize, nil
}

// encodeSegment encodes a small portion of video for size estimation purposes.
// Uses the same encoder and quality settings as the full transcode.
// Returns the size of the encoded segment in bytes, or an error if encoding fails.
func (t *HandBrakeTranscoder) encodeSegment(ctx context.Context, inputPath, outputPath string, startTime, duration float64, videoInfo *lib.VideoInfo, hasVideoToolbox bool) (int64, error) {
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
		"--start-at", fmt.Sprintf("duration:%.0f", startTime),
		"--stop-at", fmt.Sprintf("duration:%.0f", duration),
		"--verbose", "1",
	}

	encoder := t.selectEncoder(videoInfo, hasVideoToolbox)
	args = append(args, "--encoder", encoder)
	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	if err := t.runHandBrakeCLI(ctx, args); err != nil {
		return 0, fmt.Errorf("HandBrakeCLI failed: %w", err)
	}

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat output file: %w", err)
	}
	return fileInfo.Size(), nil
}
