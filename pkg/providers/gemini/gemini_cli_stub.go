package gemini

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.llmprovider/pkg/models"
)

// GeminiCLIProvider is a stub for the CLI provider (not available in standalone module)
type GeminiCLIProvider struct{}

// GeminiCLIConfig is a stub for CLI configuration
type GeminiCLIConfig struct {
	Model           string
	MaxTokens       int
	MaxOutputTokens int
	Timeout         time.Duration
	APIKey          string
}

// NewGeminiCLIProvider returns a stub (CLI not available in standalone module)
func NewGeminiCLIProvider(_ GeminiCLIConfig) *GeminiCLIProvider {
	return &GeminiCLIProvider{}
}

// IsCLIAvailable returns false (CLI not available in standalone module)
func (p *GeminiCLIProvider) IsCLIAvailable() bool { return false }

// GetCLIError returns an error (CLI not available)
func (p *GeminiCLIProvider) GetCLIError() error {
	return fmt.Errorf("Gemini CLI not available in standalone module")
}

// Complete is not available in standalone module
func (p *GeminiCLIProvider) Complete(_ context.Context, _ *models.LLMRequest) (*models.LLMResponse, error) {
	return nil, fmt.Errorf("Gemini CLI not available in standalone module")
}

// CompleteStream is not available in standalone module
func (p *GeminiCLIProvider) CompleteStream(_ context.Context, _ *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	return nil, fmt.Errorf("Gemini CLI not available in standalone module")
}

// IsGeminiCLIInstalled returns false (CLI not available in standalone module)
func IsGeminiCLIInstalled() bool { return false }

// IsGeminiCLIAuthenticated returns false (CLI not available in standalone module)
func IsGeminiCLIAuthenticated() bool { return false }

// CanUseGeminiCLI returns false (CLI not available in standalone module)
func CanUseGeminiCLI() bool { return false }

// GeminiACPProvider is a stub for the ACP provider
type GeminiACPProvider struct{}

// GeminiACPConfig is a stub for ACP configuration
type GeminiACPConfig struct {
	Model           string
	MaxTokens       int
	MaxOutputTokens int
	Timeout         time.Duration
	APIKey          string
}

// NewGeminiACPProvider returns a stub
func NewGeminiACPProvider(_ GeminiACPConfig) *GeminiACPProvider {
	return &GeminiACPProvider{}
}

// IsAvailable returns false (ACP not available in standalone module)
func (p *GeminiACPProvider) IsAvailable() bool { return false }

// CanUseGeminiACP returns false (ACP not available in standalone module)
func CanUseGeminiACP() bool { return false }

// Complete is not available for ACP
func (p *GeminiACPProvider) Complete(_ context.Context, _ *models.LLMRequest) (*models.LLMResponse, error) {
	return nil, fmt.Errorf("Gemini ACP not available in standalone module")
}

// CompleteStream is not available for ACP
func (p *GeminiACPProvider) CompleteStream(_ context.Context, _ *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	return nil, fmt.Errorf("Gemini ACP not available in standalone module")
}

// SetModel is a no-op stub
func (p *GeminiCLIProvider) SetModel(_ string) {
	_ = p
}

// SetModel is a no-op stub
func (p *GeminiACPProvider) SetModel(_ string) {
	_ = p
}

// DiscoverModels returns a minimal fallback list when the CLI isn't
// available. Callers treat the return value as the set of callable
// model identifiers; returning nil causes downstream "no models
// available" errors that look like a bug even when the stub is the
// expected configuration. The single fallback identifier matches
// what the real CLI advertises by default.
func (p *GeminiCLIProvider) DiscoverModels() []string {
	return []string{"gemini-2.5-flash"}
}
