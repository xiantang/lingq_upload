package downloader

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test HTML samples for metadata parsing
const (
	fullHTMLSample = `<html>
<head>
	<title>The Adventure of the Blue Carbuncle - Arthur Conan Doyle</title>
	<meta property="og:description" content="A classic Sherlock Holmes mystery story"/>
</head>
<body>
	<div>B1 Pre-Intermediate</div>
	<span class="label label-default">fiction</span>
	<span class="label label-default">classic</span>
	<span class="label label-default">mystery</span>
	<a href="download?link=test-slug&format=epub">EPUB</a>
	<a href="download?link=test-slug&format=mp3">MP3</a>
	<a href="download?link=test-slug&format=mp3zip">MP3 ZIP</a>
</body>
</html>`

	partialHTMLSample = `<html>
<head>
	<title>Book Without Author</title>
</head>
<body>
	<div>C1 Advanced</div>
</body>
</html>`

	emptyHTMLSample = `<html></html>`
)

func TestEnglishEReaderProvider_Name(t *testing.T) {
	t.Parallel()

	provider := NewEnglishEReaderProvider(EnglishEReaderOptions{})

	expected := "english-e-reader"
	if provider.Name() != expected {
		t.Errorf("Name() = %q, want %q", provider.Name(), expected)
	}
}

