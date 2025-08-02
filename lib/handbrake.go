package lib

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type HandBrakeTranscoder struct {
	Files        []string
	FileListPath string
	OutputSuffix string
	Overwrite    bool
	Quality      int
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

func (t *HandBrakeTranscoder) transcodeFile(ctx context.Context, filePath string, hasVideoToolbox bool) error {
	slog.Info("Processing file", "path", filePath)

	outputPath := t.generateOutputPath(filePath)
	if !t.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			slog.Info("Output file already exists, skipping", "output", outputPath)
			return nil
		}
	}

	videoInfo, err := t.getVideoInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	slog.Info("Video info", "hdr", videoInfo.IsHDR, "file", filePath)

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

func (t *HandBrakeTranscoder) executeTranscode(ctx context.Context, inputPath, outputPath string, videoInfo *VideoInfo, hasVideoToolbox bool) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	transcodeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-sigChan
		slog.Info("Interrupt received, cleaning up...", "output", outputPath)
		cancel()
		
		if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to remove incomplete file", "file", outputPath, "error", err)
		} else {
			slog.Info("Removed incomplete file", "file", outputPath)
		}
		
		os.Exit(1)
	}()
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
	}

	if hasVideoToolbox {
		if videoInfo.IsHDR {
			args = append(args, "--encoder", "vt_h265_10bit")
		} else {
			args = append(args, "--encoder", "vt_h265")
		}
	} else {
		if videoInfo.IsHDR {
			args = append(args, "--encoder", "x265_10bit")
		} else {
			args = append(args, "--encoder", "x265")
		}
	}

	args = append(args, "--quality", fmt.Sprintf("%d", t.Quality))
	args = append(args, "--all-audio", "--all-subtitles")
	args = append(args, "--format", "av_mkv")

	slog.Debug("Executing HandBrakeCLI", "args", strings.Join(args, " "))

	cmd := exec.CommandContext(transcodeCtx, "HandBrakeCLI", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	
	if err != nil {
		if _, statErr := os.Stat(outputPath); statErr == nil {
			slog.Info("Removing incomplete output file due to error", "file", outputPath)
			if removeErr := os.Remove(outputPath); removeErr != nil {
				slog.Warn("Failed to remove incomplete file", "file", outputPath, "error", removeErr)
			}
		}
	}
	
	return err
}
