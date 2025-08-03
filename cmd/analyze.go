package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze video files and generate reports",
	Long: `Scan a directory for video files, analyze their metadata using ffprobe,
and generate comprehensive reports in multiple formats (HTML, JSON, CSV, Markdown).

The HTML report includes an interactive React-based interface with sorting,
filtering, and pagination capabilities.`,
	RunE: runAnalyze,
}

var (
	inputDir    string
	outputDir   string
	parallelism int
	verbose     bool
	noCache     bool
)

func init() {
	analyzeCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Input directory to scan for video files (required)")
	analyzeCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for reports (required)")
	analyzeCmd.Flags().IntVarP(&parallelism, "parallelism", "p", runtime.NumCPU(), "Number of parallel workers")
	analyzeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	analyzeCmd.Flags().Bool("no-cache", false, "Disable caching of analysis results")

	// Mark required flags
	analyzeCmd.MarkFlagRequired("input")
	analyzeCmd.MarkFlagRequired("output")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Get no-cache flag
	noCache, _ = cmd.Flags().GetBool("no-cache")

	setupLogging(verbose)

	slog.Info("Starting media analysis",
		"input", inputDir,
		"output", outputDir,
		"parallelism", parallelism)

	ctx := context.Background()

	app := &lib.App{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		Parallelism: parallelism,
		NoCache:     noCache,
	}

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	slog.Info("Analysis completed successfully")
	return nil
}

func setupLogging(verbose bool) {
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToLower(envLevel) {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn", "warning":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if isTerminal() {
		handler = lib.NewColorHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func isTerminal() bool {
	fileInfo, _ := os.Stderr.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
