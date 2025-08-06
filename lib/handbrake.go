package lib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

type HandBrakeTranscoder struct {
	Files             []string
	FileListPath      string
	OutputSuffix      string
	Overwrite         bool
	Quality           int
	MinSavingsPercent int
	termWidth         int
	termMux           sync.RWMutex
}

type VideoInfo struct {
	Path     string
	IsHDR    bool
	Width    int
	Height   int
	Duration float64
}

type SkipInfo struct {
	Reason             string    `json:"reason"`
	Quality            int       `json:"quality"`
	Encoder            string    `json:"encoder"`
	Timestamp          time.Time `json:"timestamp"`
	OriginalSizeBytes  int64     `json:"original_size_bytes"`
	EstimatedSizeBytes int64     `json:"estimated_size_bytes"`
	RequiredSizeBytes  int64     `json:"required_size_bytes"`
}

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

func (t *HandBrakeTranscoder) checkHandBrakeCLI() error {
	_, err := exec.LookPath("HandBrakeCLI")
	if err != nil {
		return fmt.Errorf("HandBrakeCLI not found in PATH. Install with: brew install handbrake")
	}
	return nil
}

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

func (t *HandBrakeTranscoder) setupWinchHandler() {
	winchChan := make(chan os.Signal, 1)
	signal.Notify(winchChan, syscall.SIGWINCH)

	go func() {
		for range winchChan {
			t.initTerminalWidth()
		}
	}()
}

func (t *HandBrakeTranscoder) getTerminalWidth() int {
	t.termMux.RLock()
	defer t.termMux.RUnlock()
	return t.termWidth
}

func (t *HandBrakeTranscoder) transcodeFile(ctx context.Context, filePath string, hasVideoToolbox bool, fileNum, totalFiles int) error {
	slog.Info("Processing file", "current", fileNum, "total", totalFiles, "file", filepath.Base(filePath))

	finalOutputPath := t.generateOutputPath(filePath)
	if !t.Overwrite {
		if _, err := os.Stat(finalOutputPath); err == nil {
			slog.Info("Output file already exists, skipping", "file", finalOutputPath)
			return nil
		}
	}

	if t.MinSavingsPercent > 0 {
		if t.checkSkipFile(filePath) {
			slog.Info("Skipping media with skip file", "file", filepath.Base(filePath))
			return nil
		}
	}

	videoInfo, err := t.getVideoInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	originalFileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get original file info: %w", err)
	}
	originalFileSize := originalFileInfo.Size()

	if err := t.printMediaInfo(filePath); err != nil {
		slog.Warn("Failed to print media info", "file", filePath, "error", err)
	}

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

	if err := t.printMediaInfoWithRatio(finalOutputPath, originalFileSize); err != nil {
		slog.Warn("Failed to print media info for converted file", "file", finalOutputPath, "error", err)
	}

	slog.Info("Successfully transcoded", "file", filepath.Base(finalOutputPath))
	return nil
}

func (t *HandBrakeTranscoder) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	return filepath.Join(dir, base+t.OutputSuffix+".mkv")
}

func (t *HandBrakeTranscoder) getVideoInfo(filePath string) (*VideoInfo, error) {
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

	outputStr := string(output)
	isHDR := t.detectHDR(outputStr)
	duration := t.parseDuration(outputStr)

	return &VideoInfo{
		Path:     filePath,
		IsHDR:    isHDR,
		Duration: duration,
	}, nil
}

func (t *HandBrakeTranscoder) parseDuration(ffprobeOutput string) float64 {
	// Try to extract duration from format section first
	durationRegex := regexp.MustCompile(`"duration"\s*:\s*"([^"]+)"`)
	matches := durationRegex.FindStringSubmatch(ffprobeOutput)
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return duration
		}
	}

	// Fallback to a default if we can't parse
	slog.Warn("Could not parse video duration from ffprobe output, using default")
	return 3600.0 // 1 hour default
}

