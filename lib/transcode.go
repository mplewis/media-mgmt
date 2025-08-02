package lib

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type PresetConfig struct {
	Presets []Preset `yaml:"presets"`
}

type Preset struct {
	Name       string      `yaml:"name"`
	Condition  string      `yaml:"condition"`
	Container  string      `yaml:"container"`
	Video      VideoConfig `yaml:"video"`
	Audio      string      `yaml:"audio"`
	Subtitles  string      `yaml:"subtitles"`
}

type VideoConfig struct {
	Codec    string `yaml:"codec"`
	BitDepth int    `yaml:"bit_depth"`
	Quality  string `yaml:"quality"`
	HWAccel  bool   `yaml:"hw_accel"`
}

type Transcoder struct {
	PresetsFile  string
	Files        []string
	FileListPath string
	OutputSuffix string
	presets      []Preset
	hwAccel      HardwareAcceleration
}

type HardwareAcceleration struct {
	Available bool
	Type      string // "nvenc", "videotoolbox", "vaapi", etc.
	Encoder   string // "h264_nvenc", "h264_videotoolbox", etc.
}

type VideoInfo struct {
	Path     string
	IsHDR    bool
	Width    int
	Height   int
	Duration float64
}

func (t *Transcoder) Run(ctx context.Context) error {
	if err := t.loadPresets(); err != nil {
		return fmt.Errorf("failed to load presets: %w", err)
	}

	if err := t.detectHardwareAcceleration(); err != nil {
		slog.Warn("Failed to detect hardware acceleration", "error", err)
	} else {
		slog.Info("Hardware acceleration detected", "type", t.hwAccel.Type, "available", t.hwAccel.Available)
	}

	files, err := t.getFileList()
	if err != nil {
		return fmt.Errorf("failed to get file list: %w", err)
	}

	slog.Info("Processing files", "count", len(files))

	for _, file := range files {
		if err := t.transcodeFile(ctx, file); err != nil {
			slog.Error("Failed to transcode file", "file", file, "error", err)
			continue
		}
	}

	return nil
}

func (t *Transcoder) loadPresets() error {
	data, err := os.ReadFile(t.PresetsFile)
	if err != nil {
		return fmt.Errorf("failed to read presets file: %w", err)
	}

	var config PresetConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse presets YAML: %w", err)
	}

	t.presets = config.Presets
	slog.Info("Loaded presets", "count", len(t.presets))

	return nil
}

func (t *Transcoder) getFileList() ([]string, error) {
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

func (t *Transcoder) detectHardwareAcceleration() error {
	t.hwAccel = HardwareAcceleration{Available: false}

	// Check for ffmpeg availability first
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}

	// Detect based on platform
	switch runtime.GOOS {
	case "darwin":
		// Check for VideoToolbox
		if t.testEncoder("h264_videotoolbox") {
			t.hwAccel = HardwareAcceleration{
				Available: true,
				Type:      "videotoolbox",
				Encoder:   "h265_videotoolbox",
			}
		}
	case "linux", "windows":
		// Check for NVENC first (more common)
		if t.testEncoder("h264_nvenc") {
			t.hwAccel = HardwareAcceleration{
				Available: true,
				Type:      "nvenc",
				Encoder:   "hevc_nvenc",
			}
		} else if t.testEncoder("h264_vaapi") {
			t.hwAccel = HardwareAcceleration{
				Available: true,
				Type:      "vaapi",
				Encoder:   "hevc_vaapi",
			}
		}
	}

	return nil
}

func (t *Transcoder) testEncoder(encoder string) bool {
	cmd := exec.Command("ffmpeg", "-hide_banner", "-f", "lavfi", "-i", "testsrc2=duration=1:size=320x240:rate=1", "-c:v", encoder, "-f", "null", "-")
	return cmd.Run() == nil
}

func (t *Transcoder) transcodeFile(ctx context.Context, filePath string) error {
	slog.Info("Processing file", "path", filePath)

	// Get video info (including HDR detection)
	videoInfo, err := t.getVideoInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	// Find matching preset
	preset := t.findMatchingPreset(videoInfo)
	if preset == nil {
		return fmt.Errorf("no matching preset found for file")
	}

	slog.Info("Using preset", "name", preset.Name, "file", filePath)

	// Generate output filename
	outputPath := t.generateOutputPath(filePath, preset.Name)

	// Build and execute FFmpeg command
	if err := t.executeTranscode(ctx, filePath, outputPath, preset, videoInfo); err != nil {
		return fmt.Errorf("failed to execute transcode: %w", err)
	}

	slog.Info("Successfully transcoded", "input", filePath, "output", outputPath)
	return nil
}

func (t *Transcoder) getVideoInfo(filePath string) (*VideoInfo, error) {
	// Use ffprobe to get video information
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

	// For now, return basic info. HDR detection will be implemented separately
	return &VideoInfo{
		Path:  filePath,
		IsHDR: t.detectHDR(string(output)),
	}, nil
}

func (t *Transcoder) detectHDR(ffprobeOutput string) bool {
	// Look for HDR indicators in ffprobe output
	hdrIndicators := []string{
		"bt2020",
		"smpte2084",
		"arib-std-b67",
		"color_primaries=bt2020",
		"color_transfer=smpte2084",
	}

	output := strings.ToLower(ffprobeOutput)
	for _, indicator := range hdrIndicators {
		if strings.Contains(output, indicator) {
			return true
		}
	}
	return false
}

func (t *Transcoder) findMatchingPreset(videoInfo *VideoInfo) *Preset {
	for _, preset := range t.presets {
		switch preset.Condition {
		case "hdr":
			if videoInfo.IsHDR {
				return &preset
			}
		case "always":
			return &preset
		}
	}
	return nil
}

func (t *Transcoder) generateOutputPath(inputPath, presetName string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	// Default suffix pattern
	suffix := t.OutputSuffix
	if suffix == "" {
		suffix = "-optimized-{preset}"
	}

	// Replace placeholder
	suffix = strings.ReplaceAll(suffix, "{preset}", presetName)

	return filepath.Join(dir, base+suffix+".mkv")
}

func (t *Transcoder) executeTranscode(ctx context.Context, inputPath, outputPath string, preset *Preset, videoInfo *VideoInfo) error {
	args := []string{
		"-i", inputPath,
		"-y", // Overwrite output file
		"-map", "0", // Map all streams
	}

	// Video encoding
	if preset.Video.HWAccel && t.hwAccel.Available {
		args = append(args, "-c:v", t.hwAccel.Encoder)
	} else {
		// Software encoding
		if preset.Video.Codec == "h265" {
			args = append(args, "-c:v", "libx265")
		} else {
			args = append(args, "-c:v", "libx264")
		}
	}

	// Quality setting
	if strings.HasPrefix(preset.Video.Quality, "cq:") {
		cq := strings.TrimPrefix(preset.Video.Quality, "cq:")
		if t.hwAccel.Available {
			args = append(args, "-cq", cq)
		} else {
			args = append(args, "-crf", cq)
		}
	}

	// Bit depth (for software encoding)
	if !t.hwAccel.Available && preset.Video.BitDepth == 10 {
		args = append(args, "-pix_fmt", "yuv420p10le")
	}

	// Audio and subtitle passthrough
	if preset.Audio == "passthrough" {
		args = append(args, "-c:a", "copy")
	}
	if preset.Subtitles == "passthrough" {
		args = append(args, "-c:s", "copy")
	}

	args = append(args, outputPath)

	slog.Debug("Executing FFmpeg", "args", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}