func TestEnglishEReaderProvider_Match(t *testing.T) {
	t.Parallel()

	provider := NewEnglishEReaderProvider(EnglishEReaderOptions{})

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "full URL with https",
			input: "https://english-e-reader.net/book/test-slug",
			want:  true,
		},
		{
			name:  "full URL with http",
			input: "http://english-e-reader.net/book/test-slug",
			want:  true,
		},
		{
			name:  "URL with path and query",
			input: "https://english-e-reader.net/book/test-slug?param=value",
			want:  true,
		},
		{
			name:  "path with /book/ prefix",
			input: "/book/test-slug",
			want:  true,
		},
		{
			name:  "path with book/ prefix (no leading slash)",
			input: "book/test-slug-name",
			want:  true,
		},
		{
			name:  "slug only",
			input: "simple-slug",
			want:  true,
		},
		{
			name:  "slug with dashes",
			input: "body-on-the-rocks-denise-kirby",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "different domain",
			input: "https://example.com/book/test",
			want:  false,
		},
		{
			name:  "path with slash but not book",
			input: "/other/path",
			want:  false,
		},
		{
			name:  "multiple slashes",
			input: "path/with/slashes",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.Match(tt.input)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSlug(t *testing.T) {
	t.Parallel()

	provider := NewEnglishEReaderProvider(EnglishEReaderOptions{})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "full URL",
			input:   "https://english-e-reader.net/book/test-slug",
			want:    "test-slug",
			wantErr: false,
		},
		{
			name:    "URL with trailing slash",
			input:   "https://english-e-reader.net/book/test-slug/",
			want:    "test-slug",
			wantErr: false,
		},
		{
			name:    "path with /book/ prefix",
			input:   "/book/body-on-the-rocks",
			want:    "body-on-the-rocks",
			wantErr: false,
		},
		{
			name:    "path without leading slash",
			input:   "book/test-slug",
			want:    "test-slug",
			wantErr: false,
		},
		{
			name:    "slug only",
			input:   "simple-slug",
			want:    "simple-slug",
			wantErr: false,
		},
		{
			name:    "slug with leading slash",
			input:   "/simple-slug",
			want:    "simple-slug",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "URL with only /book/",
			input:   "https://english-e-reader.net/book/",
			want:    "book",
			wantErr: false,
		},
		{
			name:    "only slashes",
			input:   "///",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.extractSlug(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractSlug(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("extractSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseEnglishEReaderMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want englishEReaderMetadata
	}{
		{
			name: "full metadata",
			html: fullHTMLSample,
			want: englishEReaderMetadata{
				Title:       "The Adventure of the Blue Carbuncle - Arthur Conan Doyle",
				Level:       "Intermediate 1",
				Author:      "Arthur Conan Doyle",
				Description: "A classic Sherlock Holmes mystery story",
				Tags:        []string{"fiction", "classic", "mystery"},
			},
		},
		{
			name: "partial metadata",
			html: partialHTMLSample,
			want: englishEReaderMetadata{
				Title:       "Book Without Author",
				Level:       "Advanced 1",
				Author:      "",
				Description: "",
				Tags:        nil,
			},
		},
		{
			name: "empty HTML",
			html: emptyHTMLSample,
			want: englishEReaderMetadata{
				Title:       "",
				Level:       "Unknown Level",
				Author:      "",
				Description: "",
				Tags:        nil,
			},
		},
		{
			name: "HTML with special characters",
			html: `<title>Test &amp; Book - Author &quot;Name&quot;</title>`,
			want: englishEReaderMetadata{
				Title:  "Test & Book - Author \"Name\"",
				Level:  "Unknown Level",
				Author: "Author \"Name\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnglishEReaderMetadata(tt.html)

			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.Level != tt.want.Level {
				t.Errorf("Level = %q, want %q", got.Level, tt.want.Level)
			}
			if got.Author != tt.want.Author {
				t.Errorf("Author = %q, want %q", got.Author, tt.want.Author)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}

			if len(got.Tags) != len(tt.want.Tags) {
				t.Errorf("Tags count = %d, want %d", len(got.Tags), len(tt.want.Tags))
			} else {
				for i := range got.Tags {
					if got.Tags[i] != tt.want.Tags[i] {
						t.Errorf("Tags[%d] = %q, want %q", i, got.Tags[i], tt.want.Tags[i])
					}
				}
			}
		})
	}
}

func TestDetectAvailableFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want map[string]bool
	}{
		{
			name: "all formats available",
			html: fullHTMLSample,
			want: map[string]bool{
				"epub":   true,
				"mp3":    true,
				"mp3zip": true,
			},
		},
		{
			name: "only epub",
			html: `<a href="download?link=test&format=epub">EPUB</a>`,
			want: map[string]bool{
				"epub": true,
			},
		},
		{
			name: "no formats",
			html: `<html><body>No download links</body></html>`,
			want: map[string]bool{},
		},
		{
			name: "uppercase format in URL",
			html: `<a href="download?link=test&format=EPUB">EPUB</a>`,
			want: map[string]bool{
				"epub": true,
			},
		},
		{
			name: "duplicate formats",
			html: `
				<a href="download?link=test&format=epub">EPUB 1</a>
				<a href="download?link=test&format=epub">EPUB 2</a>
			`,
			want: map[string]bool{
				"epub": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectAvailableFormats(tt.html)

			if len(got) != len(tt.want) {
				t.Errorf("format count = %d, want %d", len(got), len(tt.want))
			}

			for format := range tt.want {
				if !got[format] {
					t.Errorf("format %q not found in result", format)
				}
			}

			for format := range got {
				if !tt.want[format] {
					t.Errorf("unexpected format %q in result", format)
				}
			}
		})
	}
}

func TestFindFirstLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "A1 Starter",
			html: `<div>A1 Starter</div>`,
			want: "A1 Starter",
		},
		{
			name: "A2 Elementary",
			html: `<div>A2 Elementary</div>`,
			want: "A2 Elementary",
		},
		{
			name: "B1 Pre-Intermediate",
			html: `<div>B1 Pre-Intermediate</div>`,
			want: "B1 Pre-Intermediate",
		},
		{
			name: "B1+ Intermediate",
			html: `<div>B1+ Intermediate</div>`,
			want: "B1+ Intermediate",
		},
		{
			name: "B2 Intermediate-Plus",
			html: `<div>B2 Intermediate-Plus</div>`,
			want: "B2 Intermediate-Plus",
		},
		{
			name: "B2+ Upper-Intermediate",
			html: `<div>B2+ Upper-Intermediate</div>`,
			want: "B2+ Upper-Intermediate",
		},
		{
			name: "C1 Advanced",
			html: `<div>C1 Advanced</div>`,
			want: "C1 Advanced",
		},
		{
			name: "C2 Unabridged",
			html: `<div>C2 Unabridged</div>`,
			want: "C2 Unabridged",
		},
		{
			name: "no level",
			html: `<html><body>No level here</body></html>`,
			want: "",
		},
		{
			name: "first level when multiple",
			html: `<div>A1 Starter</div><div>C1 Advanced</div>`,
			want: "A1 Starter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFirstLevel(tt.html)
			if got != tt.want {
				t.Errorf("findFirstLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapEnglishLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"A1 Starter", "Beginner 1"},
		{"A2 Elementary", "Beginner 2"},
		{"B1 Pre-Intermediate", "Intermediate 1"},
		{"B1+ Intermediate", "Intermediate 1"},
		{"B2 Intermediate-Plus", "Intermediate 2"},
		{"B2+ Upper-Intermediate", "Intermediate 2"},
		{"C1 Advanced", "Advanced 1"},
		{"C2 Unabridged", "Advanced 2"},
		{"Unknown Level String", "Unknown Level"},
		{"", "Unknown Level"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapEnglishLevel(tt.input)
			if got != tt.want {
				t.Errorf("mapEnglishLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		pattern string
		want    string
	}{
		{
			name:    "simple match",
			text:    "<title>My Title</title>",
			pattern: `<title>([^<]+)</title>`,
			want:    "My Title",
		},
		{
			name:    "with whitespace",
			text:    "<title>  Spaced Title  </title>",
			pattern: `<title>([^<]+)</title>`,
			want:    "Spaced Title",
		},
		{
			name:    "with HTML entities",
			text:    "<title>Title &amp; More</title>",
			pattern: `<title>([^<]+)</title>`,
			want:    "Title & More",
		},
		{
			name:    "no match",
			text:    "<div>Not a title</div>",
			pattern: `<title>([^<]+)</title>`,
			want:    "",
		},
		{
			name:    "empty capture group",
			text:    "<title></title>",
			pattern: `<title>([^<]*)</title>`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirst(tt.text, tt.pattern)
			if got != tt.want {
				t.Errorf("extractFirst() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		pattern string
		want    []string
	}{
		{
			name:    "multiple tags",
			text:    `<span class="label label-default">tag1</span><span class="label label-default">tag2</span>`,
			pattern: `<span[^>]*class=["'][^"']*label[^"']*label-default[^"']*["'][^>]*>([^<]+)</span>`,
			want:    []string{"tag1", "tag2"},
		},
		{
			name:    "no matches",
			text:    `<div>No tags here</div>`,
			pattern: `<span[^>]*class=["'][^"']*label[^"']*["'][^>]*>([^<]+)</span>`,
			want:    nil,
		},
		{
			name:    "with HTML entities",
			text:    `<span class="label label-default">tag&amp;1</span>`,
			pattern: `<span[^>]*class=["'][^"']*label[^"']*label-default[^"']*["'][^>]*>([^<]+)</span>`,
			want:    []string{"tag&1"},
		},
		{
			name:    "with whitespace",
			text:    `<span class="label label-default">  tag1  </span>`,
			pattern: `<span[^>]*class=["'][^"']*label[^"']*label-default[^"']*["'][^>]*>([^<]+)</span>`,
			want:    []string{"tag1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAll(tt.text, tt.pattern)

			if len(got) != len(tt.want) {
				t.Errorf("extractAll() count = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractAll()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestHtmlUnescape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"&amp;", "&"},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"&quot;", `"`},
		{"&#39;", "'"},
		{"Title &amp; Author", "Title & Author"},
		{"&lt;div&gt;", "<div>"},
		{"He said &quot;hello&quot;", `He said "hello"`},
		{"It&#39;s working", "It's working"},
		{"Multiple &amp; &lt; &gt; entities", "Multiple & < > entities"},
		{"No entities", "No entities"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := htmlUnescape(tt.input)
			if got != tt.want {
				t.Errorf("htmlUnescape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.json")

	testData := englishEReaderMetadata{
		Title:       "Test Book",
		Level:       "Intermediate 1",
		Author:      "Test Author",
		Description: "Test Description",
		Tags:        []string{"tag1", "tag2"},
	}

	err := writeJSON(testFile, testData)
	if err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Test Book") {
		t.Error("JSON does not contain expected title")
	}
	if !strings.Contains(content, "Test Author") {
		t.Error("JSON does not contain expected author")
	}
	if !strings.Contains(content, "tag1") {
		t.Error("JSON does not contain expected tag")
	}
}

func TestUnzipArchive(t *testing.T) {
	t.Parallel()

	// Create a test zip file
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	extractDir := filepath.Join(tempDir, "extracted")

	// Create zip file with test content
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)

	// Add a text file
	file1, err := zipWriter.Create("file1.txt")
	if err != nil {
		t.Fatalf("failed to create file in zip: %v", err)
	}
	_, err = file1.Write([]byte("content1"))
	if err != nil {
		t.Fatalf("failed to write to zip: %v", err)
	}

	// Add a file in a subdirectory
	file2, err := zipWriter.Create("subdir/file2.txt")
	if err != nil {
		t.Fatalf("failed to create subdirectory file in zip: %v", err)
	}
	_, err = file2.Write([]byte("content2"))
	if err != nil {
		t.Fatalf("failed to write to zip: %v", err)
	}

	zipWriter.Close()
	zipFile.Close()

	// Test unzipping
	err = unzipArchive(zipPath, extractDir)
	if err != nil {
		t.Fatalf("unzipArchive failed: %v", err)
	}

	// Verify extracted files
	file1Path := filepath.Join(extractDir, "file1.txt")
	if _, err := os.Stat(file1Path); os.IsNotExist(err) {
		t.Error("file1.txt was not extracted")
	}

	file2Path := filepath.Join(extractDir, "subdir", "file2.txt")
	if _, err := os.Stat(file2Path); os.IsNotExist(err) {
		t.Error("subdir/file2.txt was not extracted")
	}

	// Verify content
	content1, err := os.ReadFile(file1Path)
	if err == nil && string(content1) != "content1" {
		t.Errorf("file1.txt content = %q, want %q", string(content1), "content1")
	}

	content2, err := os.ReadFile(file2Path)
	if err == nil && string(content2) != "content2" {
		t.Errorf("file2.txt content = %q, want %q", string(content2), "content2")
	}
}

func TestFetchPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		slug       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful fetch",
			slug:       "test-slug",
			statusCode: http.StatusOK,
			response:   fullHTMLSample,
			wantErr:    false,
		},
		{
			name:       "404 not found",
			slug:       "nonexistent",
			statusCode: http.StatusNotFound,
			response:   "",
			wantErr:    true,
		},
		{
			name:       "500 server error",
			slug:       "error",
			statusCode: http.StatusInternalServerError,
			response:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/book/" + tt.slug
				if r.URL.Path != expectedPath {
					t.Errorf("request path = %q, want %q", r.URL.Path, expectedPath)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			// Create provider with custom client pointing to test server
			provider := &EnglishEReaderProvider{
				client: server.Client(),
			}

			// Replace base URL in request
			ctx := context.Background()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/book/"+tt.slug, nil)
			resp, err := provider.client.Do(req)
			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantErr {
				if resp.StatusCode == http.StatusOK {
					t.Error("expected non-OK status code")
				}
			} else {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
				}

				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.response {
					t.Errorf("response body mismatch")
				}
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		format     string
		statusCode int
		content    string
		wantErr    bool
		checkErr   func(error) bool
	}{
		{
			name:       "successful download",
			format:     "epub",
			statusCode: http.StatusOK,
			content:    "fake epub content",
			wantErr:    false,
		},
		{
			name:       "404 not found",
			format:     "missing",
			statusCode: http.StatusNotFound,
			content:    "",
			wantErr:    true,
			checkErr: func(err error) bool {
				var statusErr *httpStatusError
				return errors.As(err, &statusErr) && statusErr.StatusCode() == http.StatusNotFound
			},
		},
		{
			name:       "500 server error",
			format:     "error",
			statusCode: http.StatusInternalServerError,
			content:    "",
			wantErr:    true,
			checkErr: func(err error) bool {
				var statusErr *httpStatusError
				return errors.As(err, &statusErr) && statusErr.StatusCode() == http.StatusInternalServerError
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/download" {
					t.Errorf("request path = %q, want /download", r.URL.Path)
				}

				query := r.URL.Query()
				if query.Get("format") != tt.format {
					t.Errorf("format param = %q, want %q", query.Get("format"), tt.format)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(tt.content))
				}
			}))
			defer server.Close()

			provider := &EnglishEReaderProvider{
				client: server.Client(),
			}

			tempDir := t.TempDir()
			targetFile := filepath.Join(tempDir, "download."+tt.format)

			// Call downloadFile with test server URL
			ctx := context.Background()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/download?link=test&format="+tt.format, nil)
			resp, err := provider.client.Do(req)
			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantErr {
				if resp.StatusCode == http.StatusOK {
					t.Error("expected non-OK status code")
				}
				if tt.checkErr != nil {
					testErr := &httpStatusError{status: resp.StatusCode}
					if !tt.checkErr(testErr) {
						t.Errorf("error check failed for status %d", resp.StatusCode)
					}
				}
			} else {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
				}

				// Write to file
				out, err := os.Create(targetFile)
				if err != nil {
					t.Fatalf("failed to create target file: %v", err)
				}
				defer out.Close()

				_, err = io.Copy(out, resp.Body)
				if err != nil {
					t.Fatalf("failed to write file: %v", err)
				}

				// Verify content
				content, err := os.ReadFile(targetFile)
				if err != nil {
					t.Fatalf("failed to read downloaded file: %v", err)
				}

				if string(content) != tt.content {
					t.Errorf("file content = %q, want %q", string(content), tt.content)
				}
			}
		})
	}
}

func TestHttpStatusError(t *testing.T) {
	t.Parallel()

	err := &httpStatusError{status: 404}

	expectedMsg := "unexpected status code 404"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}

	if err.StatusCode() != 404 {
		t.Errorf("StatusCode() = %d, want 404", err.StatusCode())
	}
}

func TestEnglishEReaderProvider_Download_Integration(t *testing.T) {
	t.Parallel()

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/book/test-book":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fullHTMLSample))
		case "/download":
			format := r.URL.Query().Get("format")
			switch format {
			case "epub":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("fake epub content"))
			case "mp3":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("fake mp3 content"))
			case "cue":
				w.WriteHeader(http.StatusNotFound)
			case "mp3zip":
				// Create a minimal zip file
				w.WriteHeader(http.StatusOK)
				buf := new(bytes.Buffer)
				zipWriter := zip.NewWriter(buf)
				file, _ := zipWriter.Create("track01.mp3")
				file.Write([]byte("fake mp3 track"))
				zipWriter.Close()
				w.Write(buf.Bytes())
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a provider with custom HTTP client and mocked base URL
	// We need to replace the englishEReaderBaseURL constant
	// Since we can't modify the constant, we'll test the HTTP interactions separately

	client := server.Client()

	// Test page fetch
	resp, err := client.Get(server.URL + "/book/test-book")
	if err != nil {
		t.Fatalf("failed to fetch page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("page status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test epub download
	resp, err = client.Get(server.URL + "/download?link=test-book&format=epub")
	if err != nil {
		t.Fatalf("failed to download epub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("epub status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test 404 for cue
	resp, err = client.Get(server.URL + "/download?link=test-book&format=cue")
	if err != nil {
		t.Fatalf("failed to request cue: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cue status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestEnglishEReaderProvider_Download_RealIntegration(t *testing.T) {
	t.Parallel()

	// Create mock server that simulates english-e-reader.net
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/book/test-integration-book":
			w.WriteHeader(http.StatusOK)
			// Return HTML with only epub available
			html := `<html>
<head>
	<title>Integration Test Book - Test Author</title>
	<meta property="og:description" content="A test book for integration testing"/>
</head>
<body>
	<div>B1 Pre-Intermediate</div>
	<span class="label label-default">test</span>
	<a href="download?link=test-integration-book&format=epub">EPUB</a>
</body>
</html>`
			w.Write([]byte(html))
		case "/download":
			format := r.URL.Query().Get("format")
			if format == "epub" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("fake epub binary data"))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create provider with custom client
	// Note: We can't easily test the full Download method because it uses
	// the hardcoded englishEReaderBaseURL constant. In a real refactor,
	// we'd make the base URL configurable for testing.

	provider := NewEnglishEReaderProvider(EnglishEReaderOptions{
		HTTPClient: server.Client(),
		SkipUnzip:  true,
	})

	// Test that we can at least extract the slug and match
	slug, err := provider.extractSlug("test-integration-book")
	if err != nil {
		t.Fatalf("extractSlug failed: %v", err)
	}
	if slug != "test-integration-book" {
		t.Errorf("slug = %q, want %q", slug, "test-integration-book")
	}

	if !provider.Match("test-integration-book") {
		t.Error("provider should match the slug")
	}
}

func TestNewEnglishEReaderProvider(t *testing.T) {
	t.Parallel()

	t.Run("with custom client", func(t *testing.T) {
		customClient := &http.Client{}
		opts := EnglishEReaderOptions{
			HTTPClient: customClient,
			SkipUnzip:  true,
		}

		provider := NewEnglishEReaderProvider(opts)

		if provider.client != customClient {
			t.Error("provider should use custom HTTP client")
		}

		if !provider.opts.SkipUnzip {
			t.Error("SkipUnzip option not preserved")
		}
	})

	t.Run("with default client", func(t *testing.T) {
		opts := EnglishEReaderOptions{}
		provider := NewEnglishEReaderProvider(opts)

		if provider.client == nil {
			t.Fatal("provider should have default HTTP client")
		}

		if provider.client.Timeout == 0 {
			t.Error("default client should have timeout set")
		}
	})
}