func (t *HandBrakeTranscoder) detectHDR(ffprobeOutput string) bool {
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
		"--verbose", "1",
	}

	encoder := t.selectEncoder(videoInfo, hasVideoToolbox)

	slog.Info("Using encoder", "encoder", encoder)
	args = append(args, "--encoder", encoder)

	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	slog.Debug("Executing HandBrakeCLI", "args", strings.Join(args, " "))

	return t.runHandBrakeCLI(ctx, args)
}

func (t *HandBrakeTranscoder) filterHandBrakeOutput(pipe io.ReadCloser) {
	defer pipe.Close()

	// Supported progress formats:
	// Encoding: task 1 of 1, 2.31 %
	// Encoding: task 1 of 1, 4.50 % (224.12 fps, avg 226.07 fps, ETA 00h02m48s)
	progressRegex := regexp.MustCompile(`Encoding: task \d+ of \d+, (\d+\.\d+) %(?:\s+\((\d+\.\d+) fps,.*ETA (\d+h\d+m\d+s)\))?`)

	buf := make([]byte, 1)
	var currentLine strings.Builder

	for {
		n, err := pipe.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		if n == 0 {
			continue
		}

		char := buf[0]

		if char == '\r' {
			line := currentLine.String()
			if matches := progressRegex.FindStringSubmatch(line); matches != nil {
				percent := matches[1]
				if len(matches) > 3 && matches[2] != "" {
					fps := matches[2]
					eta := matches[3]
					extraText := fmt.Sprintf(" (%s fps, ETA %s)", fps, eta)
					progressBar := t.createProgressBarWithText(percent, extraText)
					if progressBar != "" {
						fmt.Printf("\r%s %s%%%s", progressBar, percent, extraText)
					} else {
						fmt.Printf("\r%s%%%s", percent, extraText)
					}
				} else {
					progressBar := t.createProgressBar(percent)
					if progressBar != "" {
						fmt.Printf("\r%s %s%%", progressBar, percent)
					} else {
						fmt.Printf("\r%s%%", percent)
					}
				}
			}
			currentLine.Reset()
		} else if char == '\n' {
			line := currentLine.String()
			if strings.Contains(line, "Encode done!") {
				completionText := " - Encode done!"
				progressBar := t.createProgressBarWithText("100.0", completionText)
				if progressBar != "" {
					fmt.Printf("\r%s 100.0%%%s\n", progressBar, completionText)
				} else {
					fmt.Printf("\r100.0%%%s\n", completionText)
				}
			} else if strings.Contains(line, "ERROR") || strings.Contains(line, "WARNING") {
				fmt.Printf("\n%s\n", line)
			}
			currentLine.Reset()
		} else {
			currentLine.WriteByte(char)
		}
	}

	if currentLine.Len() > 0 {
		line := currentLine.String()
		fmt.Printf("%s\n", line)
	}
}

func (t *HandBrakeTranscoder) createProgressBar(percentStr string) string {
	return t.createProgressBarWithText(percentStr, "")
}

func (t *HandBrakeTranscoder) printMediaInfo(filePath string) error {
	return t.printMediaInfoWithRatio(filePath, 0)
}

