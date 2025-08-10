package lib

import (
	"math"
	"strconv"
	"strings"
)

// VideoStreamScore represents a video stream with its calculated priority score
type VideoStreamScore struct {
	Stream Stream
	Score  float64
	Index  int
}

// VideoStreamClassification categorizes video streams
type VideoStreamClassification struct {
	Primary   *Stream  // The main video stream
	Auxiliary []Stream // Thumbnail, cover art, etc.
}

// ClassifyVideoStreams analyzes video streams and identifies the primary one
// using heuristics to differentiate real video content from thumbnails/covers
func ClassifyVideoStreams(streams []Stream, formatDuration float64) *VideoStreamClassification {
	videoStreams := extractVideoStreams(streams)
	if len(videoStreams) == 0 {
		return &VideoStreamClassification{}
	}

	if len(videoStreams) == 1 {
		return &VideoStreamClassification{
			Primary: &videoStreams[0],
		}
	}

	scores := make([]VideoStreamScore, len(videoStreams))
	for i, stream := range videoStreams {
		scores[i] = VideoStreamScore{
			Stream: stream,
			Score:  calculateStreamScore(stream, formatDuration),
			Index:  i,
		}
	}

	primaryIndex := 0
	maxScore := scores[0].Score
	for i, score := range scores[1:] {
		if score.Score > maxScore {
			maxScore = score.Score
			primaryIndex = i + 1
		}
	}

	primary := videoStreams[primaryIndex]
	auxiliary := make([]Stream, 0, len(videoStreams)-1)
	for i, stream := range videoStreams {
		if i != primaryIndex {
			auxiliary = append(auxiliary, stream)
		}
	}

	return &VideoStreamClassification{
		Primary:   &primary,
		Auxiliary: auxiliary,
	}
}

// extractVideoStreams filters streams to only video codec types
func extractVideoStreams(streams []Stream) []Stream {
	var videoStreams []Stream
	for _, stream := range streams {
		if stream.CodecType == "video" {
			videoStreams = append(videoStreams, stream)
		}
	}
	return videoStreams
}

// calculateStreamScore computes a priority score for a video stream
// Higher scores indicate more likely to be the primary video content
func calculateStreamScore(stream Stream, formatDuration float64) float64 {
	score := 0.0

	score += getCodecScore(stream.CodecName)
	score += getIndexScore(stream.Index)
	score += getPixelFormatScore(stream.PixelFormat)
	score += getDurationScore(stream, formatDuration)

	pixelCount := stream.Width * stream.Height
	if pixelCount > 0 {
		// Logarithmic scoring to avoid extreme values
		score += math.Log10(float64(pixelCount)) * 10

		// Penalty for very small resolutions (likely thumbnails)
		if pixelCount < 40000 { // 200x200
			score -= 50
		}
	}

	if bitrate := parseBitrate(stream); bitrate > 0 {
		// Logarithmic scoring for bitrate (in kbps)
		score += math.Log10(float64(bitrate)/1000) * 15

		// Penalty for very low bitrates (likely thumbnails)
		if bitrate < 100000 { // 100 kbps
			score -= 30
		}
	}

	return score
}

// getCodecScore assigns priority scores based on codec type
func getCodecScore(codecName string) float64 {
	codec := strings.ToLower(codecName)
	switch codec {
	case "hevc", "h265":
		return 100
	case "h264", "avc":
		return 95
	case "av1":
		return 90
	case "vp9":
		return 85
	case "vp8":
		return 80
	case "mpeg4":
		return 75
	case "mpeg2video":
		return 70
	case "mjpeg":
		return 10 // Very low priority - often thumbnails
	case "png", "bmp":
		return 5 // Likely static images
	default:
		return 50 // Neutral score for unknown codecs
	}
}

// getIndexScore provides slight preference for lower stream indices
func getIndexScore(index int) float64 {
	return math.Max(0, 5-float64(index))
}

// getPixelFormatScore scores based on pixel format characteristics
func getPixelFormatScore(pixelFormat string) float64 {
	pf := strings.ToLower(pixelFormat)
	score := 0.0

	if strings.Contains(pf, "yuv") {
		score += 10 // YUV formats typical for video
	}
	if strings.Contains(pf, "420") {
		score += 5 // 4:2:0 subsampling common for video
	}
	if strings.Contains(pf, "422") {
		score += 5 // 4:2:2 subsampling for higher quality
	}
	if strings.Contains(pf, "444") {
		score += 5 // 4:4:4 no subsampling
	}
	if strings.Contains(pf, "rgb") {
		score -= 5 // RGB more typical for images/thumbnails
	}

	return score
}

// getDurationScore attempts to score based on duration hints
func getDurationScore(stream Stream, formatDuration float64) float64 {
	// If format duration is very short, probably all streams are short
	if formatDuration < 10 {
		return 0
	}

	if stream.Tags != nil {
		if durationStr, exists := stream.Tags["DURATION"]; exists {
			if duration := parseDurationTag(durationStr); duration > 0 {
				// Streams matching format duration get bonus
				durationRatio := duration / formatDuration
				if durationRatio > 0.95 && durationRatio < 1.05 {
					return 20
				}
				// Very short streams compared to format are likely thumbnails
				if durationRatio < 0.1 {
					return -30
				}
			}
		}
	}

	return 0
}

// parseBitrate extracts bitrate from stream, trying multiple fields
func parseBitrate(stream Stream) int64 {
	if stream.Bitrate != "" {
		if bitrate, err := strconv.ParseInt(stream.Bitrate, 10, 64); err == nil {
			return bitrate
		}
	}

	if stream.Tags != nil {
		if bps, exists := stream.Tags["BPS"]; exists {
			if bitrate, err := strconv.ParseInt(bps, 10, 64); err == nil {
				return bitrate
			}
		}
	}

	return 0
}

// parseDurationTag parses duration from string tags (format: HH:MM:SS.mmm)
func parseDurationTag(durationStr string) float64 {
	// Simple parsing for HH:MM:SS.mmm format
	parts := strings.Split(durationStr, ":")
	if len(parts) < 3 {
		return 0
	}

	hours, _ := strconv.ParseFloat(parts[0], 64)
	minutes, _ := strconv.ParseFloat(parts[1], 64)
	seconds, _ := strconv.ParseFloat(parts[2], 64)

	return hours*3600 + minutes*60 + seconds
}
