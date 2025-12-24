package downloader

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// mockProvider is a test implementation of Provider interface
type mockProvider struct {
	name           string
	matchInput     string
	matchResult    bool
	downloadError  error
	downloadResult *Result
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Match(input string) bool {
	if m.matchInput != "" && input != m.matchInput {
		return false
	}
	return m.matchResult
}

func (m *mockProvider) Download(ctx context.Context, input string, outputRoot string) (*Result, error) {
	if m.downloadError != nil {
		return nil, m.downloadError
	}
	if m.downloadResult != nil {
		return m.downloadResult, nil
	}
	return &Result{
		Provider:  m.name,
		OutputDir: filepath.Join(outputRoot, "test-output"),
		Files:     []string{"file1.txt", "file2.txt"},
	}, nil
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	outputRoot := "/test/output"
	manager := NewManager(outputRoot)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.outputRoot != outputRoot {
		t.Errorf("outputRoot = %q, want %q", manager.outputRoot, outputRoot)
	}

	if len(manager.providers) != 0 {
		t.Errorf("initial providers length = %d, want 0", len(manager.providers))
	}
}

func TestRegisterProvider(t *testing.T) {
	t.Parallel()

	manager := NewManager("/test")
	provider1 := &mockProvider{name: "provider1"}
	provider2 := &mockProvider{name: "provider2"}

	manager.RegisterProvider(provider1)
	if len(manager.providers) != 1 {
		t.Errorf("after registering 1 provider, length = %d, want 1", len(manager.providers))
	}

	manager.RegisterProvider(provider2)
	if len(manager.providers) != 2 {
		t.Errorf("after registering 2 providers, length = %d, want 2", len(manager.providers))
	}

	// Verify order is preserved (first registered, first matched)
	if manager.providers[0].Name() != "provider1" {
		t.Errorf("first provider name = %q, want %q", manager.providers[0].Name(), "provider1")
	}
	if manager.providers[1].Name() != "provider2" {
		t.Errorf("second provider name = %q, want %q", manager.providers[1].Name(), "provider2")
	}
}

func TestManager_Download_Success(t *testing.T) {
	t.Parallel()

	outputRoot := t.TempDir()
	manager := NewManager(outputRoot)

	provider := &mockProvider{
		name:        "test-provider",
		matchResult: true,
		downloadResult: &Result{
			Provider:     "test-provider",
			OutputDir:    "relative/path",
			Files:        []string{"file1.txt"},
			MetadataPath: "metadata.json",
		},
	}

	manager.RegisterProvider(provider)

	ctx := context.Background()
	result, err := manager.Download(ctx, "test-input")

	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Provider != "test-provider" {
		t.Errorf("result.Provider = %q, want %q", result.Provider, "test-provider")
	}

	// OutputDir should be made absolute
	if !filepath.IsAbs(result.OutputDir) {
		t.Errorf("OutputDir should be absolute, got %q", result.OutputDir)
	}

	expectedPath := filepath.Join(outputRoot, "relative/path")
	if result.OutputDir != expectedPath {
		t.Errorf("OutputDir = %q, want %q", result.OutputDir, expectedPath)
	}
}

func TestManager_Download_MultipleProviders(t *testing.T) {
	t.Parallel()

	outputRoot := t.TempDir()
	manager := NewManager(outputRoot)

	// First provider doesn't match
	provider1 := &mockProvider{
		name:        "provider1",
		matchResult: false,
	}

	// Second provider matches
	provider2 := &mockProvider{
		name:        "provider2",
		matchResult: true,
		downloadResult: &Result{
			Provider:  "provider2",
			OutputDir: "output2",
			Files:     []string{"file2.txt"},
		},
	}

	// Third provider also matches but should not be called
	provider3 := &mockProvider{
		name:        "provider3",
		matchResult: true,
	}

	manager.RegisterProvider(provider1)
	manager.RegisterProvider(provider2)
	manager.RegisterProvider(provider3)

	ctx := context.Background()
	result, err := manager.Download(ctx, "test-input")

	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Should use provider2 (first matching provider)
	if result.Provider != "provider2" {
		t.Errorf("result.Provider = %q, want %q (first matching provider)", result.Provider, "provider2")
	}
}

func TestManager_Download_NoMatchingProvider(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())

	// Register providers that don't match
	provider1 := &mockProvider{
		name:        "provider1",
		matchResult: false,
	}
	provider2 := &mockProvider{
		name:        "provider2",
		matchResult: false,
	}

	manager.RegisterProvider(provider1)
	manager.RegisterProvider(provider2)

	ctx := context.Background()
	result, err := manager.Download(ctx, "unmatched-input")

	if err == nil {
		t.Fatal("expected error when no provider matches, got nil")
	}

	if result != nil {
		t.Errorf("result should be nil when error occurs, got %+v", result)
	}

	expectedErrMsg := "no provider can handle input"
	if err.Error() != `no provider can handle input "unmatched-input"` {
		t.Errorf("error message should contain %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestManager_Download_ProviderError(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())

	expectedErr := errors.New("provider download failed")
	provider := &mockProvider{
		name:          "error-provider",
		matchResult:   true,
		downloadError: expectedErr,
	}

	manager.RegisterProvider(provider)

	ctx := context.Background()
	result, err := manager.Download(ctx, "test-input")

	if err == nil {
		t.Fatal("expected error from provider, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}

	if result != nil {
		t.Errorf("result should be nil when error occurs, got %+v", result)
	}
}

func TestManager_Download_AbsolutePathHandling(t *testing.T) {
	t.Parallel()

	outputRoot := t.TempDir()

	tests := []struct {
		name           string
		outputDir      string
		shouldBeAbs    bool
		expectedPrefix string
	}{
		{
			name:           "relative path",
			outputDir:      "relative/dir",
			shouldBeAbs:    true,
			expectedPrefix: outputRoot,
		},
		{
			name:           "absolute path",
			outputDir:      "/absolute/path/dir",
			shouldBeAbs:    true,
			expectedPrefix: "",
		},
		{
			name:           "current dir",
			outputDir:      ".",
			shouldBeAbs:    true,
			expectedPrefix: outputRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockProvider{
				name:        "test-provider",
				matchResult: true,
				downloadResult: &Result{
					Provider:  "test-provider",
					OutputDir: tt.outputDir,
					Files:     []string{"file.txt"},
				},
			}

			mgr := NewManager(outputRoot)
			mgr.RegisterProvider(provider)

			ctx := context.Background()
			result, err := mgr.Download(ctx, "test")

			if err != nil {
				t.Fatalf("Download failed: %v", err)
			}

			if !filepath.IsAbs(result.OutputDir) {
				t.Errorf("OutputDir should be absolute, got %q", result.OutputDir)
			}

			if tt.expectedPrefix != "" {
				expectedPath := filepath.Join(tt.expectedPrefix, tt.outputDir)
				if result.OutputDir != expectedPath {
					t.Errorf("OutputDir = %q, want %q", result.OutputDir, expectedPath)
				}
			}
		})
	}
}

func TestManager_Download_NoProviders(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())
	// Don't register any providers

	ctx := context.Background()
	result, err := manager.Download(ctx, "test-input")

	if err == nil {
		t.Fatal("expected error when no providers registered, got nil")
	}

	if result != nil {
		t.Errorf("result should be nil when error occurs, got %+v", result)
	}
}

func TestManager_Download_ContextPropagation(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())

	// Create a custom provider that checks context
	var receivedCtx context.Context

	// Create custom mock that captures context
	customProvider := &contextCapturingProvider{
		name:        "context-checker",
		matchResult: true,
		capturedCtx: &receivedCtx,
	}

	manager.RegisterProvider(customProvider)

	// Create context with value
	type contextKey string
	key := contextKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	_, err := manager.Download(ctx, "test")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if receivedCtx == nil {
		t.Fatal("context was not propagated to provider")
	}

	if receivedCtx.Value(key) != "test-value" {
		t.Error("context value was not preserved")
	}
}

// contextCapturingProvider is a test provider that captures the context it receives
type contextCapturingProvider struct {
	name        string
	matchResult bool
	capturedCtx *context.Context
}

func (c *contextCapturingProvider) Name() string {
	return c.name
}

func (c *contextCapturingProvider) Match(input string) bool {
	return c.matchResult
}

func (c *contextCapturingProvider) Download(ctx context.Context, input string, outputRoot string) (*Result, error) {
	*c.capturedCtx = ctx
	return &Result{
		Provider:  c.name,
		OutputDir: filepath.Join(outputRoot, "test-output"),
		Files:     []string{"file.txt"},
	}, nil
}
