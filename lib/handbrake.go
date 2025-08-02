package lib

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type HandBrakeTranscoder struct {
	Files        []string
	FileListPath string
	OutputSuffix string
	Overwrite    bool
}

type VideoInfo struct {
	Path     string
	IsHDR    bool
	Width    int
	Height   int
	Duration float64
}

func (t *HandBrakeTranscoder) Run(ctx context.Context) error {
	// Check for HandBrakeCLI availability
	if err := t.checkHandBrakeCLI(); err != nil {
		return fmt.Errorf("HandBrakeCLI not available: %w", err)
	}

	// Check for VideoToolbox support
	hasVideoToolbox, err := t.detectVideoToolbox()
	if err != nil {
		slog.Warn("Failed to detect VideoToolbox", "error", err)
		hasVideoToolbox = false
	}
	slog.Info("VideoToolbox support", "available", hasVideoToolbox)

	// Get file list
	files, err := t.getFileList()
	if err != nil {
		return fmt.Errorf("failed to get file list: %w", err)
	}

	slog.Info("Processing files", "count", len(files))

	for _, file := range files {
		if err := t.transcodeFile(ctx, file, hasVideoToolbox); err != nil {
			slog.Error("Failed to transcode file", "file", file, "error", err)
			continue
		}
	}

	return nil
}

func (t *HandBrakeTranscoder) checkHandBrakeCLI() error {
	_, err := exec.LookPath("HandBrakeCLI")
	if err != nil {
		return fmt.Errorf("HandBrakeCLI not found in PATH. Install with: brew install handbrake")
	}
	return nil
}

func (t *HandBrakeTranscoder) detectVideoToolbox() (bool, error) {
	// Check if we're on macOS
	cmd := exec.Command("uname", "-s")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(string(output)) != "Darwin" {
		return false, nil
	}

	// Test VideoToolbox encoder availability with HandBrake
	cmd = exec.Command("HandBrakeCLI", "--help")
	output, err = cmd.Output()
	if err != nil {
		return false, err
	}

	// Look for VideoToolbox encoder in help output
	helpText := string(output)
	return strings.Contains(helpText, "vt_h265") || strings.Contains(helpText, "VideoToolbox"), nil
}

func (t *HandBrakeTranscoder) getFileList() ([]string, error) {
	var files []string

	// Add files from --files flag
	files = append(files, t.Files...)

	// Add files from --file-list
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

func (t *HandBrakeTranscoder) transcodeFile(ctx context.Context, filePath string, hasVideoToolbox bool) error {
	slog.Info("Processing file", "path", filePath)

	// Generate output filename
	outputPath := t.generateOutputPath(filePath)

	// Check if output already exists
	if !t.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			slog.Info("Output file already exists, skipping", "output", outputPath)
			return nil
		}
	}

	// Get video info (including HDR detection)
	videoInfo, err := t.getVideoInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	slog.Info("Video info", "hdr", videoInfo.IsHDR, "file", filePath)

	// Build and execute HandBrake command
	if err := t.executeTranscode(ctx, filePath, outputPath, videoInfo, hasVideoToolbox); err != nil {
		return fmt.Errorf("failed to execute transcode: %w", err)
	}

	slog.Info("Successfully transcoded", "input", filePath, "output", outputPath)
	return nil
}

func (t *HandBrakeTranscoder) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	// Use MKV container as specified in the original spec
	return filepath.Join(dir, base+t.OutputSuffix+".mkv")
}

func (t *HandBrakeTranscoder) getVideoInfo(filePath string) (*VideoInfo, error) {
	// Use ffprobe to get video information for HDR detection
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",
		"-print_format", "json",
		"-show_format", 
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Basic HDR detection based on common indicators
	isHDR := t.detectHDR(string(output))

	return &VideoInfo{
		Path:  filePath,
		IsHDR: isHDR,
	}, nil
}

func (t *HandBrakeTranscoder) detectHDR(ffprobeOutput string) bool {
	// Look for HDR indicators in ffprobe output
	hdrIndicators := []string{
		"bt2020",
		"smpte2084",
		"arib-std-b67",
		"color_primaries=bt2020",
		"color_transfer=smpte2084",
		"yuv420p10le",
		"yuv422p10le",
		"yuv444p10le",
	}

	output := strings.ToLower(ffprobeOutput)
	for _, indicator := range hdrIndicators {
		if strings.Contains(output, indicator) {
			return true
		}
	}
	return false
}

func (t *HandBrakeTranscoder) executeTranscode(ctx context.Context, inputPath, outputPath string, videoInfo *VideoInfo, hasVideoToolbox bool) error {
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
	}

	// Choose encoder based on VideoToolbox availability and HDR content
	if hasVideoToolbox {
		if videoInfo.IsHDR {
			args = append(args, "--encoder", "vt_h265_10bit")
		} else {
			args = append(args, "--encoder", "vt_h265")
		}
	} else {
		// Fallback to software encoder
		if videoInfo.IsHDR {
			args = append(args, "--encoder", "x265_10bit")
		} else {
			args = append(args, "--encoder", "x265")
		}
	}

	// Set quality (CQ 80 as specified)
	args = append(args, "--quality", "80")

	// Copy all audio and subtitle tracks (passthrough)
	args = append(args, "--all-audio", "--all-subtitles")

	// Add format for MKV container
	args = append(args, "--format", "av_mkv")

	slog.Debug("Executing HandBrakeCLI", "args", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "HandBrakeCLI", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}