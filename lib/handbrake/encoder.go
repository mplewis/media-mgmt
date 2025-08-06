package handbrake

import (
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"path/filepath"
	"strings"
)

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

func (t *HandBrakeTranscoder) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	return filepath.Join(dir, base+t.OutputSuffix+".mkv")
}

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