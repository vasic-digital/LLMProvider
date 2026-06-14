package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.llmprovider/pkg/models"
	"github.com/stretchr/testify/require"
)

// TestClaudeProvider_CompleteStream_NonRetryableErrorIsNotFakeSuccess proves
// the bug: CompleteStream never checks resp.StatusCode. On a NON-retryable
// error (HTTP 400) makeAPICall returns (resp, nil), the goroutine reads the
// JSON error body as SSE (no "data: " lines), hits EOF, and emits a final
// response with FinishReason="stop" and empty Content — a success-shaped
// response on an error body. Every sibling provider's CompleteStream guards
// on StatusCode and returns an error here.
func TestClaudeProvider_CompleteStream_NonRetryableErrorIsNotFakeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400 — non-retryable
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens is required"}}`))
	}))
	defer server.Close()

	provider := NewClaudeProvider("test-key", server.URL, "claude-3-sonnet")
	req := &models.LLMRequest{
		ID:       "red-stream-400",
		Messages: []models.Message{{Role: "user", Content: "Hello"}},
	}

	ch, err := provider.CompleteStream(context.Background(), req)

	// Acceptable behaviors: either an immediate error return, OR an error
	// response delivered through the channel. FORBIDDEN: a success-shaped
	// final response (FinishReason "stop") that masquerades a 400 as success.
	if err != nil {
		return // correct: surfaced the 400 up front
	}
	require.NotNil(t, ch)

	sawError := false
	sawFakeSuccess := false
	for resp := range ch {
		if resp.FinishReason == "error" {
			sawError = true
		}
		if resp.FinishReason == "stop" {
			sawFakeSuccess = true
		}
	}

	require.False(t, sawFakeSuccess,
		"CompleteStream emitted a FinishReason=stop final response on an HTTP 400 error body — a success-on-error bluff")
	require.True(t, sawError,
		"CompleteStream consumed an HTTP 400 error body without surfacing any error")
}
