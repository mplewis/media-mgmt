package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

func main() {
	var (
		inputDir    = flag.String("input", "", "Input directory to scan for video files")
		outputDir   = flag.String("output", "", "Output directory for reports")
		parallelism = flag.Int("parallelism", runtime.NumCPU(), "Number of parallel workers")
		verbose     = flag.Bool("v", false, "Enable verbose logging")
	)
	flag.Parse()

	setupLogging(*verbose)

	if *inputDir == "" {
		slog.Error("Input directory is required")
		flag.Usage()
		os.Exit(1)
	}

	if *outputDir == "" {
		slog.Error("Output directory is required")
		flag.Usage()
		os.Exit(1)
	}

	slog.Info("Starting media analysis",
		"input", *inputDir,
		"output", *outputDir,
		"parallelism", *parallelism)

	ctx := context.Background()
	
	app := &App{
		InputDir:    *inputDir,
		OutputDir:   *outputDir,
		Parallelism: *parallelism,
	}

	if err := app.Run(ctx); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Analysis completed successfully")
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
		handler = NewColorHandler(os.Stderr, opts)
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

type App struct {
	InputDir    string
	OutputDir   string
	Parallelism int
}

func (a *App) Run(ctx context.Context) error {
	slog.Debug("Application starting", "config", fmt.Sprintf("%+v", a))
	
	// Check if ffprobe is available
	if err := CheckFFprobeAvailable(); err != nil {
		return err
	}
	
	// Scan for video files
	scanner := NewFileScanner(a.InputDir)
	videoFiles, err := scanner.ScanVideoFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan video files: %w", err)
	}
	
	if len(videoFiles) == 0 {
		slog.Warn("No video files found in directory", "dir", a.InputDir)
		return nil
	}
	
	// Process files in parallel
	processor := NewMediaProcessor(a.Parallelism)
	mediaInfos, err := processor.ProcessFiles(ctx, videoFiles)
	if err != nil {
		return fmt.Errorf("failed to process video files: %w", err)
	}
	
	if len(mediaInfos) == 0 {
		slog.Warn("No files were successfully analyzed")
		return nil
	}
	
	// Generate reports
	reporter := NewReportGenerator(a.OutputDir)
	if err := reporter.GenerateAllReports(mediaInfos); err != nil {
		return fmt.Errorf("failed to generate reports: %w", err)
	}
	
	return nil
}