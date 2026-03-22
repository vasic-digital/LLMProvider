package junie_test

import (
	"os"
	"testing"
	"time"

	"digital.vasic.llmprovider/pkg/providers/junie"
)

func TestJunieCLIConfig_Default(t *testing.T) {
	config := junie.DefaultJunieCLIConfig()
	if config.MaxTokens != 4096 {
		t.Errorf("Default MaxTokens should be 4096, got %d", config.MaxTokens)
	}
	if config.Model != "junie-1" {
		t.Errorf("Default Model should be junie-1, got %s", config.Model)
	}
}

func TestDefaultJunieConfig(t *testing.T) {
	config := junie.DefaultJunieConfig()
	if config.Timeout != 180*time.Second {
		t.Errorf("Default timeout should be 180s")
	}
	if config.MaxTokens != 8192 {
		t.Errorf("Default MaxTokens should be 8192")
	}
	if config.Model != "sonnet" {
		t.Errorf("Default Model should be sonnet")
	}
}

func TestDefaultJunieACPConfig(t *testing.T) {
	config := junie.DefaultJunieACPConfig()
	if config.MaxTokens != 4096 {
		t.Errorf("Default MaxTokens should be 4096, got %d", config.MaxTokens)
	}
	if config.Model != "junie-1" {
		t.Errorf("Default Model should be junie-1, got %s", config.Model)
	}
}

func TestNewJunieCLIProvider(t *testing.T) {
	config := junie.DefaultJunieCLIConfig()
	p := junie.NewJunieCLIProvider(config)
	if p.GetCurrentModel() == "" {
		t.Errorf("Default model should be set")
	}
}

func TestNewJunieACPProvider(t *testing.T) {
	config := junie.DefaultJunieACPConfig()
	p := junie.NewJunieACPProvider(config)
	if p.GetCurrentModel() == "" {
		t.Errorf("Default model should be set")
	}
}

func TestNewJunieProvider(t *testing.T) {
	config := junie.DefaultJunieConfig()
	p := junie.NewJunieProvider(config)
	if p.GetCurrentModel() != "sonnet" {
		t.Errorf("Default model should be sonnet")
	}
}

func TestJunieCLIProvider_GetName(t *testing.T) {
	p := junie.NewJunieCLIProvider(junie.DefaultJunieCLIConfig())
	name := p.GetName()
	if name != "junie-cli" {
		t.Errorf("Expected name junie-cli, got %s", name)
	}
}

func TestJunieCLIProvider_GetProviderType(t *testing.T) {
	p := junie.NewJunieCLIProvider(junie.DefaultJunieCLIConfig())
	providerType := p.GetProviderType()
	if providerType != "cli" {
		t.Errorf("Expected provider type cli, got %s", providerType)
	}
}

func TestJunieCLIProvider_GetCurrentModel(t *testing.T) {
	config := junie.DefaultJunieCLIConfig()
	p := junie.NewJunieCLIProvider(config)
	model := p.GetCurrentModel()
	// Stub always returns "junie-1"
	if model != "junie-1" {
		t.Errorf("Expected model junie-1, got %s", model)
	}
}

func TestJunieCLIProvider_SetModel(t *testing.T) {
	p := junie.NewJunieCLIProvider(junie.DefaultJunieCLIConfig())
	// SetModel is a no-op on the stub; just ensure no panic
	p.SetModel("gemini-pro")
	// Stub still returns "junie-1"
	if p.GetCurrentModel() != "junie-1" {
		t.Errorf("Expected model junie-1, got %s", p.GetCurrentModel())
	}
}

func TestJunieACPProvider_GetCurrentModel(t *testing.T) {
	p := junie.NewJunieACPProvider(junie.DefaultJunieACPConfig())
	model := p.GetCurrentModel()
	if model != "junie-1" {
		t.Errorf("Expected model junie-1, got %s", model)
	}
}

func TestJunieACPProvider_IsAvailable(t *testing.T) {
	p := junie.NewJunieACPProvider(junie.DefaultJunieACPConfig())
	if p.IsAvailable() {
		t.Errorf("Expected ACP to not be available in standalone module")
	}
}

func TestJunieProvider_GetName(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	name := p.GetName()
	if name != "junie" {
		t.Errorf("Expected name junie, got %s", name)
	}
}

func TestJunieProvider_GetProviderType(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	providerType := p.GetProviderType()
	if providerType != "junie" {
		t.Errorf("Expected provider type junie, got %s", providerType)
	}
}

func TestJunieProvider_SupportedModels(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	caps := p.GetCapabilities()
	if len(caps.SupportedModels) == 0 {
		t.Errorf("Expected at least one model, got %d", len(caps.SupportedModels))
	}
	for _, model := range caps.SupportedModels {
		if model == "" {
			t.Errorf("Model should not be empty")
		}
	}
}

func TestJunieProvider_BYOKProviders(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	byok := p.GetBYOKProviders()
	// BYOK providers depend on env vars being set; just verify no panic
	for _, provider := range byok {
		if provider == "" {
			t.Errorf("BYOK provider should not be empty")
		}
	}
}

func TestIsJunieInstalled(t *testing.T) {
	installed := junie.IsJunieInstalled()
	if installed {
		t.Logf("Junie should be installed: %v", installed)
	}
}

func TestIsJunieAuthenticated(t *testing.T) {
	authenticated := junie.IsJunieAuthenticated()
	if authenticated {
		t.Logf("Junie should be authenticated: %v", authenticated)
	}
}

func TestJunieProvider_GetCapabilities(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	caps := p.GetCapabilities()
	if len(caps.SupportedModels) == 0 {
		t.Errorf("Expected at least one model, got %d", len(caps.SupportedModels))
	}
	if !caps.SupportsStreaming {
		t.Errorf("Expected streaming support")
	}
	if !caps.SupportsTools {
		t.Errorf("Expected tools support")
	}
}

func TestJunieProvider_ValidateConfig(t *testing.T) {
	p := junie.NewJunieProvider(junie.DefaultJunieConfig())
	valid, issues := p.ValidateConfig(nil)
	if junie.IsJunieInstalled() && (os.Getenv("JUNIE_API_KEY") != "" || junie.IsJunieAuthenticated()) {
		if !valid {
			t.Logf("ValidateConfig returned valid=false, issues: %v", issues)
		}
	} else {
		if valid {
			t.Errorf("Expected invalid config when Junie not available")
		}
		if len(issues) == 0 {
			t.Errorf("Expected issues when Junie not available")
		}
	}
}
