package lib

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var videoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
	".m4v":  true,
	".3gp":  true,
	".ogv":  true,
	".ts":   true,
	".m2ts": true,
	".mts":  true,
}

type FileScanner struct {
	rootDir string
}

func NewFileScanner(rootDir string) *FileScanner {
	return &FileScanner{rootDir: rootDir}
}

// ScanVideoFiles recursively finds all video files in the root directory
func (fs *FileScanner) ScanVideoFiles(ctx context.Context) ([]string, error) {
	slog.Debug("Starting video file scan", "rootDir", fs.rootDir)

	var videoFiles []string

	err := filepath.Walk(fs.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Error accessing path", "path", path, "error", err)
			return nil // Continue walking despite individual file errors
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			videoFiles = append(videoFiles, path)
			slog.Debug("Found video file", "path", path, "size", info.Size())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	slog.Info("Video file scan completed", "filesFound", len(videoFiles))
	return videoFiles, nil
}
