package lib

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

type MediaProcessor struct {
	analyzer    *MediaAnalyzer
	cache       *CacheManager
	parallelism int
}

func NewMediaProcessor(parallelism int) *MediaProcessor {
	return &MediaProcessor{
		analyzer:    NewMediaAnalyzer(),
		parallelism: parallelism,
	}
}

func NewMediaProcessorWithCache(parallelism int, cache *CacheManager) *MediaProcessor {
	return &MediaProcessor{
		analyzer:    NewMediaAnalyzer(),
		cache:       cache,
		parallelism: parallelism,
	}
}

// ProcessFiles analyzes multiple video files in parallel
func (mp *MediaProcessor) ProcessFiles(ctx context.Context, filePaths []string) ([]*MediaInfo, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	slog.Info("Starting parallel media analysis",
		"totalFiles", len(filePaths),
		"workers", mp.parallelism)

	bar := progressbar.NewOptions(len(filePaths),
		progressbar.OptionSetDescription("Analyzing files"),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	jobs := make(chan string, len(filePaths))
	results := make(chan *MediaInfo, len(filePaths))
	errors := make(chan error, len(filePaths))

	var wg sync.WaitGroup
	for i := 0; i < mp.parallelism; i++ {
		wg.Add(1)
		go mp.worker(ctx, &wg, jobs, results, errors)
	}

	go func() {
		defer close(jobs)
		for _, filePath := range filePaths {
			select {
			case jobs <- filePath:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	var mediaInfos []*MediaInfo
	var errs []error

	for i := 0; i < len(filePaths); i++ {
		result := <-results
		err := <-errors

		if result != nil {
			mediaInfos = append(mediaInfos, result)
		}
		if err != nil {
			errs = append(errs, err)
		}

		bar.Add(1)
	}

	bar.Finish()

	slog.Info("Parallel media analysis completed",
		"processedFiles", len(mediaInfos),
		"errors", len(errs))

	for _, err := range errs {
		slog.Warn("File analysis failed", "error", err)
	}

	return mediaInfos, nil
}

func (mp *MediaProcessor) worker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan string, results chan<- *MediaInfo, errors chan<- error) {
	defer wg.Done()

	for {
		select {
		case filePath, ok := <-jobs:
			if !ok {
				return
			}

			var mediaInfo *MediaInfo
			var err error

			if mp.cache != nil {
				fileInfo, statErr := os.Stat(filePath)
				if statErr != nil {
					errors <- fmt.Errorf("failed to stat file %s: %w", filePath, statErr)
					results <- nil
					continue
				}

				hasCache, cachedInfo, cacheErr := mp.cache.HasValidCache(filePath, fileInfo)
				if cacheErr != nil {
					slog.Warn("Cache check failed, will analyze fresh", "file", filePath, "error", cacheErr)
				}

				if hasCache && cachedInfo != nil {
					mediaInfo = cachedInfo
					slog.Debug("Using cached analysis", "file", filePath)
				} else {
					mediaInfo, err = mp.analyzer.AnalyzeFile(ctx, filePath)
					if err == nil && mediaInfo != nil {
						if saveErr := mp.cache.SaveCache(filePath, fileInfo, mediaInfo); saveErr != nil {
							slog.Warn("Failed to save analysis to cache", "file", filePath, "error", saveErr)
						}
					}
				}
			} else {
				mediaInfo, err = mp.analyzer.AnalyzeFile(ctx, filePath)
			}

			if err != nil {
				errors <- fmt.Errorf("failed to analyze %s: %w", filePath, err)
				results <- nil
			} else {
				results <- mediaInfo
				errors <- nil
			}

		case <-ctx.Done():
			return
		}
	}
}
