package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type CacheManager struct {
	CacheDir string
}

type CacheEntry struct {
	FilePath    string     `json:"file_path"`
	FileModTime time.Time  `json:"file_mod_time"`
	FileSize    int64      `json:"file_size"`
	AnalyzedAt  time.Time  `json:"analyzed_at"`
	MediaInfo   *MediaInfo `json:"media_info"`
}

func NewCacheManager(outputDir string) *CacheManager {
	cacheDir := filepath.Join(outputDir, ".cache")
	return &CacheManager{CacheDir: cacheDir}
}

// EnsureCacheDir creates the cache directory if it doesn't exist
func (cm *CacheManager) EnsureCacheDir() error {
	if err := os.MkdirAll(cm.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	return nil
}

// getCacheFileName generates a cache file name from the file path
func (cm *CacheManager) getCacheFileName(filePath string) string {
	hash := sha256.Sum256([]byte(filePath))
	return hex.EncodeToString(hash[:]) + ".json"
}

// getCacheFilePath returns the full path to the cache file
func (cm *CacheManager) getCacheFilePath(filePath string) string {
	return filepath.Join(cm.CacheDir, cm.getCacheFileName(filePath))
}

// HasValidCache checks if a valid cache entry exists for the file
func (cm *CacheManager) HasValidCache(filePath string, fileInfo os.FileInfo) (bool, *MediaInfo, error) {
	cacheFilePath := cm.getCacheFilePath(filePath)

	_, err := os.Stat(cacheFilePath)
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		slog.Warn("Failed to read cache file, will re-analyze", "file", filePath, "error", err)
		return false, nil, nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		slog.Warn("Failed to parse cache file, will re-analyze", "file", filePath, "error", err)
		return false, nil, nil
	}

	if fileInfo.ModTime().After(entry.FileModTime) {
		slog.Debug("Source file modified since cache, will re-analyze", "file", filePath,
			"sourceModTime", fileInfo.ModTime(), "cacheModTime", entry.FileModTime)
		return false, nil, nil
	}

	if fileInfo.Size() != entry.FileSize {
		slog.Debug("Source file size changed since cache, will re-analyze", "file", filePath,
			"sourceSize", fileInfo.Size(), "cacheSize", entry.FileSize)
		return false, nil, nil
	}

	if time.Since(entry.AnalyzedAt) > 30*24*time.Hour {
		slog.Debug("Cache entry too old, will re-analyze", "file", filePath, "age", time.Since(entry.AnalyzedAt))
		return false, nil, nil
	}

	slog.Debug("Using cached analysis", "file", filePath, "cachedAt", entry.AnalyzedAt)
	return true, entry.MediaInfo, nil
}

// SaveCache stores the analysis result in a cache file
func (cm *CacheManager) SaveCache(filePath string, fileInfo os.FileInfo, mediaInfo *MediaInfo) error {
	entry := CacheEntry{
		FilePath:    filePath,
		FileModTime: fileInfo.ModTime(),
		FileSize:    fileInfo.Size(),
		AnalyzedAt:  time.Now(),
		MediaInfo:   mediaInfo,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheFilePath := cm.getCacheFilePath(filePath)
	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	slog.Debug("Saved analysis to cache", "file", filePath, "cacheFile", cacheFilePath)
	return nil
}

// CleanOldCache removes cache files older than the specified duration
func (cm *CacheManager) CleanOldCache(maxAge time.Duration) error {
	entries, err := os.ReadDir(cm.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(cm.CacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				slog.Warn("Failed to remove old cache file", "file", filePath, "error", err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		slog.Info("Cleaned old cache files", "count", cleaned, "maxAge", maxAge)
	}

	return nil
}