func (t *HandBrakeTranscoder) printMediaInfoWithRatio(filePath string, originalFileSize int64) error {
	analyzer := NewMediaAnalyzer()
	mediaInfo, err := analyzer.AnalyzeFile(context.Background(), filePath)
	if err != nil {
		return err
	}

	var sizeStr string
	if mediaInfo.FileSize >= 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.1f GB", float64(mediaInfo.FileSize)/(1024*1024*1024))
	} else if mediaInfo.FileSize >= 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f MB", float64(mediaInfo.FileSize)/(1024*1024))
	} else {
		sizeStr = fmt.Sprintf("%.1f KB", float64(mediaInfo.FileSize)/1024)
	}

	durationStr := FormatDuration(mediaInfo.Duration)

	var bitrateStr string
	if mediaInfo.VideoBitrate >= 1000000 {
		bitrateStr = fmt.Sprintf("%.1f Mbps", float64(mediaInfo.VideoBitrate)/1000000)
	} else {
		bitrateStr = fmt.Sprintf("%.0f kbps", float64(mediaInfo.VideoBitrate)/1000)
	}

	logFields := []interface{}{
		"resolution", fmt.Sprintf("%dx%d", mediaInfo.VideoWidth, mediaInfo.VideoHeight),
		"duration", durationStr,
		"size", sizeStr,
		"bitrate", bitrateStr,
		"codec", mediaInfo.VideoCodec,
		"hdr", mediaInfo.HasDolbyVision || mediaInfo.ColorTransfer == "smpte2084" || mediaInfo.ColorSpace == "bt2020nc",
	}

	if originalFileSize > 0 {
		ratio := float64(mediaInfo.FileSize) / float64(originalFileSize)
		logFields = append(logFields, "size_ratio", fmt.Sprintf("%.1f%%", ratio*100))
	}

	slog.Info("Media info", logFields...)
	return nil
}

func (t *HandBrakeTranscoder) createProgressBarWithText(percentStr, extraText string) string {
	blocks := []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		return ""
	}

	termWidth := t.getTerminalWidth()

	// Calculate actual text width: " XX.X%" + extraText + space for leading space
	percentText := fmt.Sprintf(" %s%%", percentStr)
	totalTextWidth := len(percentText) + len(extraText)

	minBarWidth := 10
	minTotalWidth := minBarWidth + totalTextWidth + 2 // +2 for brackets
	if termWidth < minTotalWidth {
		return ""
	}

	barWidth := termWidth - totalTextWidth - 2 // -2 for brackets
	filled := int((percent / 100.0) * float64(barWidth))

	var bar strings.Builder
	bar.WriteRune('[')

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteRune(blocks[len(blocks)-1])
		} else if i == filled {
			partialProgress := (percent/100.0)*float64(barWidth) - float64(filled)
			if partialProgress > 0 {
				blockIndex := int(partialProgress * 8)
				if blockIndex >= len(blocks) {
					blockIndex = len(blocks) - 1
				}
				bar.WriteRune(blocks[blockIndex])
			} else {
				bar.WriteRune(' ')
			}
		} else {
			bar.WriteRune(' ')
		}
	}

	bar.WriteRune(']')
	return bar.String()
}

func (t *HandBrakeTranscoder) runHandBrakeCLI(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "HandBrakeCLI", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	go t.filterHandBrakeOutput(stdoutPipe)
	go t.filterHandBrakeOutput(stderrPipe)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start HandBrakeCLI: %w", err)
	}

	return cmd.Wait()
}

func (t *HandBrakeTranscoder) selectEncoder(videoInfo *VideoInfo, hasVideoToolbox bool) string {
	if hasVideoToolbox {
		if videoInfo.IsHDR {
			return "vt_h265_10bit"
		} else {
			return "vt_h265"
		}
	} else {
		if videoInfo.IsHDR {
			return "x265_10bit"
		} else {
			return "x265"
		}
	}
}

func (t *HandBrakeTranscoder) checkSkipFile(filePath string) bool {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	_, err := os.Stat(skipPath)
	return err == nil
}

