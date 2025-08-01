package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type MediaProcessor struct {
	analyzer    *MediaAnalyzer
	parallelism int
}

func NewMediaProcessor(parallelism int) *MediaProcessor {
	return &MediaProcessor{
		analyzer:    NewMediaAnalyzer(),
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
	
	// Create channels for work distribution
	jobs := make(chan string, len(filePaths))
	results := make(chan *MediaInfo, len(filePaths))
	errors := make(chan error, len(filePaths))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < mp.parallelism; i++ {
		wg.Add(1)
		go mp.worker(ctx, &wg, jobs, results, errors)
	}
	
	// Send all file paths to jobs channel
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
	
	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()
	
	// Collect results
	var mediaInfos []*MediaInfo
	var errs []error
	
	// Since each worker sends to both results and errors channels,
	// we need to read from both channels for each file
	for i := 0; i < len(filePaths); i++ {
		result := <-results
		err := <-errors
		
		if result != nil {
			mediaInfos = append(mediaInfos, result)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	
	slog.Info("Parallel media analysis completed",
		"processedFiles", len(mediaInfos),
		"errors", len(errs))
	
	// Log errors but don't fail the entire operation
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
				return // Channel closed, no more work
			}
			
			mediaInfo, err := mp.analyzer.AnalyzeFile(ctx, filePath)
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