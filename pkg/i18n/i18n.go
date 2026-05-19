// Package i18n is LLMProvider's CONST-046 hardcoded-content
// abstraction (round-338 §11.4 anti-bluff sweep, 2026-05-19).
//
// Per CONST-046, no user-facing text may be a static literal in
// source. Provider-description strings surfaced to end users (in
// model / provider listings) are resolved at runtime through this
// seam against the active locale.
//
// CONST-051(B) decoupling: this package is project-not-aware and
// fully reusable. It defines its own Translator contract; the
// consuming binary injects a real Translator at boot via
// SetTranslator. When no Translator is wired the package-level Tr
// helper falls back to NoopTranslator — a loud message-ID echo,
// never a silent swallow (which would be a §11.4 PASS-bluff at the
// i18n layer).
package i18n

import (
	"context"
	"sync"
)

// Translator is the contract LLMProvider uses for every
// CONST-046-migrated user-facing string. Consumers implement it
// (typically over go-i18n or an equivalent locale bundle) and
// inject it via SetTranslator.
type Translator interface {
	// T resolves messageID against the active locale. templateData
	// supplies named placeholders for interpolation; pass nil when
	// the message has no placeholders.
	T(ctx context.Context, messageID string, templateData map[string]any) (string, error)
}

// NoopTranslator returns the messageID verbatim. SAFETY default
// for unit tests and backward-compat for callers who have not yet
// wired a real Translator. Production paths MUST inject a real
// Translator.
type NoopTranslator struct{}

// T returns id unchanged (loud echo). Never returns an error.
func (NoopTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return id, nil
}

var (
	mu      sync.RWMutex
	current Translator = NoopTranslator{}
)

// SetTranslator installs the process-wide Translator. Passing nil
// restores NoopTranslator so the seam never holds a nil reference.
func SetTranslator(t Translator) {
	mu.Lock()
	defer mu.Unlock()
	if t == nil {
		current = NoopTranslator{}
		return
	}
	current = t
}

// activeTranslator returns the currently-installed Translator under
// a read lock.
func activeTranslator() Translator {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Tr resolves messageID against the active locale via the installed
// Translator. On any Translator error it falls back to the loud
// NoopTranslator echo (messageID verbatim) so a misconfigured
// bundle degrades visibly rather than returning an empty string.
func Tr(ctx context.Context, messageID string, templateData map[string]any) string {
	out, err := activeTranslator().T(ctx, messageID, templateData)
	if err != nil || out == "" {
		echo, _ := NoopTranslator{}.T(ctx, messageID, templateData)
		return echo
	}
	return out
}
