package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		outputSuffix string
		expected     string
	}{
		{
			name:         "default suffix",
			inputPath:    "/path/to/video.mp4",
			outputSuffix: "-optimized",
			expected:     "/path/to/video-optimized.mkv",
		},
		{
			name:         "custom suffix",
			inputPath:    "/path/to/video.mkv",
			outputSuffix: "-transcoded",
			expected:     "/path/to/video-transcoded.mkv",
		},
		{
			name:         "no suffix",
			inputPath:    "/path/to/video.avi",
			outputSuffix: "",
			expected:     "/path/to/video.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcoder := &HandBrakeTranscoder{
				OutputSuffix: tt.outputSuffix,
			}

			result := transcoder.generateOutputPath(tt.inputPath)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDetectHDR(t *testing.T) {
	transcoder := &HandBrakeTranscoder{}

	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "bt2020 HDR",
			output:   "color_primaries=bt2020",
			expected: true,
		},
		{
			name:     "smpte2084 HDR",
			output:   "color_transfer=smpte2084",
			expected: true,
		},
		{
			name:     "10-bit yuv420p10le",
			output:   "yuv420p10le",
			expected: true,
		},
		{
			name:     "no HDR indicators",
			output:   "color_primaries=bt709 color_transfer=bt709 yuv420p",
			expected: false,
		},
		{
			name:     "case insensitive",
			output:   "COLOR_PRIMARIES=BT2020",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transcoder.detectHDR(tt.output)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetFileList(t *testing.T) {
	// Test with direct files
	transcoder := &HandBrakeTranscoder{
		Files: []string{"file1.mkv", "file2.mp4"},
	}

	files, err := transcoder.getFileList()
	if err != nil {
		t.Fatalf("Failed to get file list: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Test with file list
	tmpDir, err := os.MkdirTemp("", "handbrake-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	listFile := filepath.Join(tmpDir, "files.txt")
	listContent := `file3.mkv
# This is a comment
file4.mp4

file5.avi`

	if err := os.WriteFile(listFile, []byte(listContent), 0644); err != nil {
		t.Fatalf("Failed to write file list: %v", err)
	}

	transcoder = &HandBrakeTranscoder{
		Files:        []string{"file1.mkv"},
		FileListPath: listFile,
	}

	files, err = transcoder.getFileList()
	if err != nil {
		t.Fatalf("Failed to get file list: %v", err)
	}

	expectedFiles := []string{"file1.mkv", "file3.mkv", "file4.mp4", "file5.avi"}
	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(files))
	}

	for i, expected := range expectedFiles {
		if i >= len(files) || files[i] != expected {
			t.Errorf("Expected file %d to be '%s', got '%s'", i, expected, files[i])
		}
	}
}

func TestCheckHandBrakeCLI(t *testing.T) {
	transcoder := &HandBrakeTranscoder{}
	
	// This test will pass if HandBrakeCLI is installed, otherwise it will fail
	// In a real CI environment, you might want to mock this or skip the test
	err := transcoder.checkHandBrakeCLI()
	if err != nil {
		t.Logf("HandBrakeCLI not available: %v", err)
		t.Skip("HandBrakeCLI not installed, skipping test")
	}
}

func TestDetectVideoToolbox(t *testing.T) {
	transcoder := &HandBrakeTranscoder{}
	
	// This test checks VideoToolbox detection
	// Results will vary based on platform
	hasVT, err := transcoder.detectVideoToolbox()
	if err != nil {
		t.Logf("VideoToolbox detection error: %v", err)
	}
	
	t.Logf("VideoToolbox available: %v", hasVT)
	
	// On macOS, VideoToolbox should be available if HandBrake is installed
	// On other platforms, it should be false
}