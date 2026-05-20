package kilo

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmprovider/pkg/i18n"
)

// vcLocaleTranslator is a unit-test-only Translator returning a fixed
// non-English string for the ValidateConfig message ID. Mocks are
// permitted in unit tests per CONST-050(A).
type vcLocaleTranslator struct{}

func (vcLocaleTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	if id == "llmprovider_validate_api_key_required" {
		return "API kljuc je obavezan", nil
	}
	return id, nil
}

// TestKiloValidateConfig_I18nSeam_Localized is the POSITIVE half of the
// round-367 CONST-046 paired mutation for the single-literal
// ValidateConfig migration class: with a real Translator wired, the
// API-key validation error is localized. Reverting the migrated literal
// to hardcoded English makes the wired translator inert and this FAILS.
func TestKiloValidateConfig_I18nSeam_Localized(t *testing.T) {
	defer i18n.SetTranslator(nil)
	i18n.SetTranslator(vcLocaleTranslator{})

	p := NewKiloProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) == 0 {
		t.Fatal("ValidateConfig with empty API key should be invalid")
	}
	if strings.Contains(errs[0], "API key is required") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 migration regressed", errs[0])
	}
	if !strings.Contains(errs[0], "kljuc") {
		t.Fatalf("ValidateConfig error %q did not flow through the i18n seam", errs[0])
	}
}

// TestKiloValidateConfig_I18nSeam_NoopFallback is the NEGATIVE half:
// with no Translator wired, the NoopTranslator echoes the message ID
// verbatim — a loud, visible fallback, never a silent empty string.
func TestKiloValidateConfig_I18nSeam_NoopFallback(t *testing.T) {
	i18n.SetTranslator(nil) // reset to NoopTranslator
	p := NewKiloProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) == 0 {
		t.Fatal("ValidateConfig with empty API key should be invalid")
	}
	if errs[0] != "llmprovider_validate_api_key_required" {
		t.Fatalf("NoopTranslator fallback = %q, want verbatim message ID echo", errs[0])
	}
}
