# AGENTS.md - Coding Agent Guide

## Project Overview

This is a dual-language utility project for automating language learning material uploads to the LingQ platform API. Go handles book downloading from sites like english-e-reader.net, while Python manages content upload and API interactions with LingQ.

**Key components:**
- Go downloader: Provider-based architecture for extensible book downloading
- Python uploaders: Scripts for books, podcasts, and YouTube playlists to LingQ API
- Multimedia handling: EPUB books, MP3 audio, cover images

## Build, Test & Run Commands

### Go Commands

**Build:**
```bash
go build ./cmd/download_book
```

**Run directly:**
```bash
go run ./cmd/download_book -book <slug-or-url> -out ./downloads
```

**Run with flags:**
```bash
# Using slug
go run ./cmd/download_book -book body-on-the-rocks-denise-kirby -out ./downloads

# Using path format
go run ./cmd/download_book -book /book/body-on-the-rocks-denise-kirby -out ./downloads

# Skip unzipping mp3 archives
go run ./cmd/download_book -book <slug> -out ./downloads -skip-unzip
```

**Test:**
```bash
# Run all tests
go test ./internal/downloader -v

# Run specific test
go test ./internal/downloader -run TestProviderMatch -v

# Run with coverage
go test ./internal/downloader -cover
```

**Lint:**
```bash
# No linter configured - recommend installing golangci-lint
golangci-lint run ./...
```

### Python Commands

**Run book uploader:**
```bash
# Simple usage (requires directory with metadata.json)
python3 upload_book.py <directory>

# With verbose logging
python3 upload_book.py <directory> -v

# Override metadata from command line
python3 upload_book.py <directory> --title "Custom Title" --level "Advanced 1" -v

# Alternative syntax using named parameter
python3 upload_book.py -d <directory> -v

# Examples
python3 upload_book.py downloads/plastic-louise-spilsbury
python3 upload_book.py the-adventure-of-the-blue-carbuncle-conan-doyle -v
python3 upload_book.py downloads/my-book --title "My Book" --tags "fiction,classic" -v
```

**Directory structure requirements:**
- Must contain `metadata.json` (with title, level, tags, description)
- Must contain `*.epub` file
- Must contain multiple `*.mp3` chapter files (in root or `<dirname>_splitted/` subdirectory)
- Optional: `cover.jpg` or `cover.png` (will extract from EPUB if missing)

**Run podcast uploader:**
```bash
python3 upload_podcast.py -a <audio_folder> -t "Podcast Title"
python3 upload_podcast_cmd.py  # CLI version
```

**Run YouTube playlist uploader:**
```bash
python3 youtube_playlist_upload.py
```

**Test:**
```bash
# No tests currently exist
# To run a single test (when tests are added):
python3 -m pytest tests/test_upload.py::test_create_collections -v
```

**Lint:**
```bash
# No linter configured - recommend installing ruff or black
ruff check .
black --check .
```

## Environment Configuration

Create a `.env` file in the project root (see `.env_example`):

```bash
APIKey="Token [your-api-key-from-lingq.com/accounts/apikey]"
postAddress="https://www.lingq.com/api/v2/en/lessons/"
status="shared"  # or "private" for copyrighted material
```

## Code Style Guidelines

### Go Code Style

**Imports:**
- Standard library imports only (no external dependencies)
- Group imports: stdlib first
- Use blank lines to separate groups if needed

**Formatting:**
- Use `gofmt` (or `go fmt`) - all Go code should be formatted
- Use tabs for indentation (Go standard)
- Line length: no strict limit, but keep reasonable (~100-120 chars)

**Types:**
- Define interfaces for extensibility (`Provider` interface pattern)
- Use struct types for configuration (`EnglishEReaderOptions`)
- Export types/functions that need to be public (PascalCase)
- Keep internal helpers lowercase

**Naming conventions:**
- Exported: `PascalCase` (e.g., `Provider`, `NewManager`)
- Unexported: `camelCase` (e.g., `extractSlug`, `fetchPage`)
- Interfaces: noun or adjective (e.g., `Provider`, `Downloader`)
- Constructors: `New<TypeName>` (e.g., `NewManager`, `NewEnglishEReaderProvider`)

**Error handling:**
- Always check and return errors explicitly
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use custom error types for specific cases (e.g., `httpStatusError`)
- Don't panic in library code - return errors instead

**Context usage:**
- Accept `context.Context` as first parameter for I/O operations
- Use context for timeouts and cancellation in HTTP requests

**Comments:**
- Document exported types, functions, and constants
- Use godoc format: start with the name being documented
- Example: `// Provider represents a site-specific downloader implementation.`

**File organization:**
- `cmd/` for CLI applications and entry points
- `internal/` for internal packages not meant for external use
- Separate concerns: one provider per file

### Python Code Style

**Imports:**
- Standard library imports first
- Third-party imports second
- Local module imports last
- Separate groups with blank lines
- Example from `upload_book.py`:
  ```python
  import argparse
  import json
  import os
  from glob import glob
  from os.path import basename

  import ebooklib
  import requests
  from bs4 import BeautifulSoup
  from dotenv import load_dotenv
  
  from generate_timestamp import generate_timestamp_for_course
  from update_lesson import update_metadata
  ```

**Formatting:**
- Use 4 spaces for indentation (no tabs)
- Line length: ~80-100 characters preferred
- Use double quotes for strings (existing codebase uses double quotes)

**Types:**
- Type hints not currently used in codebase
- When adding new code, consider adding type hints for clarity

**Naming conventions:**
- Functions/variables: `snake_case` (e.g., `chapter_to_str`, `upload_cover`)
- Constants: `UPPER_SNAKE_CASE` (e.g., `API_KEY`, `POST_ADDRESS`)
- Classes: `PascalCase` (if adding classes)
- Private: prefix with `_` (e.g., `_internal_helper`)

