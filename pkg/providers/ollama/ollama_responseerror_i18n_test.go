package ollama

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmprovider/pkg/i18n"
	"digital.vasic.llmprovider/pkg/models"
)

// reLocaleTranslator is a unit-test-only Translator returning a fixed
// non-English string for the streaming-response error message ID the
// round-441 CONST-046 migration routes through the i18n seam. The
// {{error}} placeholder carries the underlying error text. Mocks are
// permitted in unit tests per CONST-050(A).
type reLocaleTranslator struct{}

func (reLocaleTranslator) T(_ context.Context, id string, data map[string]any) (string, error) {
	if id == "llmprovider_ollama_response_error" {
		detail := ""
		if data != nil {
			if v, ok := data["error"].(string); ok {
				detail = v
			}
		}
		return "Greška: " + detail, nil
	}
	return id, nil
}

// streamErr drains a CompleteStream channel and returns the first
// non-empty Content carrying the "error" FinishReason — the user-facing
// failure message.
func streamErr(t *testing.T, p *OllamaProvider) string {
	t.Helper()
	ch, err := p.CompleteStream(context.Background(), &models.LLMRequest{
		ID:     "round-441-test",
		Prompt: "hello",
	})
	if err != nil {
		t.Fatalf("CompleteStream returned an immediate error: %v", err)
	}
	for resp := range ch {
		if resp != nil && resp.FinishReason == "error" && resp.Content != "" {
			return resp.Content
		}
	}
	t.Fatalf("CompleteStream produced no error response — expected an unreachable-host failure")
	return ""
}

// TestOllamaResponseError_I18nSeam_Localized is the POSITIVE half of the
// round-441 CONST-046 paired mutation: pointed at an unreachable host,
// CompleteStream surfaces the HTTP failure as a user-facing Content
// string. With a real Translator wired that string is localized.
// Reverting the migrated literal to fmt.Sprintf("Error: %v", err) makes
// the wired translator inert for that case and this FAILS.
func TestOllamaResponseError_I18nSeam_Localized(t *testing.T) {
	defer i18n.SetTranslator(nil)
	i18n.SetTranslator(reLocaleTranslator{})

	// 127.0.0.1:1 is reliably unreachable — httpClient.Do errors,
	// driving the migrated error-response Content branch.
	p := NewOllamaProvider("http://127.0.0.1:1", "llama2")
	got := streamErr(t, p)

	if strings.HasPrefix(got, "Error: ") {
		t.Fatalf("CompleteStream emitted hardcoded English literal %q — CONST-046 round-441 migration regressed", got)
	}
	if !strings.HasPrefix(got, "Greška: ") {
		t.Fatalf("CompleteStream error Content %q missing localized prefix — i18n seam not exercised", got)
	}
}

// TestOllamaResponseError_I18nSeam_NoopFallback is the NEGATIVE half:
// with no Translator wired the NoopTranslator echoes the message ID
// verbatim — a loud, visible fallback, never a silent empty string.
func TestOllamaResponseError_I18nSeam_NoopFallback(t *testing.T) {
	i18n.SetTranslator(nil) // reset to NoopTranslator
	p := NewOllamaProvider("http://127.0.0.1:1", "llama2")
	got := streamErr(t, p)
	if got != "llmprovider_ollama_response_error" {
		t.Fatalf("NoopTranslator fallback = %q, want a verbatim message ID echo", got)
	}
}
