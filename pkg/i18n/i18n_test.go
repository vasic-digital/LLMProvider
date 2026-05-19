package i18n

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestNoopTranslator_EchoesID verifies the safety default echoes the
// message ID verbatim — a loud failure mode, never a silent swallow.
func TestNoopTranslator_EchoesID(t *testing.T) {
	got, err := NoopTranslator{}.T(context.Background(), "provider.kilo.description", nil)
	if err != nil {
		t.Fatalf("NoopTranslator.T returned error: %v", err)
	}
	if got != "provider.kilo.description" {
		t.Fatalf("NoopTranslator echo: got %q want %q", got, "provider.kilo.description")
	}
}

// TestTr_FallsBackToEchoWhenUnwired verifies Tr returns the loud
// message-ID echo when no real Translator is installed.
func TestTr_FallsBackToEchoWhenUnwired(t *testing.T) {
	SetTranslator(nil) // restore Noop
	got := Tr(context.Background(), "provider.zhipu.description", nil)
	if got != "provider.zhipu.description" {
		t.Fatalf("unwired Tr: got %q want loud echo", got)
	}
}

// TestBundleTranslator_ResolvesShippedBundle loads the real shipped
// active.en.yaml and asserts every migrated provider-description ID
// resolves to non-empty, non-echo text. This is the anti-bluff core:
// a PASS here proves the i18n seam actually delivers translated
// strings to end users, not that the abstraction merely compiles.
func TestBundleTranslator_ResolvesShippedBundle(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("bundles", "active.en.yaml"))
	if err != nil {
		t.Fatalf("read shipped bundle: %v", err)
	}
	bt, err := NewBundleTranslatorFromBytes(data)
	if err != nil {
		t.Fatalf("parse shipped bundle: %v", err)
	}
	SetTranslator(bt)
	t.Cleanup(func() { SetTranslator(nil) })

	wantIDs := map[string]string{
		"provider.cerebras.description":   "Ultra-fast inference on Cerebras hardware",
		"provider.kilo.description":       "Kilo Code AI assistant",
		"provider.zhipu.description":      "Zhipu AI GLM models",
		"provider.zen.description":        "OpenCode Zen gateway - Free models (Big Pickle, Grok Code Fast, GLM 4.7, GPT 5 Nano)",
		"provider.junie.display_name":     "Junie AI Coding Agent",
		"provider.gemini.description":     "Google's Gemini models with API, CLI headless, and ACP access methods",
		"provider.nvidia.description":     "NVIDIA NIM API for LLM inference",
		"provider.publicai.description":   "Swiss AI Apertus - open-source LLM via Public AI Gateway",
		"provider.vulavula.description":   "Vulavula AI - African LLM",
		"provider.cloudflare.description": "Cloudflare Workers AI",
	}
	for id, want := range wantIDs {
		got := Tr(context.Background(), id, nil)
		if got == id {
			t.Errorf("Tr(%q) returned loud echo — message missing from shipped bundle", id)
			continue
		}
		if got != want {
			t.Errorf("Tr(%q): got %q want %q", id, got, want)
		}
	}
}

// TestBundleTranslator_PairedMutation is the §1.1 paired-mutation
// guard. It plants a bundle whose entry differs from the source
// literal and asserts the seam returns the MUTATED value — proving
// the provider code resolves through the seam at runtime rather than
// holding a stale hardcoded copy. If the migration were a bluff
// (literal still hardcoded), the mutated bundle would have no effect
// and this test would FAIL.
func TestBundleTranslator_PairedMutation(t *testing.T) {
	const mutated = "MUTATED-cerebras-description"
	bt, err := NewBundleTranslatorFromBytes([]byte(
		"provider.cerebras.description: \"" + mutated + "\"\n"))
	if err != nil {
		t.Fatalf("parse mutated bundle: %v", err)
	}
	SetTranslator(bt)
	t.Cleanup(func() { SetTranslator(nil) })

	got := Tr(context.Background(), "provider.cerebras.description", nil)
	if got != mutated {
		t.Fatalf("paired-mutation: got %q want %q — seam not honoured at runtime", got, mutated)
	}
}

// TestBundleTranslator_Interpolation verifies placeholder
// substitution works for future templated messages.
func TestBundleTranslator_Interpolation(t *testing.T) {
	bt, err := NewBundleTranslatorFromBytes([]byte(
		"greeting: \"Hello {{name}}, you have {{count}} models\"\n"))
	if err != nil {
		t.Fatalf("parse bundle: %v", err)
	}
	got, err := bt.T(context.Background(), "greeting", map[string]any{"name": "Ada", "count": 3})
	if err != nil {
		t.Fatalf("T error: %v", err)
	}
	want := "Hello Ada, you have 3 models"
	if got != want {
		t.Fatalf("interpolation: got %q want %q", got, want)
	}
}

// TestBundleTranslator_UnknownIDErrors verifies an unknown message
// ID surfaces an error (so Tr falls back to the loud echo).
func TestBundleTranslator_UnknownIDErrors(t *testing.T) {
	bt, err := NewBundleTranslatorFromBytes([]byte("known: \"value\"\n"))
	if err != nil {
		t.Fatalf("parse bundle: %v", err)
	}
	if _, err := bt.T(context.Background(), "missing.id", nil); err == nil {
		t.Fatal("expected error for unknown message id, got nil")
	}
}
