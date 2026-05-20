package venice

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmprovider/pkg/i18n"
)

// vcLocaleTranslator is a unit-test-only Translator returning a fixed
// non-English string for the ValidateConfig message ID the round-441
// CONST-046 migration routes through the i18n seam. Mocks are permitted
// in unit tests per CONST-050(A).
type vcLocaleTranslator struct{}

func (vcLocaleTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	if id == "llmprovider_validate_api_key_required_venice" {
		return "API ključ je obavezan (postavite VENICE_API_KEY)", nil
	}
	return id, nil
}

// TestVeniceValidateConfig_I18nSeam_Localized is the POSITIVE half of
// the round-441 CONST-046 paired mutation: with a real Translator wired
// the ValidateConfig api-key error is localized. Reverting the migrated
// literal to hardcoded English makes the wired translator inert for that
// case and this FAILS.
func TestVeniceValidateConfig_I18nSeam_Localized(t *testing.T) {
	defer i18n.SetTranslator(nil)
	i18n.SetTranslator(vcLocaleTranslator{})

	p := NewProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) != 1 {
		t.Fatalf("ValidateConfig with empty api key should yield 1 error, got %d", len(errs))
	}
	if strings.Contains(errs[0], "API key is required (set VENICE_API_KEY)") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 round-441 migration regressed", errs[0])
	}
	if !strings.Contains(errs[0], "API ključ je obavezan") {
		t.Fatalf("ValidateConfig error %q missing localized fragment — i18n seam not exercised", errs[0])
	}
}

// TestVeniceValidateConfig_I18nSeam_NoopFallback is the NEGATIVE half:
// with no Translator wired the NoopTranslator echoes the message ID
// verbatim — a loud, visible fallback, never a silent empty string.
func TestVeniceValidateConfig_I18nSeam_NoopFallback(t *testing.T) {
	i18n.SetTranslator(nil) // reset to NoopTranslator
	p := NewProvider("", "", "")
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) != 1 {
		t.Fatalf("ValidateConfig with empty api key should yield 1 error, got %d", len(errs))
	}
	if errs[0] != "llmprovider_validate_api_key_required_venice" {
		t.Fatalf("NoopTranslator fallback = %q, want a verbatim message ID echo", errs[0])
	}
}
