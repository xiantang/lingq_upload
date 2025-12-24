package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAudioProcessor(t *testing.T) {
	t.Parallel()

	processor := NewAudioProcessor()

	if processor.M4bToolPath != "m4b-tool" {
		t.Errorf("M4bToolPath = %q, want %q", processor.M4bToolPath, "m4b-tool")
	}
	if processor.OutputFormat != "mp3" {
		t.Errorf("OutputFormat = %q, want %q", processor.OutputFormat, "mp3")
	}
	if processor.AudioBitrate != "96k" {
		t.Errorf("AudioBitrate = %q, want %q", processor.AudioBitrate, "96k")
	}
	if processor.AudioChannels != 1 {
		t.Errorf("AudioChannels = %d, want %d", processor.AudioChannels, 1)
	}
	if processor.AudioSamplerate != 22050 {
		t.Errorf("AudioSamplerate = %d, want %d", processor.AudioSamplerate, 22050)
	}
}

func TestFindFilesByExt(t *testing.T) {
	t.Parallel()

	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"book.mp3",
		"Book.MP3", // Test case insensitivity
		"chapter.cue",
		"CHAPTER.CUE", // Test case insensitivity
		"readme.txt",
		"cover.jpg",
	}

	for _, name := range testFiles {
		filePath := filepath.Join(tempDir, name)
		if err := os.WriteFile(filePath, []byte("test content"), 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	tests := []struct {
		name     string
		ext      string
		wantLen  int
		wantName string // Check if specific filename is included
	}{
		{
			name:     "find mp3 files",
			ext:      ".mp3",
			wantLen:  2,
			wantName: "book.mp3",
		},
		{
			name:     "find cue files",
			ext:      ".cue",
			wantLen:  2,
			wantName: "chapter.cue",
		},
		{
			name:    "find txt files",
			ext:     ".txt",
			wantLen: 1,
		},
		{
			name:    "find non-existent extension",
			ext:     ".pdf",
			wantLen: 0,
		},
		{
			name:     "case insensitive matching",
			ext:      ".MP3",
			wantLen:  2,
			wantName: "Book.MP3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := findFilesByExt(tempDir, tt.ext)
			if err != nil {
				t.Fatalf("findFilesByExt() error = %v", err)
			}

			if len(files) != tt.wantLen {
				t.Errorf("findFilesByExt() returned %d files, want %d", len(files), tt.wantLen)
			}

			// Check if specific file is included (if specified)
			if tt.wantName != "" {
				found := false
				for _, file := range files {
					if filepath.Base(file) == tt.wantName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected file %q not found in results", tt.wantName)
				}
			}
		})
	}
}

func TestFindFilesByExt_InvalidDirectory(t *testing.T) {
	t.Parallel()

	_, err := findFilesByExt("/nonexistent/directory", ".mp3")
	if err == nil {
		t.Error("findFilesByExt() expected error for invalid directory, got nil")
	}
}

