package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AudioProcessor handles post-download audio processing, specifically
// detecting and splitting unsplit audiobooks that come with CUE sheets.
type AudioProcessor struct {
	M4bToolPath     string // Path to m4b-tool binary (default: "m4b-tool")
	OutputFormat    string // Audio output format (default: "mp3")
	AudioBitrate    string // Audio bitrate (default: "96k")
	AudioChannels   int    // Number of audio channels (default: 1)
	AudioSamplerate int    // Audio sample rate (default: 22050)
}

// NewAudioProcessor creates an AudioProcessor with default settings
// optimized for audiobook chapter splitting.
func NewAudioProcessor() *AudioProcessor {
	return &AudioProcessor{
		M4bToolPath:     "m4b-tool",
		OutputFormat:    "mp3",
		AudioBitrate:    "96k",
		AudioChannels:   1,
		AudioSamplerate: 22050,
	}
}

// ProcessResult captures the result of audio processing.
type ProcessResult struct {
	Processed     bool   // Whether any processing was performed
	SplitFilesDir string // Directory containing split files (if split occurred)
	OriginalFile  string // Path to the original MP3 file
	CueFile       string // Path to the CUE file (if exists)
}

// Process checks if audio splitting is needed and performs it if necessary.
// It detects the presence of a CUE file with a single MP3 file, which indicates
// an unsplit audiobook that needs to be split into chapters.
//
// Returns:
//   - ProcessResult with details about what was processed
//   - error if processing failed (strict mode - processing errors are fatal)
func (p *AudioProcessor) Process(ctx context.Context, outputDir string) (*ProcessResult, error) {
	result := &ProcessResult{}

	// Step 1: Find CUE and MP3 files in the output directory
	cueFiles, err := findFilesByExt(outputDir, ".cue")
	if err != nil {
		return nil, fmt.Errorf("find CUE files: %w", err)
	}

	mp3Files, err := findFilesByExt(outputDir, ".mp3")
	if err != nil {
		return nil, fmt.Errorf("find MP3 files: %w", err)
	}

	// Step 2: Check if splitting is needed
	if !needsSplitting(mp3Files, cueFiles) {
		log.Printf("No audio splitting needed (found %d MP3 file(s), %d CUE file(s))",
			len(mp3Files), len(cueFiles))
		return result, nil
	}

	// Step 3: Splitting is needed - prepare for processing
	mp3File := mp3Files[0]
	cueFile := cueFiles[0]

	result.OriginalFile = mp3File
	result.CueFile = cueFile

	log.Printf("Detected CUE file '%s' with single MP3 '%s', splitting audio...",
		filepath.Base(cueFile), filepath.Base(mp3File))

	// Step 4: Perform the split
	if err := p.splitAudio(ctx, mp3File); err != nil {
		return nil, err // Strict mode: fail on error
	}

	// Step 5: Determine the output directory created by m4b-tool
	// m4b-tool creates a directory named {basename}_splitted
	baseName := strings.TrimSuffix(filepath.Base(mp3File), filepath.Ext(mp3File))
	splitDir := filepath.Join(outputDir, baseName+"_splitted")

	result.Processed = true
	result.SplitFilesDir = splitDir

	log.Printf("Audio splitting completed successfully")
	log.Printf("Split files saved to: %s", splitDir)

	return result, nil
}

// needsSplitting determines if audio splitting is required.
// Returns true if there is exactly 1 MP3 file and at least 1 CUE file,
// which indicates an unsplit audiobook.
func needsSplitting(mp3Files, cueFiles []string) bool {
	return len(mp3Files) == 1 && len(cueFiles) > 0
}

// splitAudio calls m4b-tool to split the audio file according to the CUE sheet.
// This function implements strict error handling - any failure is fatal.
func (p *AudioProcessor) splitAudio(ctx context.Context, mp3File string) error {
	// Check if m4b-tool is available in PATH
	m4bToolPath, err := exec.LookPath(p.M4bToolPath)
	if err != nil {
		return fmt.Errorf(
			"m4b-tool not found in PATH.\n"+
				"\n"+
				"Please install m4b-tool:\n"+
				"  - macOS:  brew install sandreas/tap/m4b-tool\n"+
				"  - Linux:  see https://github.com/sandreas/m4b-tool\n"+
				"  - Docker: docker pull sandreas/m4b-tool:latest\n"+
				"\n"+
				"After installation, verify with: m4b-tool --version\n"+
				"\n"+
				"Original error: %w", err)
	}

	// Prepare m4b-tool arguments
	args := []string{
		"split",
		"--audio-format", p.OutputFormat,
		"--audio-bitrate", p.AudioBitrate,
		"--audio-channels", fmt.Sprintf("%d", p.AudioChannels),
		"--audio-samplerate", fmt.Sprintf("%d", p.AudioSamplerate),
		mp3File,
	}

	// Create the command with context for cancellation support
	cmd := exec.CommandContext(ctx, m4bToolPath, args...)

	// Set working directory to the parent of the MP3 file
	cmd.Dir = filepath.Dir(mp3File)

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Running: %s %s", p.M4bToolPath, strings.Join(args, " "))

	// Execute m4b-tool
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"m4b-tool split failed: %w\n"+
				"\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"You can try running this command manually:\n"+
				"\n"+
				"  cd %s\n"+
				"  m4b-tool split \\\n"+
				"    --audio-format mp3 \\\n"+
				"    --audio-bitrate 96k \\\n"+
				"    --audio-channels 1 \\\n"+
				"    --audio-samplerate 22050 \\\n"+
				"    %s\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n",
			err, filepath.Dir(mp3File), filepath.Base(mp3File))
	}

	return nil
}

// findFilesByExt finds all files with the given extension in a directory.
// The extension should include the leading dot (e.g., ".mp3", ".cue").
// Returns an empty slice if no files are found (not an error).
func findFilesByExt(dir string, ext string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	// Normalize extension to lowercase for case-insensitive matching
	ext = strings.ToLower(ext)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Case-insensitive extension matching
		if strings.HasSuffix(strings.ToLower(entry.Name()), ext) {
			fullPath := filepath.Join(dir, entry.Name())
			files = append(files, fullPath)
		}
	}

	return files, nil
}
