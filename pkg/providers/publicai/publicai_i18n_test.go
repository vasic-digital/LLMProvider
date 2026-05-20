package publicai

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
	case "llmprovider_validate_api_key_required":
		return "API kljuc je obavezan", nil
	case "llmprovider_validate_base_url_required":
		return "Osnovni URL je obavezan", nil
	case "llmprovider_validate_model_required":
		return "Model je obavezan", nil
	}
	return id, nil
}

// TestPublicAIValidateConfig_I18nSeam_Localized is the POSITIVE half of
// the round-367 CONST-046 paired mutation: with a real Translator
// wired, ValidateConfig's user-facing validation errors are localized.
// If the migrated literals were reverted to hardcoded English, the
// wired translator would have no effect and these assertions FAIL.
func TestPublicAIValidateConfig_I18nSeam_Localized(t *testing.T) {
	defer i18n.SetTranslator(nil)
	i18n.SetTranslator(localeTranslator{})

	// Empty API key → "API key is required" path.
	p := &PublicAIProvider{apiKey: "", baseURL: "https://api.publicai.co/v1", model: "m"}
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

	// Empty base URL path.
	p2 := &PublicAIProvider{apiKey: "k", baseURL: "", model: "m"}
	ok2, errs2 := p2.ValidateConfig(nil)
	if ok2 || len(errs2) == 0 {
		t.Fatal("ValidateConfig with empty base URL should be invalid")
	}
	if strings.Contains(errs2[0], "base URL is required") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 migration regressed", errs2[0])
	}
	if !strings.Contains(errs2[0], "URL") || !strings.Contains(errs2[0], "Osnovni") {
		t.Fatalf("ValidateConfig error %q did not flow through the i18n seam", errs2[0])
	}

	// Empty model path.
	p3 := &PublicAIProvider{apiKey: "k", baseURL: "https://api.publicai.co/v1", model: ""}
	ok3, errs3 := p3.ValidateConfig(nil)
	if ok3 || len(errs3) == 0 {
		t.Fatal("ValidateConfig with empty model should be invalid")
	}
	if strings.Contains(errs3[0], "model is required") {
		t.Fatalf("ValidateConfig emitted hardcoded English literal %q — CONST-046 migration regressed", errs3[0])
	}
	if !strings.Contains(errs3[0], "Model je") {
		t.Fatalf("ValidateConfig error %q did not flow through the i18n seam", errs3[0])
	}
}

// TestPublicAIValidateConfig_I18nSeam_NoopFallback is the NEGATIVE
// half: with no Translator wired, the NoopTranslator echoes the
// message ID verbatim — a loud, visible fallback, never a silent
// empty string.
func TestPublicAIValidateConfig_I18nSeam_NoopFallback(t *testing.T) {
	i18n.SetTranslator(nil) // reset to NoopTranslator
	p := &PublicAIProvider{apiKey: "", baseURL: "", model: ""}
	ok, errs := p.ValidateConfig(nil)
	if ok || len(errs) != 3 {
		t.Fatalf("ValidateConfig with all fields empty should yield 3 errors, got %v", errs)
	}
	want := []string{
		"llmprovider_validate_api_key_required",
		"llmprovider_validate_base_url_required",
		"llmprovider_validate_model_required",
	}
	for i, w := range want {
		if errs[i] != w {
			t.Fatalf("NoopTranslator fallback errs[%d] = %q, want verbatim message ID %q", i, errs[i], w)
		}
	}
}
