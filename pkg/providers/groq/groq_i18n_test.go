package groq

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmprovider/pkg/i18n"
)

// localeTranslator is a unit-test-only Translator returning a fixed
// non-English string per message ID. Mocks are permitted in unit
// tests per CONST-050(A).
type localeTranslator struct{}

func (localeTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	switch id {
	case "llmprovider_validate_api_key_required_groq":
		return "Groq API kljuc je obavezan (pocinje sa 'gsk_')", nil
	case "llmprovider_validate_api_key_format_groq":
		return "Neispravan format API kljuca", nil
	}
	return id, nil
}

// TestValidateConfig_I18nSeam_Localized is the POSITIVE half of the
// round-337 CONST-046 paired mutation: with a real Translator wired,
// ValidateConfig's user-facing validation errors are localized. If the
// migrated literals were reverted to hardcoded English, the wired
// translator would have no effect and these assertions would FAIL.
func TestValidateConfig_I18nSeam_Localized(t *testing.T) {
	defer i18n.SetTranslator(nil)
	i18n.SetTranslator(localeTranslator{})

	// Empty API key → "API key is required" path.
	p := NewProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok {
		t.Fatal("ValidateConfig with empty API key should be invalid")
	}
	if len(errs) == 0 {
		t.Fatal("ValidateConfig returned no errors for empty API key")
	}
	if strings.Contains(errs[0], "API key is required (Groq API key starts with 'gsk_')") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 migration regressed", errs[0])
	}
	if !strings.Contains(errs[0], "kljuc") {
		t.Fatalf("ValidateConfig error %q did not flow through the i18n seam", errs[0])
	}

	// Malformed API key → "Invalid API key format" path.
	p2 := NewProvider("not-a-gsk-key", "", "")
	ok2, errs2 := p2.ValidateConfig(nil)
	if ok2 || len(errs2) == 0 {
		t.Fatal("ValidateConfig with malformed key should be invalid")
	}
	if strings.Contains(errs2[0], "Invalid API key format (should start with 'gsk_')") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 migration regressed", errs2[0])
	}
	if !strings.Contains(errs2[0], "Neispravan") {
		t.Fatalf("ValidateConfig error %q did not flow through the i18n seam", errs2[0])
	}
}

// TestValidateConfig_I18nSeam_NoopFallback is the NEGATIVE half: with
// no Translator wired, the NoopTranslator echoes the message ID
// verbatim — a loud, visible fallback, never a silent empty string.
func TestValidateConfig_I18nSeam_NoopFallback(t *testing.T) {
	i18n.SetTranslator(nil) // reset to NoopTranslator
	p := NewProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) == 0 {
		t.Fatal("ValidateConfig with empty API key should be invalid")
	}
	if errs[0] != "llmprovider_validate_api_key_required_groq" {
		t.Fatalf("NoopTranslator fallback = %q, want verbatim message ID echo", errs[0])
	}
}
