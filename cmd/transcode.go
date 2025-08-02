package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var transcodeCmd = &cobra.Command{
	Use:   "transcode",
	Short: "Transcode video files using FFmpeg presets",
	Long: `Convert one or more video files based on YAML-defined presets.
Uses hardware acceleration when available (NVENC, VideoToolbox).
Automatically detects HDR content and applies appropriate presets.`,
	RunE: runTranscode,
}

var (
	presetsFile  string
	files        []string
	fileListPath string
	outputSuffix string
	transcodeVerbose bool
)

func init() {
	transcodeCmd.Flags().StringVarP(&presetsFile, "presets", "p", "", "Path to presets YAML file (required)")
	transcodeCmd.Flags().StringSliceVarP(&files, "files", "f", []string{}, "Comma-separated list of video files to transcode")
	transcodeCmd.Flags().StringVarP(&fileListPath, "file-list", "l", "", "Path to text file containing list of video files (one per line)")
	transcodeCmd.Flags().StringVarP(&outputSuffix, "suffix", "s", "-optimized-{preset}", "Output file suffix pattern")
	transcodeCmd.Flags().BoolVarP(&transcodeVerbose, "verbose", "v", false, "Enable verbose logging")

	transcodeCmd.MarkFlagRequired("presets")
}

func runTranscode(cmd *cobra.Command, args []string) error {
	setupLogging(transcodeVerbose)

	if len(files) == 0 && fileListPath == "" {
		return fmt.Errorf("must specify either --files or --file-list")
	}

	slog.Info("Starting video transcoding",
		"presets", presetsFile,
		"files_count", len(files),
		"file_list", fileListPath,
		"suffix", outputSuffix)

	ctx := context.Background()

	transcoder := &lib.Transcoder{
		PresetsFile:  presetsFile,
		Files:        files,
		FileListPath: fileListPath,
		OutputSuffix: outputSuffix,
	}

	if err := transcoder.Run(ctx); err != nil {
		return fmt.Errorf("transcoding failed: %w", err)
	}

	slog.Info("Transcoding completed successfully")
	return nil
}