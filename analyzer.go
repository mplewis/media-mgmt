package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type MediaInfo struct {
	FilePath      string        `json:"file_path"`
	FileSize      int64         `json:"file_size"`
	Duration      float64       `json:"duration"`
	VideoCodec    string        `json:"video_codec"`
	VideoBitrate  int64         `json:"video_bitrate"`
	VideoWidth    int           `json:"video_width"`
	VideoHeight   int           `json:"video_height"`
	AudioTracks   []AudioTrack  `json:"audio_tracks"`
	SubtitleTracks []SubtitleTrack `json:"subtitle_tracks"`
	AnalyzedAt    time.Time     `json:"analyzed_at"`
}

type AudioTrack struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Bitrate  int64  `json:"bitrate"`
	Language string `json:"language"`
	Channels int    `json:"channels"`
}

type SubtitleTrack struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Language string `json:"language"`
}

type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

type Stream struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecType     string            `json:"codec_type"`
	Bitrate       string            `json:"bit_rate,omitempty"`
	Width         int               `json:"width,omitempty"`
	Height        int               `json:"height,omitempty"`
	Channels      int               `json:"channels,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type Format struct {
	Filename   string            `json:"filename"`
	Size       string            `json:"size"`
	Duration   string            `json:"duration"`
	Bitrate    string            `json:"bit_rate"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type MediaAnalyzer struct{}

func NewMediaAnalyzer() *MediaAnalyzer {
	return &MediaAnalyzer{}
}

// AnalyzeFile analyzes a single video file using FFprobe
func (ma *MediaAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*MediaInfo, error) {
	slog.Debug("Analyzing file", "path", filePath)
	
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	
	// Run FFprobe
	probeData, err := ma.runFFprobe(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %w", filePath, err)
	}
	
	// Parse the results
	mediaInfo := &MediaInfo{
		FilePath:   filePath,
		FileSize:   fileInfo.Size(),
		AnalyzedAt: time.Now(),
	}
	
	if err := ma.parseFFprobeOutput(probeData, mediaInfo); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output for %s: %w", filePath, err)
	}
	
	slog.Debug("File analysis completed", 
		"path", filePath,
		"codec", mediaInfo.VideoCodec,
		"duration", mediaInfo.Duration,
		"audioTracks", len(mediaInfo.AudioTracks),
		"subtitleTracks", len(mediaInfo.SubtitleTracks))
	
	return mediaInfo, nil
}

func (ma *MediaAnalyzer) runFFprobe(ctx context.Context, filePath string) (*FFProbeOutput, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", 
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)
	
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe exit code %d: %s", exitError.ExitCode(), string(exitError.Stderr))
		}
		return nil, err
	}
	
	var probeOutput FFProbeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe JSON output: %w", err)
	}
	
	return &probeOutput, nil
}

func (ma *MediaAnalyzer) parseFFprobeOutput(probe *FFProbeOutput, info *MediaInfo) error {
	// Parse format information
	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.Duration = duration
	}
	
	// Parse streams
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			info.VideoCodec = stream.CodecName
			info.VideoWidth = stream.Width
			info.VideoHeight = stream.Height
			if bitrate, err := strconv.ParseInt(stream.Bitrate, 10, 64); err == nil {
				info.VideoBitrate = bitrate
			}
			
		case "audio":
			track := AudioTrack{
				Index:   stream.Index,
				Codec:   stream.CodecName,
				Channels: stream.Channels,
			}
			
			if bitrate, err := strconv.ParseInt(stream.Bitrate, 10, 64); err == nil {
				track.Bitrate = bitrate
			}
			
			if lang, exists := stream.Tags["language"]; exists {
				track.Language = lang
			}
			
			info.AudioTracks = append(info.AudioTracks, track)
			
		case "subtitle":
			track := SubtitleTrack{
				Index: stream.Index,
				Codec: stream.CodecName,
			}
			
			if lang, exists := stream.Tags["language"]; exists {
				track.Language = lang
			}
			
			info.SubtitleTracks = append(info.SubtitleTracks, track)
		}
	}
	
	return nil
}

// CheckFFprobeAvailable verifies that ffprobe is available in PATH
func CheckFFprobeAvailable() error {
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return fmt.Errorf("ffprobe not found in PATH - please install FFmpeg")
	}
	return nil
}