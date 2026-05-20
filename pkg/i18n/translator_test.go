package i18n

import (
	"context"
	"errors"
	"testing"
)

// fakeTranslator is a unit-test-only Translator that returns a fixed
// localized string per message ID — mocks are permitted in unit tests
// per CONST-050(A).
type fakeTranslator struct {
	table  map[string]string
	failID string // when non-empty, T returns an error for this ID
}

func (f fakeTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	if id == f.failID {
		return "", errors.New("forced translator failure")
	}
	if v, ok := f.table[id]; ok {
		return v, nil
	}
	return id, nil
}

// TestNoopTranslator_EchoesID asserts the SAFETY default echoes the
// message ID verbatim — a loud, visible fallback, never a silent
// empty string (which would be a §11.4 PASS-bluff at the i18n layer).
func TestNoopTranslator_EchoesID(t *testing.T) {
	got, err := NoopTranslator{}.T(context.Background(), "llmprovider_validate_api_key_required", nil)
	if err != nil {
		t.Fatalf("NoopTranslator.T returned unexpected error: %v", err)
	}
	if got != "llmprovider_validate_api_key_required" {
		t.Fatalf("NoopTranslator.T = %q, want verbatim message ID", got)
	}
}

// TestTr_DefaultIsNoop asserts Tr falls back to the message ID when no
// real Translator is wired.
func TestTr_DefaultIsNoop(t *testing.T) {
	SetTranslator(nil) // reset to NoopTranslator
	got := Tr(context.Background(), "llmprovider_validate_base_url_required", nil)
	if got != "llmprovider_validate_base_url_required" {
		t.Fatalf("Tr with no wired translator = %q, want message ID echo", got)
	}
}

// TestTr_UsesWiredTranslator is the POSITIVE half of the paired
// mutation: a real Translator IS consulted and its localized output
// reaches the caller. If Tr were a no-op stub ignoring the wired
// translator, this assertion would FAIL — proving the seam is live.
func TestTr_UsesWiredTranslator(t *testing.T) {
	defer SetTranslator(nil)
	SetTranslator(fakeTranslator{table: map[string]string{
		"llmprovider_validate_api_key_required": "Kljuc API je obavezan", // Serbian
	}})
	got := Tr(context.Background(), "llmprovider_validate_api_key_required", nil)
	if got == "llmprovider_validate_api_key_required" {
		t.Fatal("Tr returned the message ID — wired translator was NOT consulted (i18n seam is a bluff)")
	}
	if got != "Kljuc API je obavezan" {
		t.Fatalf("Tr = %q, want localized string from wired translator", got)
	}
}

// TestTr_MutationGuard is the NEGATIVE half of the paired mutation:
// if the migrated literals are reverted to hardcoded English, the
// localized path would never be exercised and a wired translator
// would have no effect. This test plants a translator that maps the
// ID to a non-English string and asserts the English literal does
// NOT appear — i.e. the production code path MUST resolve through Tr,
// not a static string.
func TestTr_MutationGuard(t *testing.T) {
	defer SetTranslator(nil)
	SetTranslator(fakeTranslator{table: map[string]string{
		"llmprovider_validate_model_required": "model je obavezan",
	}})
	got := Tr(context.Background(), "llmprovider_validate_model_required", nil)
	if got == "model is required" {
		t.Fatal("Tr returned the hardcoded English literal — CONST-046 migration regressed")
	}
	if got != "model je obavezan" {
		t.Fatalf("Tr = %q, want wired-translator localized output", got)
	}
}

// TestTr_TranslatorErrorFallsBackToID asserts a translator error
// produces a loud message-ID echo, never an empty string.
func TestTr_TranslatorErrorFallsBackToID(t *testing.T) {
	defer SetTranslator(nil)
	SetTranslator(fakeTranslator{failID: "llmprovider_validate_account_id_required"})
	got := Tr(context.Background(), "llmprovider_validate_account_id_required", nil)
	if got != "llmprovider_validate_account_id_required" {
		t.Fatalf("Tr on translator error = %q, want message-ID echo (loud fallback)", got)
	}
}

// TestCurrentTranslator_ReflectsWiring asserts the accessor tracks
// SetTranslator state.
func TestCurrentTranslator_ReflectsWiring(t *testing.T) {
	defer SetTranslator(nil)
	if _, ok := CurrentTranslator().(NoopTranslator); !ok {
		SetTranslator(nil)
	}
	if _, ok := CurrentTranslator().(NoopTranslator); !ok {
		t.Fatal("default CurrentTranslator should be NoopTranslator")
	}
	ft := fakeTranslator{}
	SetTranslator(ft)
	if _, ok := CurrentTranslator().(fakeTranslator); !ok {
		t.Fatal("CurrentTranslator did not reflect SetTranslator(fakeTranslator{})")
	}
}
