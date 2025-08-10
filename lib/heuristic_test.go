package lib

import (
	"strings"
	"testing"
)

func TestClassifyVideoStreams_SingleStream(t *testing.T) {
	streams := []Stream{
		{
			Index:     0,
			CodecType: "video",
			CodecName: "h264",
			Width:     1920,
			Height:    1080,
			Bitrate:   "5000000",
		},
	}

	result := ClassifyVideoStreams(streams, 3600.0)

	if result.Primary == nil {
		t.Fatal("Expected primary stream to be set")
	}
	if len(result.Auxiliary) != 0 {
		t.Errorf("Expected 0 auxiliary streams, got %d", len(result.Auxiliary))
	}
	if result.Primary.CodecName != "h264" {
		t.Errorf("Expected primary codec h264, got %s", result.Primary.CodecName)
	}
}

func TestClassifyVideoStreams_NoVideoStreams(t *testing.T) {
	streams := []Stream{
		{
			Index:     0,
			CodecType: "audio",
			CodecName: "aac",
		},
	}

	result := ClassifyVideoStreams(streams, 3600.0)

	if result.Primary != nil {
		t.Error("Expected no primary stream")
	}
	if len(result.Auxiliary) != 0 {
		t.Errorf("Expected 0 auxiliary streams, got %d", len(result.Auxiliary))
	}
}

func TestClassifyVideoStreams_H264VsMJPEG(t *testing.T) {
	streams := []Stream{
		{
			Index:     0,
			CodecType: "video",
			CodecName: "mjpeg",
			Width:     160,
			Height:    120,
			Bitrate:   "50000",
		},
		{
			Index:     1,
			CodecType: "video",
			CodecName: "h264",
			Width:     1920,
			Height:    1080,
			Bitrate:   "5000000",
		},
	}

	result := ClassifyVideoStreams(streams, 3600.0)

	if result.Primary == nil {
		t.Fatal("Expected primary stream to be set")
	}
	if result.Primary.CodecName != "h264" {
		t.Errorf("Expected H.264 to be primary, got %s", result.Primary.CodecName)
	}
	if len(result.Auxiliary) != 1 {
		t.Errorf("Expected 1 auxiliary stream, got %d", len(result.Auxiliary))
	}
	if result.Auxiliary[0].CodecName != "mjpeg" {
		t.Errorf("Expected MJPEG to be auxiliary, got %s", result.Auxiliary[0].CodecName)
	}
}

func TestClassifyVideoStreams_ResolutionPriority(t *testing.T) {
	streams := []Stream{
		{
			Index:     0,
			CodecType: "video",
			CodecName: "h264",
			Width:     640,
			Height:    480,
			Bitrate:   "1000000",
		},
		{
			Index:     1,
			CodecType: "video",
			CodecName: "h264",
			Width:     1920,
			Height:    1080,
			Bitrate:   "5000000",
		},
	}

	result := ClassifyVideoStreams(streams, 3600.0)

	if result.Primary == nil {
		t.Fatal("Expected primary stream to be set")
	}
	if result.Primary.Width != 1920 || result.Primary.Height != 1080 {
		t.Errorf("Expected higher resolution to be primary, got %dx%d", result.Primary.Width, result.Primary.Height)
	}
}

func TestGetCodecScore(t *testing.T) {
	tests := []struct {
		codec    string
		expected float64
	}{
		{"hevc", 100},
		{"h264", 95},
		{"av1", 90},
		{"mjpeg", 10},
		{"png", 5},
		{"unknown", 50},
	}

	for _, test := range tests {
		score := getCodecScore(test.codec)
		if score != test.expected {
			t.Errorf("getCodecScore(%s) = %f, expected %f", test.codec, score, test.expected)
		}
	}
}