**Error handling:**
- Use try/except blocks for API calls and file operations
- Raise exceptions with descriptive messages
- Example: `raise Exception("Sorry, chapters length cannot be zero")`
- Consider using specific exception types (ValueError, FileNotFoundError, etc.)

**API interactions:**
- Use `requests` library for HTTP calls
- Set proper headers with Authorization and Content-Type
- Use `MultipartEncoder` from `requests_toolbelt` for file uploads
- Example pattern:
  ```python
  header = {"Authorization": key, "Content-Type": "application/json"}
  r = requests.post(url, json=body, headers=header)
  result = r.json()
  ```

**File handling:**
- Use `glob()` for file pattern matching
- Use `with` statement for file operations (when possible)
- Check file existence before operations

**Main block:**
- Use `if __name__ == "__main__":` for script entry points
- Parse command-line arguments with `argparse`

## Architecture Patterns

### Go Provider Pattern

The downloader uses a provider-based architecture for extensibility:

1. **Define Provider interface** (`internal/downloader/manager.go`)
2. **Implement provider** (e.g., `EnglishEReaderProvider`)
3. **Register provider** in `cmd/download_book/main.go`

When adding a new provider:
- Implement `Provider` interface: `Name()`, `Match()`, `Download()`
- Add provider-specific options struct if needed
- Register in main.go before calling `manager.Download()`

### Python Upload Pattern

Upload scripts follow this pattern:
1. Load environment variables from `.env`
2. Parse command-line arguments
3. Create LingQ collection (course)
4. Upload cover image (if available)
5. Upload lessons with audio in sequence
6. Update metadata and generate timestamps

**EPUB Format Handling:**
- **Split EPUBs**: Books with chapters named like `split_001.xhtml` are processed as multiple lessons
- **Single-file EPUBs**: Books with one content file (e.g., from Go downloader) are processed as one lesson
- **Strict matching**: Chapter count must equal MP3 count - script will error if mismatched
- **API version**: Uses LingQ API v3 for lesson creation (v2 is deprecated)

### Audio Processing Pipeline

The Go downloader automatically detects and splits unsplit audiobooks during download.

**How it works:**
1. **Detection**: After downloading files, `AudioProcessor` scans the output directory
2. **Conditions**: Splitting triggered when:
   - Exactly one `.cue` file exists
   - Exactly one `.mp3` file exists
   - MP3 file size > 70MB (indicates unsplit audiobook)
3. **Splitting**: Calls `m4b-tool split` with optimized settings for LingQ:
   - Format: MP3 (compatible with LingQ API)
   - Bitrate: 96kbps (balance of quality and file size)
   - Channels: 1 (mono, sufficient for spoken audio)
   - Sample rate: 22050 Hz (reduces file size)
4. **Output**: Split chapters saved to `<book>_splitted/` subdirectory

**Key files:**
- `internal/downloader/processor.go` - Audio processing logic
- `internal/downloader/processor_test.go` - Comprehensive test suite
- `internal/downloader/englishereader.go:119-132` - Integration point

**Error handling:**
- **Strict mode**: Download fails if m4b-tool not found or splitting fails
- **Helpful errors**: Provides installation instructions and manual commands
- **Logging**: Clear progress messages during splitting

**Python uploader validation:**
- Simplified MP3 file validation (assumes Go handled splitting)
- **Defensive check**: Warns if CUE + large MP3 detected (catches manual downloads)
- Provides manual m4b-tool command if splitting needed

**Requirements:**
- `m4b-tool` must be installed and in PATH
- Install: `brew install m4b-tool` (macOS) or via Nix (NixOS)
- Version: Tested with m4b-tool v0.5.2+

## Common Pitfalls

1. **Missing .env file**: Scripts require `.env` with API credentials
2. **File path mismatches**: Ensure chapter count matches audio file count
3. **Tag limits**: LingQ allows max 10 tags per collection
4. **API rate limiting**: No retry logic currently implemented
5. **Error handling**: Limited error handling in Python scripts - check responses
6. **Hardcoded values**: Some collection IDs and paths are hardcoded - parameterize when refactoring

## Testing Guidelines

**Go:**
- Tests exist for downloader components (52.8% coverage)
- Place tests in same package: `<file>_test.go`
- Use table-driven tests for multiple cases
- Mock HTTP calls using `httptest` package
- Test provider matching logic separately
- Example: `TestProcess_MultipleMp3Files` tests audio processing

**Python:**
- No tests currently exist
- When adding tests:
  - Use `pytest` as test framework
  - Place tests in `tests/` directory
  - Mock API calls with `responses` or `unittest.mock`
  - Test metadata parsing and file operations

## Making Changes

1. **Adding new book provider (Go):**
   - Create `internal/downloader/<provider>.go`
   - Implement `Provider` interface
   - Register in `cmd/download_book/main.go`

2. **Modifying upload scripts (Python):**
   - Avoid hardcoding collection IDs or paths
   - Add command-line arguments for configurability
   - Validate file existence before processing
   - Add error handling for API calls

3. **Environment changes:**
   - Update `.env_example` with new variables
   - Document in README.md

## LingQ API Endpoints

- Collections: `https://www.lingq.com/api/v3/en/collections/`
- Lessons: `https://www.lingq.com/api/v2/en/lessons/`
- API docs: `https://www.lingq.com/apidocs/`

## Dependencies

**Go:** Standard library only (Go 1.21+)

**Python:** (inferred - no requirements.txt exists)
- requests
- requests-toolbelt
- python-dotenv
- beautifulsoup4
- ebooklib
- pydub

When adding dependencies, create `requirements.txt` or `pyproject.toml`.
