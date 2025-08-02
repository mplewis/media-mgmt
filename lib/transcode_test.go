package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPresets(t *testing.T) {
	// Create temporary presets file
	yamlContent := `presets:
  - name: "hdr"
    condition: "hdr"
    container: "mkv"
    video:
      codec: "h265"
      bit_depth: 10
      quality: "cq:80"
      hw_accel: true
    audio: "passthrough"
    subtitles: "passthrough"
    
  - name: "sdr"
    condition: "always"
    container: "mkv"
    video:
      codec: "h265"
      bit_depth: 8
      quality: "cq:80"
      hw_accel: true
    audio: "passthrough"
    subtitles: "passthrough"`

	tmpFile, err := os.CreateTemp("", "presets-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	transcoder := &Transcoder{
		PresetsFile: tmpFile.Name(),
	}

	err = transcoder.loadPresets()
	if err != nil {
		t.Fatalf("Failed to load presets: %v", err)
	}

	if len(transcoder.presets) != 2 {
		t.Errorf("Expected 2 presets, got %d", len(transcoder.presets))
	}

	if transcoder.presets[0].Name != "hdr" {
		t.Errorf("Expected first preset name to be 'hdr', got '%s'", transcoder.presets[0].Name)
	}

	if transcoder.presets[0].Video.BitDepth != 10 {
		t.Errorf("Expected HDR preset bit depth to be 10, got %d", transcoder.presets[0].Video.BitDepth)
	}

	if transcoder.presets[1].Name != "sdr" {
		t.Errorf("Expected second preset name to be 'sdr', got '%s'", transcoder.presets[1].Name)
	}
}

func TestFindMatchingPreset(t *testing.T) {
	presets := []Preset{
		{
			Name:      "hdr",
			Condition: "hdr",
		},
		{
			Name:      "sdr",
			Condition: "always",
		},
	}

	transcoder := &Transcoder{
		presets: presets,
	}

	// Test HDR video
	hdrVideo := &VideoInfo{IsHDR: true}
	preset := transcoder.findMatchingPreset(hdrVideo)
	if preset == nil || preset.Name != "hdr" {
		t.Errorf("Expected HDR preset for HDR video, got %v", preset)
	}

	// Test SDR video
	sdrVideo := &VideoInfo{IsHDR: false}
	preset = transcoder.findMatchingPreset(sdrVideo)
	if preset == nil || preset.Name != "sdr" {
		t.Errorf("Expected SDR preset for SDR video, got %v", preset)
	}
}

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		presetName   string
		outputSuffix string
		expected     string
	}{
		{
			name:         "default suffix",
			inputPath:    "/path/to/video.mp4",
			presetName:   "hdr",
			outputSuffix: "-optimized-{preset}",
			expected:     "/path/to/video-optimized-hdr.mkv",
		},
		{
			name:         "custom suffix",
			inputPath:    "/path/to/video.mkv",
			presetName:   "sdr",
			outputSuffix: "-{preset}-transcoded",
			expected:     "/path/to/video-sdr-transcoded.mkv",
		},
		{
			name:         "empty suffix",
			inputPath:    "/path/to/video.avi",
			presetName:   "hdr",
			outputSuffix: "",
			expected:     "/path/to/video-optimized-hdr.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcoder := &Transcoder{
				OutputSuffix: tt.outputSuffix,
			}

			result := transcoder.generateOutputPath(tt.inputPath, tt.presetName)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDetectHDR(t *testing.T) {
	transcoder := &Transcoder{}

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
			name:     "arib-std-b67 HDR",
			output:   "arib-std-b67",
			expected: true,
		},
		{
			name:     "no HDR indicators",
			output:   "color_primaries=bt709 color_transfer=bt709",
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
	transcoder := &Transcoder{
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
	tmpDir, err := os.MkdirTemp("", "transcode-test-")
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

	transcoder = &Transcoder{
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