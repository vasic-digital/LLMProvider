package junie

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.llmprovider/pkg/models"
)

// JunieCLIProvider is a stub for the CLI provider (not available in standalone module)
type JunieCLIProvider struct{}

// JunieCLIConfig is a stub for CLI configuration
type JunieCLIConfig struct {
	Model           string
	MaxTokens       int
	MaxOutputTokens int
	Timeout         time.Duration
	APIKey          string
}

// NewJunieCLIProvider returns a stub
func NewJunieCLIProvider(_ JunieCLIConfig) *JunieCLIProvider {
	return &JunieCLIProvider{}
}

// IsCLIAvailable returns false (CLI not available in standalone module)
func (p *JunieCLIProvider) IsCLIAvailable() bool { return false }

// GetCLIError returns an error
func (p *JunieCLIProvider) GetCLIError() error {
	return fmt.Errorf("Junie CLI not available in standalone module")
}

// Complete is not available
func (p *JunieCLIProvider) Complete(_ context.Context, _ *models.LLMRequest) (*models.LLMResponse, error) {
	return nil, fmt.Errorf("Junie CLI not available in standalone module")
}

// CompleteStream is not available
func (p *JunieCLIProvider) CompleteStream(_ context.Context, _ *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	return nil, fmt.Errorf("Junie CLI not available in standalone module")
}

// IsJunieAuthenticated returns false (CLI not available in standalone module)
func IsJunieAuthenticated() bool { return false }

// IsJunieInstalled returns false (CLI not available in standalone module)
func IsJunieInstalled() bool { return false }

// JunieACPProvider is a stub for the ACP provider
type JunieACPProvider struct{}

// JunieACPConfig is a stub
type JunieACPConfig struct {
	Model           string
	MaxTokens       int
	MaxOutputTokens int
	Timeout         time.Duration
	APIKey          string
}

// NewJunieACPProvider returns a stub
func NewJunieACPProvider(_ JunieACPConfig) *JunieACPProvider {
	return &JunieACPProvider{}
}

// IsAvailable returns false
func (p *JunieACPProvider) IsAvailable() bool { return false }

var knownJunieModels = []string{"junie-1"}
var byokModels = []string{}

// Complete is not available for ACP
func (p *JunieACPProvider) Complete(_ context.Context, _ *models.LLMRequest) (*models.LLMResponse, error) {
	return nil, fmt.Errorf("Junie ACP not available in standalone module")
}

// CompleteStream is not available for ACP
func (p *JunieACPProvider) CompleteStream(_ context.Context, _ *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	return nil, fmt.Errorf("Junie ACP not available in standalone module")
}

// HealthCheck for CLI stub
func (p *JunieCLIProvider) HealthCheck() error {
	return fmt.Errorf("Junie CLI not available in standalone module")
}

// HealthCheck for ACP stub
func (p *JunieACPProvider) HealthCheck() error {
	return fmt.Errorf("Junie ACP not available in standalone module")
}

// SetModel for CLI stub
func (p *JunieCLIProvider) SetModel(_ string) {}

// SetModel for ACP stub
func (p *JunieACPProvider) SetModel(_ string) {}

// CanUseJunieCLI returns false (not available in standalone module)
func CanUseJunieCLI() bool { return false }

// CanUseJunieACP returns false (not available in standalone module)
func CanUseJunieACP() bool { return false }

// DefaultJunieCLIConfig returns a default CLI config stub
func DefaultJunieCLIConfig() JunieCLIConfig {
	return JunieCLIConfig{Model: "junie-1", MaxTokens: 4096}
}

// DefaultJunieACPConfig returns a default ACP config stub
func DefaultJunieACPConfig() JunieACPConfig {
	return JunieACPConfig{Model: "junie-1", MaxTokens: 4096}
}

// GetCurrentModel returns the current model (stub)
func (p *JunieCLIProvider) GetCurrentModel() string { return "junie-1" }

// GetCurrentModel returns the current model (stub)
func (p *JunieACPProvider) GetCurrentModel() string { return "junie-1" }

// GetName returns the provider name (stub)
func (p *JunieCLIProvider) GetName() string { return "junie-cli" }

// GetProviderType returns the provider type (stub)
func (p *JunieCLIProvider) GetProviderType() string { return "cli" }
