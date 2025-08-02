package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

type ReportGenerator struct {
	outputDir string
}

func NewReportGenerator(outputDir string) *ReportGenerator {
	return &ReportGenerator{outputDir: outputDir}
}

// GenerateAllReports creates all report formats
func (rg *ReportGenerator) GenerateAllReports(mediaInfos []*MediaInfo) error {
	if err := os.MkdirAll(rg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	slog.Info("Generating reports", "outputDir", rg.outputDir, "mediaCount", len(mediaInfos))

	timestamp := time.Now().Format("20060102_150405")

	csvFilename := fmt.Sprintf("media_report_%s.csv", timestamp)
	if err := rg.GenerateCSV(mediaInfos, csvFilename); err != nil {
		return fmt.Errorf("failed to generate CSV report: %w", err)
	}

	jsonFilename := fmt.Sprintf("media_report_%s.json", timestamp)
	if err := rg.GenerateJSON(mediaInfos, jsonFilename); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	mdFilename := fmt.Sprintf("media_report_%s.md", timestamp)
	if err := rg.GenerateMarkdown(mediaInfos, mdFilename); err != nil {
		return fmt.Errorf("failed to generate Markdown report: %w", err)
	}

	htmlFilename := fmt.Sprintf("media_report_%s.html", timestamp)
	if err := rg.GenerateHTML(mediaInfos, htmlFilename); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	slog.Info("All reports generated successfully", "paths", []string{
		filepath.Join(rg.outputDir, csvFilename),
		filepath.Join(rg.outputDir, jsonFilename),
		filepath.Join(rg.outputDir, mdFilename),
		filepath.Join(rg.outputDir, htmlFilename),
	})
	return nil
}

// GenerateCSV creates a CSV report
func (rg *ReportGenerator) GenerateCSV(mediaInfos []*MediaInfo, filename string) error {
	filePath := filepath.Join(rg.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"File Path", "File Size (MB)", "Duration (min)", "Video Codec",
		"Video Bitrate (kbps)", "Resolution", "Audio Tracks", "Subtitle Tracks",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Sort by file path for consistent output
	sort.Slice(mediaInfos, func(i, j int) bool {
		return mediaInfos[i].FilePath < mediaInfos[j].FilePath
	})

	// Write data rows
	for _, info := range mediaInfos {
		row := []string{
			info.FilePath,
			fmt.Sprintf("%.2f", float64(info.FileSize)/(1024*1024)),
			fmt.Sprintf("%.2f", info.Duration/60),
			info.VideoCodec,
			strconv.FormatInt(info.VideoBitrate/1000, 10),
			fmt.Sprintf("%dx%d", info.VideoWidth, info.VideoHeight),
			strconv.Itoa(len(info.AudioTracks)),
			strconv.Itoa(len(info.SubtitleTracks)),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	slog.Debug("CSV report generated", "path", filePath)
	return nil
}

// GenerateJSON creates a JSON report
func (rg *ReportGenerator) GenerateJSON(mediaInfos []*MediaInfo, filename string) error {
	filePath := filepath.Join(rg.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	report := map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"total_files":  len(mediaInfos),
		"media_files":  mediaInfos,
	}

	if err := encoder.Encode(report); err != nil {
		return err
	}

	slog.Debug("JSON report generated", "path", filePath)
	return nil
}

// GenerateMarkdown creates a Markdown report
func (rg *ReportGenerator) GenerateMarkdown(mediaInfos []*MediaInfo, filename string) error {
	filePath := filepath.Join(rg.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Media Analysis Report\n\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Total Files: %d\n\n", len(mediaInfos))

	// Summary statistics
	var totalSize int64
	var totalDuration float64
	codecCount := make(map[string]int)

	for _, info := range mediaInfos {
		totalSize += info.FileSize
		totalDuration += info.Duration
		codecCount[info.VideoCodec]++
	}

	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- **Total Size**: %.2f GB\n", float64(totalSize)/(1024*1024*1024))
	fmt.Fprintf(file, "- **Total Duration**: %.2f hours\n", totalDuration/3600)
	fmt.Fprintf(file, "\n### Video Codecs\n\n")

	for codec, count := range codecCount {
		fmt.Fprintf(file, "- **%s**: %d files\n", codec, count)
	}

	fmt.Fprintf(file, "\n## Detailed Analysis\n\n")
	fmt.Fprintf(file, "| File | Size (MB) | Duration | Codec | Bitrate | Resolution | Audio | Subs |\n")
	fmt.Fprintf(file, "|------|-----------|----------|-------|---------|------------|-------|------|\n")

	// Sort by file path
	sort.Slice(mediaInfos, func(i, j int) bool {
		return mediaInfos[i].FilePath < mediaInfos[j].FilePath
	})

	for _, info := range mediaInfos {
		fileName := filepath.Base(info.FilePath)
		fmt.Fprintf(file, "| %s | %.1f | %.1fm | %s | %dkbps | %dx%d | %d | %d |\n",
			fileName,
			float64(info.FileSize)/(1024*1024),
			info.Duration/60,
			info.VideoCodec,
			info.VideoBitrate/1000,
			info.VideoWidth, info.VideoHeight,
			len(info.AudioTracks),
			len(info.SubtitleTracks))
	}

	slog.Debug("Markdown report generated", "path", filePath)
	return nil
}

// GenerateHTML creates an interactive HTML report
func (rg *ReportGenerator) GenerateHTML(mediaInfos []*MediaInfo, filename string) error {
	filePath := filepath.Join(rg.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	html := rg.generateHTMLContent(mediaInfos)
	if _, err := file.WriteString(html); err != nil {
		return err
	}

	slog.Debug("HTML report generated", "path", filePath)
	return nil
}

func (rg *ReportGenerator) generateHTMLContent(mediaInfos []*MediaInfo) string {
	// Sort by file path for consistent output
	sort.Slice(mediaInfos, func(i, j int) bool {
		return mediaInfos[i].FilePath < mediaInfos[j].FilePath
	})

	// Sanitize media data to ensure nil slices become empty arrays
	sanitizedMediaInfos := make([]*MediaInfo, len(mediaInfos))
	for i, info := range mediaInfos {
		sanitized := *info // Copy the struct

		// Ensure slices are not nil (nil becomes null in JSON, breaking React)
		if sanitized.AudioTracks == nil {
			sanitized.AudioTracks = []AudioTrack{}
		}
		if sanitized.SubtitleTracks == nil {
			sanitized.SubtitleTracks = []SubtitleTrack{}
		}

		// Ensure string fields are not empty for critical data
		if sanitized.VideoCodec == "" {
			sanitized.VideoCodec = "unknown"
		}

		sanitizedMediaInfos[i] = &sanitized
	}

	// Prepare media data
	mediaData := map[string]interface{}{
		"mediaFiles":  sanitizedMediaInfos,
		"totalFiles":  len(mediaInfos),
		"generatedAt": time.Now().Format(time.RFC3339),
		"inputDir":    rg.getInputDir(mediaInfos),
	}

	// Build React bundle with esbuild
	uiBuilder := NewUIBuilder()
	jsBundle, err := uiBuilder.BuildReactBundle(mediaData)
	if err != nil {
		slog.Error("Failed to build React bundle", "error", err)
		return fmt.Sprintf("<html><body><h1>Error: Failed to build UI</h1><p>%s</p></body></html>", err.Error())
	}

	// Read template shell from embedded filesystem
	templateBytes, err := templatesFS.ReadFile("templates/report-shell.html")
	if err != nil {
		slog.Error("Failed to read HTML template", "error", err)
		return fmt.Sprintf("<html><body><h1>Error: Failed to load template</h1><p>%s</p></body></html>", err.Error())
	}

	// Replace the placeholder with compiled JavaScript bundle
	templateContent := string(templateBytes)
	templateContent = strings.Replace(templateContent, "{{.JSBundle}}", jsBundle, 1)

	return templateContent
}

// getInputDir finds the common input directory from all file paths
func (rg *ReportGenerator) getInputDir(mediaInfos []*MediaInfo) string {
	if len(mediaInfos) == 0 {
		return ""
	}

	// Find the longest common prefix of all file paths
	firstPath := mediaInfos[0].FilePath
	commonPrefix := firstPath

	for _, info := range mediaInfos[1:] {
		// Find common prefix between commonPrefix and current path
		i := 0
		minLen := len(commonPrefix)
		if len(info.FilePath) < minLen {
			minLen = len(info.FilePath)
		}

		for i < minLen && commonPrefix[i] == info.FilePath[i] {
			i++
		}

		commonPrefix = commonPrefix[:i]
	}

	// Trim to the last directory separator to get a valid directory path
	lastSlash := strings.LastIndex(commonPrefix, "/")
	if lastSlash >= 0 {
		return commonPrefix[:lastSlash]
	}

	return ""
}