func TestGetCodecScore_CaseInsensitive(t *testing.T) {
	tests := []string{"HEVC", "H264", "AV1", "MJPEG"}
	for _, codec := range tests {
		upperScore := getCodecScore(codec)
		lowerScore := getCodecScore(strings.ToLower(codec))
		if upperScore != lowerScore {
			t.Errorf("Codec scoring should be case insensitive: %s vs %s", codec, strings.ToLower(codec))
		}
	}
}

func TestGetIndexScore(t *testing.T) {
	// Lower indices should get higher scores
	score0 := getIndexScore(0)
	score1 := getIndexScore(1)
	score5 := getIndexScore(5)

	if score0 <= score1 {
		t.Errorf("Index 0 should score higher than index 1: %f vs %f", score0, score1)
	}
	if score1 <= score5 {
		t.Errorf("Index 1 should score higher than index 5: %f vs %f", score1, score5)
	}
	if score5 < 0 {
		t.Errorf("Index scores should not be negative: %f", score5)
	}
}

func TestGetPixelFormatScore(t *testing.T) {
	tests := []struct {
		format   string
		expected float64
	}{
		{"yuv420p", 18}, // yuv(10) + 420(8)
		{"yuv422p", 16}, // yuv(10) + 422(6)  
		{"yuv444p", 14}, // yuv(10) + 444(4)
		{"rgb24", -5},
		{"unknown", 0},
	}

	for _, test := range tests {
		score := getPixelFormatScore(test.format)
		if score != test.expected {
			t.Errorf("getPixelFormatScore(%s) = %f, expected %f", test.format, score, test.expected)
		}
	}
}

func TestParseBitrate(t *testing.T) {
	tests := []struct {
		name     string
		stream   Stream
		expected int64
	}{
		{
			name: "bitrate field",
			stream: Stream{
				Bitrate: "5000000",
			},
			expected: 5000000,
		},
		{
			name: "BPS tag",
			stream: Stream{
				Tags: map[string]string{
					"BPS": "3000000",
				},
			},
			expected: 3000000,
		},
		{
			name: "bitrate field priority over BPS",
			stream: Stream{
				Bitrate: "5000000",
				Tags: map[string]string{
					"BPS": "3000000",
				},
			},
			expected: 5000000,
		},
		{
			name:     "no bitrate info",
			stream:   Stream{},
			expected: 0,
		},
		{
			name: "invalid bitrate",
			stream: Stream{
				Bitrate: "invalid",
			},
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseBitrate(test.stream)
			if result != test.expected {
				t.Errorf("parseBitrate() = %d, expected %d", result, test.expected)
			}
		})
	}
}

func TestParseDurationTag(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"01:30:45.500", 5445.5}, // 1*3600 + 30*60 + 45.5
		{"00:05:30.000", 330.0},  // 5*60 + 30
		{"02:00:00.000", 7200.0}, // 2*3600
		{"invalid", 0.0},
		{"10:30", 0.0}, // incomplete format
	}

	for _, test := range tests {
		result := parseDurationTag(test.input)
		if result != test.expected {
			t.Errorf("parseDurationTag(%s) = %f, expected %f", test.input, result, test.expected)
		}
	}
}

func TestGetDurationScore(t *testing.T) {
	tests := []struct {
		name           string
		stream         Stream
		formatDuration float64
		expectedRange  [2]float64 // min, max expected score
	}{
		{
			name: "matching duration gets bonus",
			stream: Stream{
				Tags: map[string]string{
					"DURATION": "01:00:00.000", // 3600 seconds
				},
			},
			formatDuration: 3600.0,
			expectedRange:  [2]float64{15, 25}, // should get ~20 bonus
		},
		{
			name: "very short duration gets penalty",
			stream: Stream{
				Tags: map[string]string{
					"DURATION": "00:00:01.000", // 1 second
				},
			},
			formatDuration: 3600.0,
			expectedRange:  [2]float64{-35, -25}, // should get ~-30 penalty
		},
		{
			name: "short format duration returns neutral",
			stream: Stream{
				Tags: map[string]string{
					"DURATION": "00:00:05.000",
				},
			},
			formatDuration: 5.0,
			expectedRange:  [2]float64{-1, 1}, // should return 0
		},
		{
			name:           "no duration tag returns neutral",
			stream:         Stream{},
			formatDuration: 3600.0,
			expectedRange:  [2]float64{-1, 1}, // should return 0
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score := getDurationScore(test.stream, test.formatDuration)
			if score < test.expectedRange[0] || score > test.expectedRange[1] {
				t.Errorf("getDurationScore() = %f, expected range [%f, %f]", 
					score, test.expectedRange[0], test.expectedRange[1])
			}
		})
	}
}

