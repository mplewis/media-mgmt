package lib

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/term"
)

type HandBrakeTranscoder struct {
	Files        []string
	FileListPath string
	OutputSuffix string
	Overwrite    bool
	Quality      int
	WriteInPlace bool
	tempDir      string
	termWidth    int
	termMux      sync.RWMutex
}

type VideoInfo struct {
	Path     string
	IsHDR    bool
	Width    int
	Height   int
	Duration float64
}

func (t *HandBrakeTranscoder) Run(ctx context.Context) error {
	if err := t.checkHandBrakeCLI(); err != nil {
		return fmt.Errorf("HandBrakeCLI not available: %w", err)
	}

	if !t.WriteInPlace {
		if err := t.setupTempDir(); err != nil {
			return fmt.Errorf("failed to setup temp directory: %w", err)
		}
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

	var outputPath string
	if t.WriteInPlace {
		outputPath = finalOutputPath
	} else {
		outputPath = t.generateTempOutputPath(filePath)
	}
	outputDir := filepath.Dir(outputPath)

	if t.WriteInPlace {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	var cleanupFile bool
	if t.WriteInPlace {
		cleanupFile = true
		defer func() {
			if cleanupFile {
				if err := os.Remove(outputPath); err != nil {
					slog.Warn("Failed to clean up unfinished file", "file", outputPath, "error", err)
				} else {
					slog.Info("Cleaned up unfinished file", "file", outputPath)
				}
			}
		}()
	}

	transcodeErr := t.executeTranscode(ctx, filePath, outputPath, videoInfo, hasVideoToolbox, t.WriteInPlace)

	if t.WriteInPlace {
		if transcodeErr != nil {
			cleanupFile = true
		} else {
			cleanupFile = false
		}
	}

	if transcodeErr != nil {
		return fmt.Errorf("failed to execute transcode: %w", transcodeErr)
	}

	if t.WriteInPlace {
		dirFile, err := os.Open(outputDir)
		if err != nil {
			return fmt.Errorf("failed to open directory for sync: %w", err)
		}
		defer dirFile.Close()

		if err := dirFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync directory to disk: %w", err)
		}

		slog.Debug("Directory synced to disk", "dir", outputDir)

	} else {
		if err := t.copyToFinalDestination(outputPath, finalOutputPath); err != nil {
			return fmt.Errorf("failed to copy to final destination: %w", err)
		}
	}

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

	isHDR := t.detectHDR(string(output))

	return &VideoInfo{
		Path:  filePath,
		IsHDR: isHDR,
	}, nil
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

func (t *HandBrakeTranscoder) executeTranscode(ctx context.Context, inputPath, outputPath string, videoInfo *VideoInfo, hasVideoToolbox bool, writeInPlace bool) error {
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
		"--verbose", "1",
	}

	var encoder string
	if hasVideoToolbox {
		if videoInfo.IsHDR {
			encoder = "vt_h265_10bit"
		} else {
			encoder = "vt_h265"
		}
	} else {
		if videoInfo.IsHDR {
			encoder = "x265_10bit"
		} else {
			encoder = "x265"
		}
	}

	slog.Info("Using encoder", "encoder", encoder)
	args = append(args, "--encoder", encoder)

	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	slog.Debug("Executing HandBrakeCLI", "args", strings.Join(args, " "))

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

func (t *HandBrakeTranscoder) setupTempDir() error {
	tempDir, err := os.MkdirTemp("", "handbrake-transcode-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	t.tempDir = tempDir
	slog.Debug("Created temp directory", "path", tempDir)

	runtime.SetFinalizer(t, (*HandBrakeTranscoder).cleanup)
	return nil
}

func (t *HandBrakeTranscoder) cleanup() {
	if t.tempDir != "" {
		slog.Debug("Cleaning up temp directory", "path", t.tempDir)
		if err := os.RemoveAll(t.tempDir); err != nil {
			slog.Warn("Failed to clean up temp directory", "path", t.tempDir, "error", err)
		}
	}
}

func (t *HandBrakeTranscoder) generateTempOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	return filepath.Join(t.tempDir, base+t.OutputSuffix+".mkv")
}

func (t *HandBrakeTranscoder) copyToFinalDestination(tempPath, finalPath string) error {
	slog.Debug("Copying to final destination", "temp", tempPath, "final", finalPath)

	src, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	dst, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(finalPath)
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
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

	hours := int(mediaInfo.Duration) / 3600
	minutes := int(mediaInfo.Duration) / 60
	seconds := int(mediaInfo.Duration) % 60
	var durationStr string
	if hours > 0 {
		durationStr = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	} else {
		durationStr = fmt.Sprintf("%d:%02d", minutes, seconds)
	}

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
