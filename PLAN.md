# Space-Saving Optimization Feature Plan

## Overview

Add intelligent pre-encoding size estimation to skip files that won't save
sufficient disk space, avoiding lengthy encodes that don't meet user-defined
savings thresholds.

## User Interface Changes

### New CLI Flag

Add to `cmd/transcode.go`:

```go
transcodeMinSavings int // --min-savings-percent, default 20
```

Usage:

```bash
./media-mgmt transcode --files movie.mkv --min-savings-percent 20
```

### New Struct Field

Add to `HandBrakeTranscoder` in `lib/handbrake.go`:

```go
type HandBrakeTranscoder struct {
    // ... existing fields
    MinSavingsPercent int
}
```

## Implementation Details

### 1. Marker File System

**Location**: `lib/handbrake.go`

**Skip File Format**:

```
/path/to/movie.mkv → /path/to/movie.skip
```

**Skip File Contents** (JSON):

```json
{
  "reason": "insufficient_savings",
  "quality": 70,
  "encoder": "vt_h265",
  "timestamp": "2025-08-06T15:30:00Z",
  "original_size_bytes": 5368709120,
  "estimated_size_bytes": 4914980659,
  "required_size_bytes": 4294967296
}
```

### 2. Size Estimation Logic

**Location**: New method in `lib/handbrake.go`

```go
func (t *HandBrakeTranscoder) estimateOutputSize(ctx context.Context, inputPath string, videoInfo *VideoInfo, hasVideoToolbox bool) (int64, error)
```

**Approach**:

- Encode 3 segments of 10 seconds each at 25%, 50%, and 75% progress through video
- Calculate average bytes per second across all segments
- Extrapolate: `estimatedSize = avgBytesPerSecond * fullDuration`
- Use temporary files: `inputPath + ".size-test-1.mkv"`, `.size-test-2.mkv`, `.size-test-3.mkv`
- Clean up test files after estimation

**Test Encode Segment Logic**:

```go
segmentDuration := 10.0 // seconds
positions := []float64{0.25, 0.50, 0.75} // 25%, 50%, 75% through video
for i, pos := range positions {
    startTime := videoInfo.Duration * pos
    // Encode 10s segment starting at startTime
}
```

### 3. Pre-Encode Validation

**Location**: Modify `transcodeFile()` in `lib/handbrake.go:170`

**Flow**:

1. Check for existing `.skip` file
2. If found, log skip reason and return early
3. If not found, run size estimation
4. Calculate savings: `(originalSize - estimatedSize) / originalSize * 100`
5. If savings < threshold:
   - Create `.skip` file
   - Log skip with details
   - Return early
6. Otherwise, proceed with full encode

### 4. Skip File Management

**Check Skip File**:

```go
func (t *HandBrakeTranscoder) checkSkipFile(filePath string) (*SkipInfo, error) {
    skipPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".skip"
    // Read and parse skip file - if exists, return skip info
}
```

**Create Skip File**:

```go
func (t *HandBrakeTranscoder) createSkipFile(filePath string, reason string, originalSize, estimatedSize int64, encoder string) error {
    requiredSize := int64(float64(originalSize) * (100 - float64(t.MinSavingsPercent)) / 100)
    skipInfo := SkipInfo{
        Reason: reason,
        Quality: t.Quality,
        Encoder: encoder,
        Timestamp: time.Now(),
        OriginalSizeBytes: originalSize,
        EstimatedSizeBytes: estimatedSize,
        RequiredSizeBytes: requiredSize,
    }
    // Write JSON to .skip file
}
```

### 5. Enhanced Logging

**Skip Logging**:

```go
slog.Info("Skipping file - insufficient space savings",
    "file", filepath.Base(filePath),
    "estimated_savings", fmt.Sprintf("%.1f%%", estimatedSavings),
    "required_savings", fmt.Sprintf("%d%%", t.MinSavingsPercent),
    "original_size", formatSize(originalSize),
    "estimated_size", formatSize(estimatedSize))
```

**Previously Skipped**:

```go
slog.Info("Skipping file - previously marked as insufficient savings",
    "file", filepath.Base(filePath),
    "skip_date", skipInfo.Timestamp.Format(time.RFC3339))
```

## File Structure Changes

### Modified Files

- `cmd/transcode.go` - Add CLI flag and pass to HandBrakeTranscoder
- `lib/handbrake.go` - Main implementation logic

### New Types

```go
type SkipInfo struct {
    Reason             string    `json:"reason"`
    Quality            int       `json:"quality"`
    Encoder            string    `json:"encoder"`
    Timestamp          time.Time `json:"timestamp"`
    OriginalSizeBytes  int64     `json:"original_size_bytes"`
    EstimatedSizeBytes int64     `json:"estimated_size_bytes"`
    RequiredSizeBytes  int64     `json:"required_size_bytes"`
}
```

## Error Handling

- Size estimation failures should not block processing (log warning, proceed
  with full encode)
- Skip file read errors should not block processing (re-evaluate file)
- Skip file write errors should be logged but not fail the operation

## Testing Strategy

1. Test with files that should be skipped (high-quality H.265 sources)
2. Test with files that should be processed (large H.264 sources)
3. Verify skip files are created correctly
4. Verify skip files prevent re-processing
5. Test settings hash invalidation when quality changes

## Default Settings

- **Minimum savings threshold**: 20% (files must save at least 20% space to be processed)
- **Test encode segments**: 3 segments of 10 seconds each (30 seconds total)
- **Segment positions**: 25%, 50%, 75% through video duration
- **Skip when disabled**: When `--min-savings-percent 0`, feature is disabled and all files are processed

## Future Enhancements

- `--force-recheck` flag to ignore existing skip files
- Skip file cleanup command
- Statistics on total time/space saved by skipping
