package handbrake

import (
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"path/filepath"
	"strings"
)

// selectEncoder chooses the appropriate HandBrake encoder based on video characteristics and hardware support.
// Uses VideoToolbox hardware encoders on macOS when available, falls back to software encoders.
// Selects 10-bit encoders for HDR content, 8-bit for SDR content.
func (t *HandBrakeTranscoder) selectEncoder(videoInfo *lib.VideoInfo, hasVideoToolbox bool) string {
	if hasVideoToolbox {
		if videoInfo.IsHDR {
			return "vt_h265_10bit"
		} else {
			return "vt_h265"
		}
	} else {
		if videoInfo.IsHDR {
			return "x265_10bit"
		} else {
			return "x265"
		}
	}
}

// generateOutputPath creates the output file path by adding the configured suffix.
// Replaces the original extension with .mkv and inserts the suffix before the extension.
// Example: "movie.mp4" with suffix "-optimized" becomes "movie-optimized.mkv"
func (t *HandBrakeTranscoder) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	return filepath.Join(dir, base+t.OutputSuffix+".mkv")
}

// executeTranscode performs the actual video transcoding using HandBrakeCLI.
// Builds command arguments, selects encoder, and executes the transcoding process.
// Returns an error if the transcoding process fails.
func (t *HandBrakeTranscoder) executeTranscode(ctx context.Context, inputPath, outputPath string, videoInfo *lib.VideoInfo, hasVideoToolbox bool) error {
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
		"--verbose", "1",
	}

	encoder := t.selectEncoder(videoInfo, hasVideoToolbox)

	slog.Info("Using encoder", "encoder", encoder)
	args = append(args, "--encoder", encoder)

	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	slog.Debug("Executing HandBrakeCLI", "args", strings.Join(args, " "))

	return t.runHandBrakeCLI(ctx, args)
}