func TestNeedsSplitting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mp3Files []string
		cueFiles []string
		want     bool
	}{
		{
			name:     "single mp3 with cue - needs splitting",
			mp3Files: []string{"book.mp3"},
			cueFiles: []string{"book.cue"},
			want:     true,
		},
		{
			name:     "single mp3 with multiple cues - needs splitting",
			mp3Files: []string{"book.mp3"},
			cueFiles: []string{"book.cue", "extra.cue"},
			want:     true,
		},
		{
			name:     "multiple mp3s with cue - no splitting needed",
			mp3Files: []string{"chapter1.mp3", "chapter2.mp3"},
			cueFiles: []string{"book.cue"},
			want:     false,
		},
		{
			name:     "single mp3 without cue - no splitting needed",
			mp3Files: []string{"book.mp3"},
			cueFiles: []string{},
			want:     false,
		},
		{
			name:     "no mp3s - no splitting needed",
			mp3Files: []string{},
			cueFiles: []string{"book.cue"},
			want:     false,
		},
		{
			name:     "no files at all - no splitting needed",
			mp3Files: []string{},
			cueFiles: []string{},
			want:     false,
		},
		{
			name:     "multiple mp3s without cue - no splitting needed",
			mp3Files: []string{"chapter1.mp3", "chapter2.mp3", "chapter3.mp3"},
			cueFiles: []string{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsSplitting(tt.mp3Files, tt.cueFiles)
			if got != tt.want {
				t.Errorf("needsSplitting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcess_NoCueFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create MP3 files but no CUE file
	mp3Path := filepath.Join(tempDir, "chapter1.mp3")
	if err := os.WriteFile(mp3Path, []byte("fake mp3"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	processor := NewAudioProcessor()
	ctx := context.Background()

	result, err := processor.Process(ctx, tempDir)
	if err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	if result.Processed {
		t.Error("Process() should not process when no CUE file present")
	}
	if result.SplitFilesDir != "" {
		t.Errorf("Process() SplitFilesDir = %q, want empty", result.SplitFilesDir)
	}
}

func TestProcess_MultipleMp3Files(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create multiple MP3 files with different names and a CUE file
	for i := 1; i <= 3; i++ {
		mp3Path := filepath.Join(tempDir, fmt.Sprintf("chapter_%02d.mp3", i))
		if err := os.WriteFile(mp3Path, []byte("fake mp3"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	cuePath := filepath.Join(tempDir, "book.cue")
	if err := os.WriteFile(cuePath, []byte("fake cue"), 0o644); err != nil {
		t.Fatalf("failed to create cue file: %v", err)
	}

	processor := NewAudioProcessor()
	ctx := context.Background()

	result, err := processor.Process(ctx, tempDir)
	if err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	if result.Processed {
		t.Error("Process() should not process when multiple MP3 files present")
	}
}

func TestProcess_DirectoryReadError(t *testing.T) {
	t.Parallel()

	processor := NewAudioProcessor()
	ctx := context.Background()

	// Try to process a non-existent directory
	_, err := processor.Process(ctx, "/nonexistent/directory")
	if err == nil {
		t.Error("Process() expected error for non-existent directory, got nil")
	}
}

func TestSplitAudio_M4bToolNotFound(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	mp3Path := filepath.Join(tempDir, "test.mp3")
	if err := os.WriteFile(mp3Path, []byte("fake mp3"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Use a processor with non-existent m4b-tool path
	processor := &AudioProcessor{
		M4bToolPath:     "nonexistent-m4b-tool-binary",
		OutputFormat:    "mp3",
		AudioBitrate:    "96k",
		AudioChannels:   1,
		AudioSamplerate: 22050,
	}

	ctx := context.Background()
	err := processor.splitAudio(ctx, mp3Path)

	if err == nil {
		t.Fatal("splitAudio() expected error when m4b-tool not found, got nil")
	}

	// Check error message contains helpful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "m4b-tool not found") {
		t.Errorf("error message should mention m4b-tool not found, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "brew install") || !strings.Contains(errMsg, "https://github.com") {
		t.Errorf("error message should contain installation instructions, got: %v", errMsg)
	}
}

func TestProcessResult_Fields(t *testing.T) {
	t.Parallel()

	result := &ProcessResult{
		Processed:     true,
		SplitFilesDir: "/path/to/split",
		OriginalFile:  "/path/to/book.mp3",
		CueFile:       "/path/to/book.cue",
	}

	if !result.Processed {
		t.Error("Processed field should be true")
	}
	if result.SplitFilesDir != "/path/to/split" {
		t.Errorf("SplitFilesDir = %q, want %q", result.SplitFilesDir, "/path/to/split")
	}
	if result.OriginalFile != "/path/to/book.mp3" {
		t.Errorf("OriginalFile = %q, want %q", result.OriginalFile, "/path/to/book.mp3")
	}
	if result.CueFile != "/path/to/book.cue" {
		t.Errorf("CueFile = %q, want %q", result.CueFile, "/path/to/book.cue")
	}
}

// TestProcess_WithActualM4bTool is an integration test that requires m4b-tool to be installed.
// It's skipped if m4b-tool is not available in PATH.
func TestProcess_WithActualM4bTool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if m4b-tool is available
	processor := NewAudioProcessor()
	if _, err := os.Stat(processor.M4bToolPath); err != nil {
		t.Skipf("m4b-tool not found in PATH, skipping integration test")
	}

	// Note: Actual m4b-tool testing would require a real MP3 + CUE file
	// This is left as a manual integration test since we can't ship test audio files
	t.Log("m4b-tool is available - manual testing recommended with real audio files")
}
