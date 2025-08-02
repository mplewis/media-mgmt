package lib

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type App struct {
	InputDir    string
	OutputDir   string
	Parallelism int
	NoCache     bool
}

func (a *App) Run(ctx context.Context) error {
	slog.Debug("Application starting", "config", fmt.Sprintf("%+v", a))

	if err := CheckFFprobeAvailable(); err != nil {
		return err
	}

	scanner := NewFileScanner(a.InputDir)
	videoFiles, err := scanner.ScanVideoFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan video files: %w", err)
	}

	if len(videoFiles) == 0 {
		slog.Warn("No video files found in directory", "dir", a.InputDir)
		return nil
	}

	var processor *MediaProcessor
	if a.NoCache {
		slog.Debug("Caching disabled, using direct processor")
		processor = NewMediaProcessor(a.Parallelism)
	} else {
		cache := NewCacheManager(a.OutputDir)
		if err := cache.EnsureCacheDir(); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		if err := cache.CleanOldCache(60 * 24 * time.Hour); err != nil {
			slog.Warn("Failed to clean old cache files", "error", err)
		}

		slog.Debug("Caching enabled", "cacheDir", cache.CacheDir)
		processor = NewMediaProcessorWithCache(a.Parallelism, cache)
	}

	mediaInfos, err := processor.ProcessFiles(ctx, videoFiles)
	if err != nil {
		return fmt.Errorf("failed to process video files: %w", err)
	}

	if len(mediaInfos) == 0 {
		slog.Warn("No files were successfully analyzed")
		return nil
	}

	reporter := NewReportGenerator(a.OutputDir)
	if err := reporter.GenerateAllReports(mediaInfos); err != nil {
		return fmt.Errorf("failed to generate reports: %w", err)
	}

	return nil
}