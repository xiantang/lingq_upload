package downloader

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const englishEReaderBaseURL = "https://english-e-reader.net"

// EnglishEReaderOptions controls provider behavior.
type EnglishEReaderOptions struct {
	SkipUnzip  bool
	HTTPClient *http.Client
}

// EnglishEReaderProvider downloads books from english-e-reader.net.
type EnglishEReaderProvider struct {
	opts   EnglishEReaderOptions
	client *http.Client
}

func NewEnglishEReaderProvider(opts EnglishEReaderOptions) *EnglishEReaderProvider {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &EnglishEReaderProvider{
		opts:   opts,
		client: client,
	}
}

func (p *EnglishEReaderProvider) Name() string { return "english-e-reader" }

func (p *EnglishEReaderProvider) Match(input string) bool {
	if strings.Contains(input, "english-e-reader.net") {
		return true
	}
	trimmed := strings.TrimPrefix(input, "/")
	if strings.HasPrefix(trimmed, "book/") {
		return true
	}
	return trimmed != "" && !strings.Contains(trimmed, "/")
}

func (p *EnglishEReaderProvider) Download(ctx context.Context, input string, outputRoot string) (*Result, error) {
	slug, err := p.extractSlug(input)
	if err != nil {
		return nil, err
	}

	outputDir, err := filepath.Abs(filepath.Join(outputRoot, slug))
	if err != nil {
		return nil, fmt.Errorf("resolve output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	pageContent, err := p.fetchPage(ctx, slug)
	if err != nil {
		return nil, err
	}
	availableFormats := detectAvailableFormats(pageContent)

	meta := parseEnglishEReaderMetadata(pageContent)
	metaPath := filepath.Join(outputDir, "metadata.json")
	if err := writeJSON(metaPath, meta); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	files := []string{metaPath}
	formats := []struct {
		format string
		ext    string
	}{
		{"epub", ".epub"},
		{"mp3", ".mp3"},
		{"cue", ".cue"},
		{"mp3zip", ".zip"},
	}

	for _, f := range formats {
		if len(availableFormats) > 0 && !availableFormats[f.format] {
			log.Printf("format %s not listed on page; skipping", f.format)
			continue
		}

		target := filepath.Join(outputDir, slug+f.ext)
		if err := p.downloadFile(ctx, slug, f.format, target); err != nil {
			var statusErr *httpStatusError
			if errors.As(err, &statusErr) && statusErr.StatusCode() == http.StatusNotFound {
				log.Printf("format %s returned 404; skipping", f.format)
				continue
			}
			return nil, fmt.Errorf("download %s: %w", f.format, err)
		}
		files = append(files, target)

		if f.format == "mp3zip" && !p.opts.SkipUnzip {
			targetDir := filepath.Join(outputDir, slug+"_splitted")
			if err := unzipArchive(target, targetDir); err != nil {
				return nil, fmt.Errorf("unzip mp3 archive: %w", err)
			}
			files = append(files, targetDir)
		}
	}

	return &Result{
		Provider:     p.Name(),
		OutputDir:    outputDir,
		Files:        files,
		MetadataPath: metaPath,
	}, nil
}

func (p *EnglishEReaderProvider) extractSlug(input string) (string, error) {
	if input == "" {
		return "", errors.New("empty input")
	}
	if strings.HasPrefix(input, "http") {
		u, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("parse url: %w", err)
		}
		slug := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
		slug = strings.TrimPrefix(slug, "book/")
		if slug == "" {
			return "", fmt.Errorf("cannot determine slug from %q", input)
		}
		return slug, nil
	}
	trimmed := strings.TrimPrefix(input, "/")
	slug := strings.TrimPrefix(trimmed, "book/")
	slug = strings.Trim(slug, "/")
	if slug == "" {
		return "", fmt.Errorf("cannot determine slug from %q", input)
	}
	return slug, nil
}

func (p *EnglishEReaderProvider) fetchPage(ctx context.Context, slug string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, englishEReaderBaseURL+"/book/"+slug, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type httpStatusError struct {
	status int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("unexpected status code %d", e.status)
}

func (e *httpStatusError) StatusCode() int {
	return e.status
}

func (p *EnglishEReaderProvider) downloadFile(ctx context.Context, slug string, format string, target string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/download?link=%s&format=%s", englishEReaderBaseURL, slug, format), nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &httpStatusError{status: resp.StatusCode}
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

type englishEReaderMetadata struct {
	Title       string   `json:"title"`
	Level       string   `json:"level"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func parseEnglishEReaderMetadata(html string) englishEReaderMetadata {
	title := extractFirst(html, `<title>([^<]+)</title>`)
	author := ""
	if title != "" {
		parts := strings.Split(title, " - ")
		if len(parts) > 1 {
			author = strings.TrimSpace(parts[1])
		}
	}
	description := extractFirst(html, `<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
	level := mapEnglishLevel(findFirstLevel(html))
	tags := extractAll(html, `<span[^>]*class=["'][^"']*label[^"']*label-default[^"']*["'][^>]*>([^<]+)</span>`)

	return englishEReaderMetadata{
		Title:       title,
		Level:       level,
		Author:      author,
		Description: description,
		Tags:        tags,
	}
}

func detectAvailableFormats(html string) map[string]bool {
	re := regexp.MustCompile(`download\\?link=[^&]+&format=([a-zA-Z0-9]+)`)
	matches := re.FindAllStringSubmatch(strings.ToLower(html), -1)
	formats := make(map[string]bool, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			formats[strings.ToLower(m[1])] = true
		}
	}
	return formats
}

func extractFirst(text string, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return strings.TrimSpace(htmlUnescape(matches[1]))
	}
	return ""
}

func extractAll(text string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)
	var results []string
	for _, m := range matches {
		if len(m) >= 2 {
			results = append(results, strings.TrimSpace(htmlUnescape(m[1])))
		}
	}
	return results
}

func findFirstLevel(html string) string {
	levels := []string{
		"A1 Starter",
		"A2 Elementary",
		"B1 Pre-Intermediate",
		"B1+ Intermediate",
		"B2 Intermediate-Plus",
		"B2+ Upper-Intermediate",
		"C1 Advanced",
		"C2 Unabridged",
	}
	for _, lvl := range levels {
		if strings.Contains(html, lvl) {
			return lvl
		}
	}
	return ""
}

func mapEnglishLevel(original string) string {
	mapping := map[string]string{
		"A1 Starter":             "Beginner 1",
		"A2 Elementary":          "Beginner 2",
		"B1 Pre-Intermediate":    "Intermediate 1",
		"B1+ Intermediate":       "Intermediate 1",
		"B2 Intermediate-Plus":   "Intermediate 2",
		"B2+ Upper-Intermediate": "Intermediate 2",
		"C1 Advanced":            "Advanced 1",
		"C2 Unabridged":          "Advanced 2",
	}
	if val, ok := mapping[original]; ok {
		return val
	}
	return "Unknown Level"
}

func htmlUnescape(s string) string {
	// Minimal unescape for common entities used in titles.
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
	}
	for k, v := range replacements {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func unzipArchive(zipPath, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		destPath := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return err
		}
		srcFile.Close()
		dstFile.Close()
	}
	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
