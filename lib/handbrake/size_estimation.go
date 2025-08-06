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

type SkipInfo struct {
	Reason             string    `json:"reason"`
	Quality            int       `json:"quality"`
	Encoder            string    `json:"encoder"`
	Timestamp          time.Time `json:"timestamp"`
	OriginalSizeBytes  int64     `json:"original_size_bytes"`
	EstimatedSizeBytes int64     `json:"estimated_size_bytes"`
	RequiredSizeBytes  int64     `json:"required_size_bytes"`
}

func (t *HandBrakeTranscoder) checkSkipFile(filePath string) bool {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	_, err := os.Stat(skipPath)
	return err == nil
}

func (t *HandBrakeTranscoder) checkSizeSavings(ctx context.Context, filePath string, originalFileSize int64, videoInfo *lib.VideoInfo, hasVideoToolbox bool) (bool, error) {
	slog.Info("Estimating output size", "file", filepath.Base(filePath))

	estimatedSize, err := t.estimateOutputSize(ctx, filePath, videoInfo, hasVideoToolbox)
	if err != nil {
		return false, err
	}

	savingsBytes := originalFileSize - estimatedSize
	savingsPercent := float64(savingsBytes) / float64(originalFileSize) * 100

	if savingsPercent < float64(t.MinSavingsPercent) {
		encoder := t.selectEncoder(videoInfo, hasVideoToolbox)

		slog.Info("Skipping file, insufficient space savings",
			"file", filepath.Base(filePath),
			"estimated_savings", fmt.Sprintf("%.1f%%", savingsPercent),
			"required_savings", fmt.Sprintf("%d%%", t.MinSavingsPercent))
		if err := t.createSkipFile(filePath, "insufficient_savings", originalFileSize, estimatedSize, encoder); err != nil {
			slog.Warn("Failed to create skip file", "file", filePath, "error", err)
		}
		return true, nil
	}

	slog.Info("Size estimation passed threshold",
		"file", filepath.Base(filePath),
		"estimated_savings", fmt.Sprintf("%.1f%%", savingsPercent))
	return false, nil
}

func (t *HandBrakeTranscoder) createSkipFile(filePath string, reason string, originalSize, estimatedSize int64, encoder string) error {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	requiredSize := int64(float64(originalSize) * (100 - float64(t.MinSavingsPercent)) / 100)
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

		// Clean up test file when done
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