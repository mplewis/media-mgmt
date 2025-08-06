package lib

import (
	"context"
	"fmt"
	"log/slog"
)

func FormatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
}

func PrintMediaInfo(filePath string) error {
	return printMediaInfoWithRatio(filePath, 0)
}

func PrintMediaInfoWithRatio(filePath string, originalFileSize int64) error {
	return printMediaInfoWithRatio(filePath, originalFileSize)
}

func printMediaInfoWithRatio(filePath string, originalFileSize int64) error {
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