func (t *HandBrakeTranscoder) checkSizeSavings(ctx context.Context, filePath string, originalFileSize int64, videoInfo *VideoInfo, hasVideoToolbox bool) (bool, error) {
	slog.Info("Estimating output size", "file", filepath.Base(filePath))

	estimatedSize, err := t.estimateOutputSize(ctx, filePath, videoInfo, hasVideoToolbox)
	if err != nil {
		return false, err
	}

	savingsBytes := originalFileSize - estimatedSize
	savingsPercent := float64(savingsBytes) / float64(originalFileSize) * 100

	if savingsPercent < float64(t.MinSavingsPercent) {
		encoder := t.selectEncoder(videoInfo, hasVideoToolbox)

		slog.Info("Skipping file, insufficient space savings",
			"file", filepath.Base(filePath),
			"estimated_savings", fmt.Sprintf("%.1f%%", savingsPercent),
			"required_savings", fmt.Sprintf("%d%%", t.MinSavingsPercent))
		if err := t.createSkipFile(filePath, "insufficient_savings", originalFileSize, estimatedSize, encoder); err != nil {
			slog.Warn("Failed to create skip file", "file", filePath, "error", err)
		}
		return true, nil
	}

	slog.Info("Size estimation passed threshold",
		"file", filepath.Base(filePath),
		"estimated_savings", fmt.Sprintf("%.1f%%", savingsPercent))
	return false, nil
}

func (t *HandBrakeTranscoder) createSkipFile(filePath string, reason string, originalSize, estimatedSize int64, encoder string) error {
	skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
	requiredSize := int64(float64(originalSize) * (100 - float64(t.MinSavingsPercent)) / 100)
	skipInfo := SkipInfo{
		Reason:             reason,
		Quality:            t.Quality,
		Encoder:            encoder,
		Timestamp:          time.Now(),
		OriginalSizeBytes:  originalSize,
		EstimatedSizeBytes: estimatedSize,
		RequiredSizeBytes:  requiredSize,
	}

	data, err := json.MarshalIndent(skipInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skip info: %w", err)
	}
	if err := os.WriteFile(skipPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skip file: %w", err)
	}
	return nil
}

func (t *HandBrakeTranscoder) estimateOutputSize(ctx context.Context, inputPath string, videoInfo *VideoInfo, hasVideoToolbox bool) (int64, error) {
	segmentDuration := 10.0                  // seconds
	positions := []float64{0.25, 0.50, 0.75} // 25%, 50%, 75% through video

	var totalSize int64
	var successfulSegments int

	for i, pos := range positions {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		startTime := videoInfo.Duration * pos
		testOutputPath := fmt.Sprintf("%s.size-test-%d.mkv", inputPath, i+1)

		// Clean up test file when done
		defer func(path string) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				slog.Warn("Failed to clean up test file", "file", path, "error", err)
			}
		}(testOutputPath)

		segmentSize, err := t.encodeSegment(ctx, inputPath, testOutputPath, startTime, segmentDuration, videoInfo, hasVideoToolbox)
		if err != nil {
			slog.Warn("Failed to encode test segment", "segment", i+1, "error", err)
			continue
		}

		totalSize += segmentSize
		successfulSegments++
	}

	if successfulSegments == 0 {
		return 0, fmt.Errorf("failed to encode any test segments")
	}

	avgBytesPerSecond := float64(totalSize) / (float64(successfulSegments) * segmentDuration)
	estimatedSize := int64(avgBytesPerSecond * videoInfo.Duration)

	slog.Debug("Size estimation",
		"successful_segments", successfulSegments,
		"avg_bytes_per_second", int64(avgBytesPerSecond),
		"estimated_size_bytes", estimatedSize)

	return estimatedSize, nil
}

func (t *HandBrakeTranscoder) encodeSegment(ctx context.Context, inputPath, outputPath string, startTime, duration float64, videoInfo *VideoInfo, hasVideoToolbox bool) (int64, error) {
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
		"--start-at", fmt.Sprintf("duration:%.0f", startTime),
		"--stop-at", fmt.Sprintf("duration:%.0f", duration),
		"--verbose", "1",
	}

	encoder := t.selectEncoder(videoInfo, hasVideoToolbox)
	args = append(args, "--encoder", encoder)
	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	if err := t.runHandBrakeCLI(ctx, args); err != nil {
		return 0, fmt.Errorf("HandBrakeCLI failed: %w", err)
	}

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat output file: %w", err)
	}
	return fileInfo.Size(), nil
}

func (t *HandBrakeTranscoder) formatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
}
