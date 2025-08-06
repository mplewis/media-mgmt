package handbrake

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"media-mgmt/lib"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// HandBrakeTranscoder manages video transcoding operations using HandBrakeCLI.
// Supports batch processing, size estimation, and intelligent skipping of files
// that don't meet minimum space savings requirements.
type HandBrakeTranscoder struct {
	Files             []string // List of files to transcode
	FileListPath      string   // Path to text file containing file list
	OutputSuffix      string   // Suffix for output files (e.g., "-optimized")
	Overwrite         bool     // Whether to overwrite existing output files
	Quality           int      // Video quality setting (0-100, higher is better)
	MinSavingsPercent int      // Minimum space savings required (0 disables)
	termWidth         int      // Current terminal width for progress bars
	termMux           sync.RWMutex // Mutex for terminal width access
}

// Run executes the transcoding process for all configured files.
// Handles setup, file processing, and graceful shutdown on context cancellation.
// Returns an error if HandBrakeCLI is unavailable or if critical failures occur.
func (t *HandBrakeTranscoder) Run(ctx context.Context) error {
	if err := t.checkHandBrakeCLI(); err != nil {
		return fmt.Errorf("HandBrakeCLI not available: %w", err)
	}

	t.initTerminalWidth()
	t.setupWinchHandler()

	hasVideoToolbox, err := t.detectVideoToolbox()
	if err != nil {
		slog.Warn("Failed to detect VideoToolbox", "error", err)
		hasVideoToolbox = false
	}
	slog.Info("VideoToolbox support", "available", hasVideoToolbox)

	files, err := t.getFileList()
	if err != nil {
		return fmt.Errorf("failed to get file list: %w", err)
	}

	slog.Info("Processing files", "count", len(files))

	for i, file := range files {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, stopping file processing")
			return ctx.Err()
		default:
		}

		fileNum := i + 1
		totalFiles := len(files)
		if err := t.transcodeFile(ctx, file, hasVideoToolbox, fileNum, totalFiles); err != nil {
			slog.Error("Failed to transcode file", "file", file, "error", err)
			if ctx.Err() != nil {
				slog.Info("Context cancelled, stopping file processing")
				return ctx.Err()
			}
			continue
		}
	}

	return nil
}

// transcodeFile processes a single video file through the complete transcoding pipeline.
// Handles output path checking, skip file validation, size estimation, and actual transcoding.
// Returns an error if any step fails, or nil if the file is successfully processed or skipped.
func (t *HandBrakeTranscoder) transcodeFile(ctx context.Context, filePath string, hasVideoToolbox bool, fileNum, totalFiles int) error {
	slog.Info("Processing file", "current", fileNum, "total", totalFiles, "file", filepath.Base(filePath))

	finalOutputPath := t.generateOutputPath(filePath)
	if !t.Overwrite {
		if _, err := os.Stat(finalOutputPath); err == nil {
			slog.Info("Output file already exists, skipping", "file", finalOutputPath)
			return nil
		}
	}

	// Check for existing skip file first
	if t.MinSavingsPercent > 0 {
		if t.checkSkipFile(filePath) {
			slog.Info("Skipping media with skip file", "file", filepath.Base(filePath))
			return nil
		}
	}

	videoInfo, err := lib.GetVideoInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	originalFileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get original file info: %w", err)
	}
	originalFileSize := originalFileInfo.Size()

	if err := lib.PrintMediaInfo(filePath); err != nil {
		slog.Warn("Failed to print media info", "file", filePath, "error", err)
	}

	// Perform size estimation if minimum savings threshold is set
	if t.MinSavingsPercent > 0 {
		shouldSkip, err := t.checkSizeSavings(ctx, filePath, originalFileSize, videoInfo, hasVideoToolbox)
		if err != nil {
			slog.Warn("Size check failed, proceeding with full encode", "file", filePath, "error", err)
		} else if shouldSkip {
			return nil
		}
	}

	inProgressPath := finalOutputPath + ".tmp"
	outputDir := filepath.Dir(inProgressPath)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cleanupFile := true
	defer func() {
		if cleanupFile {
			if err := os.Remove(inProgressPath); err != nil {
				slog.Warn("Failed to clean up unfinished file", "file", inProgressPath, "error", err)
			} else {
				slog.Info("Cleaned up unfinished file", "file", inProgressPath)
			}
		}
	}()

	if err := t.executeTranscode(ctx, filePath, inProgressPath, videoInfo, hasVideoToolbox); err != nil {
		return fmt.Errorf("failed to execute transcode: %w", err)
	}

	if err := os.Rename(inProgressPath, finalOutputPath); err != nil {
		return fmt.Errorf("failed to move temp file to final location: %w", err)
	}
	cleanupFile = false

	if err := lib.PrintMediaInfoWithRatio(finalOutputPath, originalFileSize); err != nil {
		slog.Warn("Failed to print media info for converted file", "file", finalOutputPath, "error", err)
	}

	slog.Info("Successfully transcoded", "file", filepath.Base(finalOutputPath))
	return nil
}

