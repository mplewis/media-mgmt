package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var transcodeCmd = &cobra.Command{
	Use:   "transcode",
	Short: "Transcode video files using HandBrake with VideoToolbox acceleration",
	Long: `Convert one or more video files using HandBrakeCLI with VideoToolbox hardware acceleration.
Automatically detects HDR content and applies appropriate encoding settings.
Uses H.265 10-bit for HDR content and H.265 8-bit for SDR content.
Files are transcoded in-place using temporary .tmp files for safety.`,
	RunE: runTranscode,
}

var (
	transcodeFiles        []string
	transcodeFileListPath string
	transcodeOutputSuffix string
	transcodeOverwrite    bool
	transcodeVerbose      bool
	transcodeQuality      int
	transcodeMinSavings   int
)

func init() {
	transcodeCmd.Flags().StringSliceVarP(&transcodeFiles, "files", "f", []string{}, "Comma-separated list of video files to transcode")
	transcodeCmd.Flags().StringVarP(&transcodeFileListPath, "file-list", "l", "", "Path to text file containing list of video files (one per line)")
	transcodeCmd.Flags().StringVarP(&transcodeOutputSuffix, "suffix", "s", "-optimized", "Output file suffix")
	transcodeCmd.Flags().BoolVarP(&transcodeOverwrite, "overwrite", "o", false, "Overwrite existing output files")
	transcodeCmd.Flags().BoolVarP(&transcodeVerbose, "verbose", "v", false, "Enable verbose logging")
	transcodeCmd.Flags().IntVarP(&transcodeQuality, "quality", "q", 70, "Video quality (0-100, higher is better quality)")
	transcodeCmd.Flags().IntVarP(&transcodeMinSavings, "min-savings-percent", "m", 20, "Minimum space savings percentage required (0 disables)")
}

func runTranscode(cmd *cobra.Command, args []string) error {
	setupLogging(transcodeVerbose)

	if len(transcodeFiles) == 0 && transcodeFileListPath == "" {
		return fmt.Errorf("must specify either --files or --file-list")
	}

	slog.Info("Starting video transcoding with HandBrake",
		"files_count", len(transcodeFiles),
		"file_list", transcodeFileListPath,
		"suffix", transcodeOutputSuffix,
		"overwrite", transcodeOverwrite)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Received signal, shutting down gracefully", "signal", sig)
		cancel()
	}()

	transcoder := &lib.HandBrakeTranscoder{
		Files:             transcodeFiles,
		FileListPath:      transcodeFileListPath,
		OutputSuffix:      transcodeOutputSuffix,
		Overwrite:         transcodeOverwrite,
		Quality:           transcodeQuality,
		MinSavingsPercent: transcodeMinSavings,
	}

	if err := transcoder.Run(ctx); err != nil {
		if ctx.Err() == context.Canceled {
			slog.Info("Transcoding was cancelled by user")
			return nil
		}
		return fmt.Errorf("transcoding failed: %w", err)
	}

	slog.Info("Transcoding completed successfully")
	return nil
}