func TestCalculateStreamScore_Integration(t *testing.T) {
	// Test that a typical main video stream scores higher than a typical thumbnail
	mainStream := Stream{
		Index:       0,
		CodecName:   "h264",
		Width:       1920,
		Height:      1080,
		Bitrate:     "5000000",
		PixelFormat: "yuv420p",
	}

	thumbnailStream := Stream{
		Index:       1,
		CodecName:   "mjpeg",
		Width:       160,
		Height:      120,
		Bitrate:     "50000",
		PixelFormat: "rgb24",
	}

	mainScore := calculateStreamScore(mainStream, 3600.0)
	thumbScore := calculateStreamScore(thumbnailStream, 3600.0)

	if mainScore <= thumbScore {
		t.Errorf("Main video stream should score higher than thumbnail: %f vs %f", mainScore, thumbScore)
	}

	// Ensure scores are reasonable (not extreme values)
	if mainScore < 0 || mainScore > 1000 {
		t.Errorf("Main stream score seems unreasonable: %f", mainScore)
	}
	if thumbScore > 100 {
		t.Errorf("Thumbnail score seems too high: %f", thumbScore)
	}
}

func TestExtractVideoStreams(t *testing.T) {
	streams := []Stream{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "video", CodecName: "mjpeg"},
		{CodecType: "subtitle", CodecName: "srt"},
	}

	videoStreams := extractVideoStreams(streams)

	if len(videoStreams) != 2 {
		t.Errorf("Expected 2 video streams, got %d", len(videoStreams))
	}

	for _, stream := range videoStreams {
		if stream.CodecType != "video" {
			t.Errorf("Expected only video streams, got %s", stream.CodecType)
		}
	}
}

func TestClassifyVideoStreams_ComplexScenario(t *testing.T) {
	// Real-world scenario: movie file with cover art and main video
	streams := []Stream{
		{
			Index:       0,
			CodecType:   "video",
			CodecName:   "mjpeg",
			Width:       600,
			Height:      900, // Portrait cover art
			Bitrate:     "200000",
			PixelFormat: "rgb24",
		},
		{
			Index:       1,
			CodecType:   "audio",
			CodecName:   "ac3",
		},
		{
			Index:       2,
			CodecType:   "video",
			CodecName:   "hevc",
			Width:       3840,
			Height:      2160, // 4K main video
			Bitrate:     "15000000",
			PixelFormat: "yuv420p10le",
		},
	}

	result := ClassifyVideoStreams(streams, 7200.0) // 2 hour movie

	if result.Primary == nil {
		t.Fatal("Expected primary stream to be set")
	}

	// HEVC 4K should be primary over MJPEG cover art
	if result.Primary.CodecName != "hevc" {
		t.Errorf("Expected HEVC to be primary, got %s", result.Primary.CodecName)
	}
	if result.Primary.Width != 3840 {
		t.Errorf("Expected 4K resolution to be primary, got %dx%d", result.Primary.Width, result.Primary.Height)
	}

	if len(result.Auxiliary) != 1 {
		t.Errorf("Expected 1 auxiliary stream, got %d", len(result.Auxiliary))
	}
	if len(result.Auxiliary) > 0 && result.Auxiliary[0].CodecName != "mjpeg" {
		t.Errorf("Expected MJPEG to be auxiliary, got %s", result.Auxiliary[0].CodecName)
	}
}