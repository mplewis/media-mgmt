package main

import (
	"context"
	"testing"
)

func TestMediaProcessor_ProcessFiles(t *testing.T) {
	processor := NewMediaProcessor(2)

	// Empty file list should return empty results
	ctx := context.Background()
	results, err := processor.ProcessFiles(ctx, []string{})
	if err != nil {
		t.Errorf("ProcessFiles with empty list should not error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty list, got %d", len(results))
	}
}

func TestMediaProcessor_WorkerCount(t *testing.T) {
	testCases := []int{1, 2, 4, 8}

	for _, parallelism := range testCases {
		processor := NewMediaProcessor(parallelism)
		if processor.parallelism != parallelism {
			t.Errorf("Expected parallelism %d, got %d", parallelism, processor.parallelism)
		}
	}
}
