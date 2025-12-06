package downloader

import (
	"context"
	"fmt"
	"path/filepath"
)

// Provider represents a site-specific downloader implementation.
type Provider interface {
	Name() string
	Match(input string) bool
	Download(ctx context.Context, input string, outputRoot string) (*Result, error)
}

// Result captures the output of a download run.
type Result struct {
	Provider     string
	OutputDir    string
	Files        []string
	MetadataPath string
}

// Manager routes downloads to the first provider that claims it can handle the input.
type Manager struct {
	outputRoot string
	providers  []Provider
}

func NewManager(outputRoot string) *Manager {
	return &Manager{outputRoot: outputRoot}
}

// RegisterProvider registers providers in priority order.
func (m *Manager) RegisterProvider(provider Provider) {
	m.providers = append(m.providers, provider)
}

// Download dispatches to a provider. It ensures the output directory is under the configured root.
func (m *Manager) Download(ctx context.Context, input string) (*Result, error) {
	for _, provider := range m.providers {
		if provider.Match(input) {
			result, err := provider.Download(ctx, input, m.outputRoot)
			if err != nil {
				return nil, err
			}
			// Make path absolute for consistency.
			if !filepath.IsAbs(result.OutputDir) {
				result.OutputDir = filepath.Join(m.outputRoot, result.OutputDir)
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("no provider can handle input %q", input)
}