// checkHandBrakeCLI verifies that HandBrakeCLI is available in the system PATH.
// Returns an error with installation instructions if HandBrakeCLI is not found.
func (t *HandBrakeTranscoder) checkHandBrakeCLI() error {
	_, err := exec.LookPath("HandBrakeCLI")
	if err != nil {
		return fmt.Errorf("HandBrakeCLI not found in PATH. Install with: brew install handbrake")
	}
	return nil
}

// detectVideoToolbox checks if VideoToolbox hardware acceleration is available.
// Only available on macOS systems with compatible hardware.
// Returns true if VideoToolbox encoders are detected in HandBrakeCLI help output.
func (t *HandBrakeTranscoder) detectVideoToolbox() (bool, error) {
	cmd := exec.Command("uname", "-s")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(string(output)) != "Darwin" {
		return false, nil
	}

	cmd = exec.Command("HandBrakeCLI", "--help")
	output, err = cmd.Output()
	if err != nil {
		return false, err
	}

	helpText := string(output)
	return strings.Contains(helpText, "vt_h265") || strings.Contains(helpText, "VideoToolbox"), nil
}

// getFileList combines files from direct specification and file list into a single slice.
// Processes the FileListPath if specified, filtering out comments and empty lines.
// Returns the combined list of files to process, or an error if file reading fails.
func (t *HandBrakeTranscoder) getFileList() ([]string, error) {
	var files []string

	files = append(files, t.Files...)
	if t.FileListPath != "" {
		file, err := os.Open(t.FileListPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file list: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				files = append(files, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read file list: %w", err)
		}
	}

	return files, nil
}

// initTerminalWidth determines and stores the current terminal width.
// Uses a default of 80 columns if terminal detection fails.
// Thread-safe access via mutex for concurrent progress bar rendering.
func (t *HandBrakeTranscoder) initTerminalWidth() {
	width := 80
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			width = w
		}
	}
	t.termMux.Lock()
	t.termWidth = width
	t.termMux.Unlock()
}

// setupWinchHandler installs a signal handler for terminal resize events (SIGWINCH).
// Automatically updates terminal width when the terminal is resized.
// Runs in a background goroutine for the lifetime of the transcoder.
func (t *HandBrakeTranscoder) setupWinchHandler() {
	winchChan := make(chan os.Signal, 1)
	signal.Notify(winchChan, syscall.SIGWINCH)

	go func() {
		for range winchChan {
			t.initTerminalWidth()
		}
	}()
}

// getTerminalWidth returns the current terminal width in a thread-safe manner.
// Used by progress bar rendering functions to determine available display space.
func (t *HandBrakeTranscoder) getTerminalWidth() int {
	t.termMux.RLock()
	defer t.termMux.RUnlock()
	return t.termWidth
}