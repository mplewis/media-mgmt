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
	FilePath       string        `json:"file_path"`
	FileSize       int64         `json:"file_size"`
	Duration       float64       `json:"duration"`
	VideoCodec     string        `json:"video_codec"`
	VideoBitrate   int64         `json:"video_bitrate"`
	VideoWidth     int           `json:"video_width"`
	VideoHeight    int           `json:"video_height"`
	VideoProfile   string        `json:"video_profile"`
	VideoLevel     string        `json:"video_level"`
	PixelFormat    string        `json:"pixel_format"`
	IsVBR          bool          `json:"is_vbr"`
	ColorSpace     string        `json:"color_space"`
	ColorTransfer  string        `json:"color_transfer"`
	HasDolbyVision bool          `json:"has_dolby_vision"`
	AudioTracks    []AudioTrack  `json:"audio_tracks"`
	SubtitleTracks []SubtitleTrack `json:"subtitle_tracks"`
	AnalyzedAt     time.Time     `json:"analyzed_at"`
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
	Profile       string            `json:"profile,omitempty"`
	Level         int               `json:"level,omitempty"`
	PixelFormat   string            `json:"pix_fmt,omitempty"`
	ColorSpace    string            `json:"color_space,omitempty"`
	ColorTransfer string            `json:"color_transfer,omitempty"`
	Bitrate       string            `json:"bit_rate,omitempty"`
	Width         int               `json:"width,omitempty"`
	Height        int               `json:"height,omitempty"`
	Channels      int               `json:"channels,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	SideDataList  []SideData        `json:"side_data_list,omitempty"`
}

type SideData struct {
	SideDataType string `json:"side_data_type"`
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
	
	// Try to get overall bitrate from format first
	var overallBitrate int64
	if probe.Format.Bitrate != "" {
		if bitrate, err := strconv.ParseInt(probe.Format.Bitrate, 10, 64); err == nil {
			overallBitrate = bitrate
		}
	}
	
	// Parse streams
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			info.VideoCodec = stream.CodecName
			info.VideoWidth = stream.Width
			info.VideoHeight = stream.Height
			info.VideoProfile = stream.Profile
			info.PixelFormat = stream.PixelFormat
			info.ColorSpace = stream.ColorSpace
			info.ColorTransfer = stream.ColorTransfer
			
			// Convert level to readable format
			if stream.Level > 0 {
				info.VideoLevel = formatLevel(stream.Level)
			}
			
			// Check for Dolby Vision
			for _, sideData := range stream.SideDataList {
				if sideData.SideDataType == "DOVI configuration record" {
					info.HasDolbyVision = true
					break
				}
			}
			
			// Try to get bitrate from stream first
			if stream.Bitrate != "" {
				if bitrate, err := strconv.ParseInt(stream.Bitrate, 10, 64); err == nil {
					info.VideoBitrate = bitrate
				}
			} else {
				// If no direct bitrate, check for BPS in tags (indicates VBR)
				if bps, exists := stream.Tags["BPS"]; exists {
					if bitrate, err := strconv.ParseInt(bps, 10, 64); err == nil {
						info.VideoBitrate = bitrate
						info.IsVBR = true
					}
				}
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
	
	// If video bitrate is still 0, try fallback methods
	if info.VideoBitrate == 0 {
		// Method 1: Use overall bitrate and subtract estimated audio bitrate
		if overallBitrate > 0 {
			estimatedAudioBitrate := int64(len(info.AudioTracks)) * 256000 // 256kbps per track estimate
			if overallBitrate > estimatedAudioBitrate {
				info.VideoBitrate = overallBitrate - estimatedAudioBitrate
			} else {
				// If overall is less than audio estimate, use 85% of overall
				info.VideoBitrate = int64(float64(overallBitrate) * 0.85)
			}
		}
		
		// Method 2: Calculate from file size and duration (last resort)
		if info.VideoBitrate == 0 && info.Duration > 0 && info.FileSize > 0 {
			// Calculate total bitrate from file size and duration
			totalBits := info.FileSize * 8 // Convert bytes to bits
			totalBitrate := int64(float64(totalBits) / info.Duration)
			// Estimate video is 85% of total bitrate
			info.VideoBitrate = int64(float64(totalBitrate) * 0.85)
		}
	}
	
	return nil
}

// formatLevel converts numeric level to readable format
func formatLevel(level int) string {
	// HEVC levels: 30=1, 60=2, 63=2.1, 90=3, 93=3.1, 120=4, 123=4.1, 150=5, 153=5.1, 156=5.2, 180=6, 183=6.1, 186=6.2
	switch level {
	case 30:
		return "1"
	case 60:
		return "2"
	case 63:
		return "2.1"
	case 90:
		return "3"
	case 93:
		return "3.1"
	case 120:
		return "4"
	case 123:
		return "4.1"
	case 150:
		return "5"
	case 153:
		return "5.1"
	case 156:
		return "5.2"
	case 180:
		return "6"
	case 183:
		return "6.1"
	case 186:
		return "6.2"
	default:
		return fmt.Sprintf("%.1f", float64(level)/30.0)
	}
}

// CheckFFprobeAvailable verifies that ffprobe is available in PATH
func CheckFFprobeAvailable() error {
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return fmt.Errorf("ffprobe not found in PATH - please install FFmpeg")
	}
	return nil
}