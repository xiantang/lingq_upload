# lingq_upload

## Purpose
This project automates uploading audiobooks to [LingQ](https://www.lingq.com) using the [LingQ API](https://www.lingq.com/apidocs/).

It consists of two main components:
1. **Go downloader** - Downloads books from sites like english-e-reader.net
2. **Python uploader** - Uploads books (EPUB + MP3s) to LingQ

## Quick Start

### 1. Setup Environment

Create a `.env` file in the project root:

```bash
# Get your API key from https://www.lingq.com/accounts/apikey
APIKey="Token YOUR_API_KEY_HERE"

# Language code (en for English, fr for French, etc.)
postAddress="https://www.lingq.com/api/v2/en/lessons/"

# Status: "shared" for public, "private" for copyrighted material
status="shared"
```

### 2. Download a Book (Go)

```bash
# Download from english-e-reader.net
go run ./cmd/download_book -book plastic-louise-spilsbury -out ./downloads

# Or use full URL
go run ./cmd/download_book -book https://english-e-reader.net/book/plastic-louise-spilsbury -out ./downloads
```

This creates a directory with:
```
downloads/plastic-louise-spilsbury/
├── metadata.json          # Book info (title, level, tags, description)
├── plastic-louise-spilsbury.epub
├── plastic-louise-spilsbury.mp3  # May need splitting
├── plastic-louise-spilsbury.cue
└── cover.jpg
```

### 3. Upload to LingQ (Python)

**Simple usage** (one directory with everything):
```bash
python3 upload_book.py downloads/plastic-louise-spilsbury
```

**With verbose logging**:
```bash
python3 upload_book.py downloads/plastic-louise-spilsbury -v
```

**Override metadata**:
```bash
python3 upload_book.py my-book --title "Custom Title" --level "Advanced 1" -v
```

**Alternative syntax**:
```bash
python3 upload_book.py -d downloads/my-book -v
```

## Directory Structure

The uploader supports two directory formats:

### Format 1: Flat structure (Go downloader output)
```
book-name/
├── metadata.json
├── book-name.epub
├── cover.jpg
├── Chapter_01.mp3
├── Chapter_02.mp3
└── ...
```

### Format 2: Legacy format with _splitted subdirectory
```
book-name/
├── metadata.json
├── book-name.epub
└── book-name_splitted/
    ├── cover.jpg
    ├── Chapter_01.mp3
    ├── Chapter_02.mp3
    └── ...
```

Both formats are automatically detected.

## Requirements

**Directory must contain:**
- ✅ `metadata.json` (required) - Book metadata
- ✅ `*.epub` file (required) - Book text
- ✅ Multiple `*.mp3` files (required) - Chapter audio (one per chapter)
- ✅ `cover.jpg` or `cover.png` (optional) - Will extract from EPUB if missing

**Note:** If you have a single large MP3 file + CUE, you need to split it into chapters first.

## Command Reference

### Upload Book

```bash
# Basic usage
python3 upload_book.py <directory>

# Options
python3 upload_book.py <directory> [options]

  -d, --dir DIRECTORY        Alternative to positional directory argument
  --title TITLE              Override title from metadata.json
  --level LEVEL              Override level (Beginner 1/2, Intermediate 1/2, Advanced 1/2)
  --tags TAGS                Override tags (comma-separated)
  -v, --verbose              Enable verbose debug logging
  -h, --help                 Show help message

# Examples
python3 upload_book.py downloads/my-book
python3 upload_book.py downloads/my-book -v
python3 upload_book.py downloads/my-book --title "Custom Title" --level "Advanced 1"
python3 upload_book.py -d downloads/my-book --tags "fiction,classic,novel" -v
```

### Download Book (Go)

```bash
# Download book
go run ./cmd/download_book -book <slug-or-url> -out <directory>

# Options
  -book, -b <value>          Book slug, /book/<slug>, or full URL (required)
  -out <directory>           Output directory (default: ./downloads)
  -skip-unzip                Skip extracting MP3 zip archive

# Examples
go run ./cmd/download_book -book plastic-louise-spilsbury -out ./downloads
go run ./cmd/download_book -book /book/body-on-the-rocks-denise-kirby -out ./downloads
go run ./cmd/download_book -book https://english-e-reader.net/book/the-goldbug-edgar-allan-poe -out ./books
```

## metadata.json Format

```json
{
    "title": "Book Title - Author Name - English-e-reader",
    "level": "Beginner 1",
    "author": "Author Name",
    "description": "Book description...",
    "tags": ["tag1", "tag2", "tag3"]
}
```

**Levels:** `Beginner 1`, `Beginner 2`, `Intermediate 1`, `Intermediate 2`, `Advanced 1`, `Advanced 2`

**Tags:** Maximum 9 tags (script adds "book" tag automatically = 10 total max)

## Troubleshooting

**"metadata.json not found"**
- Ensure your directory contains `metadata.json`
- Use the Go downloader to automatically generate it

**"No MP3 files found"**
- Check that you have multiple MP3 files (one per chapter)
- If you have a single large MP3 + CUE file, split it first using a tool like `mp3splt`

**"Chapter count must match MP3 count"**
- The EPUB must have the same number of chapters as MP3 files
- Check your EPUB structure and MP3 file count

**"Skipping large MP3 file"**
- Files over 100MB are assumed to be unsplit
- Use `mp3splt` or similar to split based on CUE file

## Go Downloader Details

- **Build/run:** `go run ./cmd/download_book -book <slug> -out ./downloads`
- **Flags:** `-book` (or `-b`) accepts a slug, `/book/<slug>`, or full english-e-reader URL; `-out` sets the destination root; `-skip-unzip` skips extracting the mp3 zip
- **Extensibility:** Downloaders are provider-based (see `internal/downloader`). Add a new provider implementing the `Provider` interface and register it in `cmd/download_book/main.go` to support more sites

## Credits

- [mescyn](https://www.lingq.com/en/learn/fr/web/community/forum/lingq-developer-forum/python-uploading-audio-via-api) - LingQ Developer Forum
- [beeman](https://www.lingq.com/en/learn/fr/web/community/forum/lingq-developer-forum/python-example-for-creating-a-lesson-for-a-course) - LingQ Developer Forum
- [@gbouthenot](https://github.com/gbouthenot) - For making [mp3splitter-js](https://github.com/gbouthenot/mp3splitter-js)

## References

- [LingQ API Documentation](https://www.lingq.com/apidocs/)
- [Django REST Framework File Upload](https://goodcode.io/articles/django-rest-framework-file-upload/)

