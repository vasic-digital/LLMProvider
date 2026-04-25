package junie

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.llmprovider/pkg/models"
)

// JunieCLIProvider is a stub for the CLI provider (not available in standalone module)
type JunieCLIProvider struct {
	model string
}

// JunieCLIConfig is a stub for CLI configuration
type JunieCLIConfig struct {
	Model           string
	MaxTokens       int
	MaxOutputTokens int
	Timeout         time.Duration
	APIKey          string
}

// NewJunieCLIProvider returns a stub that stores the configured model.
// When the config model is empty the stub falls back to "junie-1" so that
// GetCurrentModel always returns a non-empty value.
func NewJunieCLIProvider(cfg JunieCLIConfig) *JunieCLIProvider {
	model := cfg.Model
	if model == "" {
		model = "junie-1"
	}
	return &JunieCLIProvider{model: model}
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
	// CWD is the working directory passed to the Junie ACP process.
	// Defaults to "." (current directory).
	CWD string
}

// NewJunieACPProvider returns a stub
func NewJunieACPProvider(_ JunieACPConfig) *JunieACPProvider {
	return &JunieACPProvider{}
}

// IsAvailable returns false
func (p *JunieACPProvider) IsAvailable() bool { return false }

var knownJunieModels = []string{"junie-1"}

// byokModels is a flat slice of additional model names available via BYOK.
// Used internally by getAllJunieModels.
var byokModels = []string{}

// byokProviders maps BYOK provider names to their supported model lists.
// Returned by GetBYOKModels.
var byokProviders = map[string][]string{
	"anthropic": {"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
	"openai":    {"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"},
	"google":    {"gemini-pro", "gemini-flash"},
	"grok":      {"grok-1"},
}

// GetKnownJunieModels returns the list of known Junie model identifiers.
func GetKnownJunieModels() []string { return knownJunieModels }

// GetBYOKModels returns the map of BYOK provider names to their model lists.
func GetBYOKModels() map[string][]string { return byokProviders }

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

// SetModel updates the model on the CLI stub.
func (p *JunieCLIProvider) SetModel(model string) { p.model = model }

// SetModel is a no-op for the ACP stub.
func (p *JunieACPProvider) SetModel(_ string) {}

// CanUseJunieCLI returns false (not available in standalone module)
func CanUseJunieCLI() bool { return false }

// CanUseJunieACP returns false (not available in standalone module)
func CanUseJunieACP() bool { return false }

// DefaultJunieCLIConfig returns a default CLI config stub.
// Model is intentionally empty — callers set it explicitly.
func DefaultJunieCLIConfig() JunieCLIConfig {
	return JunieCLIConfig{
		MaxTokens:       4096,
		MaxOutputTokens: 8192,
		Timeout:         180 * time.Second,
	}
}

// DefaultJunieACPConfig returns a default ACP config stub.
func DefaultJunieACPConfig() JunieACPConfig {
	return JunieACPConfig{
		MaxTokens: 8192,
		Timeout:   180 * time.Second,
		CWD:       ".",
	}
}

// GetCurrentModel returns the model set on the CLI stub.
func (p *JunieCLIProvider) GetCurrentModel() string { return p.model }

// GetCurrentModel returns the current model (ACP stub).
func (p *JunieACPProvider) GetCurrentModel() string { return "junie-1" }

// GetName returns the provider name.
func (p *JunieACPProvider) GetName() string { return "junie-acp" }

// GetProviderType returns the provider type.
func (p *JunieACPProvider) GetProviderType() string { return "junie" }

// GetName returns the provider name.
func (p *JunieCLIProvider) GetName() string { return "junie-cli" }

// GetProviderType returns the provider type.
func (p *JunieCLIProvider) GetProviderType() string { return "junie" }
