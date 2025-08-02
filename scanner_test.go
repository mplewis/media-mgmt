package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileScanner_ScanVideoFiles(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		path    string
		isVideo bool
	}{
		{"video1.mp4", true},
		{"video2.mkv", true},
		{"video3.avi", true},
		{"document.txt", false},
		{"image.jpg", false},
		{"subdir/video4.mov", true},
		{"subdir/audio.mp3", false},
		{"subdir/deep/video5.webm", true},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tempDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test scanning
	scanner := NewFileScanner(tempDir)
	ctx := context.Background()

	videoFiles, err := scanner.ScanVideoFiles(ctx)
	if err != nil {
		t.Fatalf("ScanVideoFiles failed: %v", err)
	}

	// Count expected video files
	expectedCount := 0
	for _, tf := range testFiles {
		if tf.isVideo {
			expectedCount++
		}
	}

	if len(videoFiles) != expectedCount {
		t.Errorf("Expected %d video files, got %d", expectedCount, len(videoFiles))
	}

	// Verify all found files are actually video files
	for _, file := range videoFiles {
		ext := strings.ToLower(filepath.Ext(file))
		if !videoExtensions[ext] {
			t.Errorf("Non-video file found: %s", file)
		}
	}
}

func TestVideoExtensions(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.mp4", true},
		{"test.MP4", true}, // Extension check should be case-insensitive
		{"test.mkv", true},
		{"test.avi", true},
		{"test.txt", false},
		{"test.jpg", false},
		{"test", false},
		{"test.mov", true},
	}

	for _, tc := range testCases {
		ext := strings.ToLower(filepath.Ext(tc.filename))
		result := videoExtensions[ext]
		if result != tc.expected {
			t.Errorf("For %s: expected %v, got %v", tc.filename, tc.expected, result)
		}
	}
}
