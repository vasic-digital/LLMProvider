package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.llmprovider/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOllamaProvider_CompleteStream_HTTPErrorStatus reproduces the
// success-on-HTTP-error-body bug class: the streaming path does NOT guard on
// response.StatusCode before decoding the body. When Ollama returns a non-2xx
// HTTP status with a VALID JSON error body (e.g. 404/500 "model not found"),
// that body decodes cleanly into an OllamaResponse{Response:"", Done:false},
// the loop runs once, hits EOF, and the channel closes WITHOUT ever emitting a
// FinishReason=="error" response. The end user receives no content and no error
// signal — a silent failure. The non-streaming Complete path correctly returns
// an error for the same status (TestOllamaProvider_Complete server-error case),
// so the stream path diverges from the contract.
func TestOllamaProvider_CompleteStream_HTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Real Ollama returns a JSON error body with a non-2xx status when
		// the model is missing. This JSON is valid and decodes into an empty
		// OllamaResponse, so the decode-error path never fires.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model 'llama2' not found, try pulling it first"}`))
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "llama2")
	req := &models.LLMRequest{ID: "test-http-error", Prompt: "test prompt"}

	ch, err := provider.CompleteStream(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, ch)

	var responses []*models.LLMResponse
	for resp := range ch {
		responses = append(responses, resp)
	}

	// The contract: an HTTP error MUST surface as an error response to the
	// consumer, never a silent empty stream.
	require.NotEmpty(t, responses, "stream must emit at least one response; "+
		"an HTTP error must not produce a silent empty stream")
	sawError := false
	for _, r := range responses {
		if r.FinishReason == "error" {
			sawError = true
		}
	}
	assert.True(t, sawError,
		"HTTP %d error body must surface as a FinishReason=error response, "+
			"got responses=%+v", http.StatusNotFound, responses)

	// Anti-bluff: the stream MUST NOT report success (FinishReason "stop") on
	// an HTTP error.
	for _, r := range responses {
		assert.NotEqual(t, "stop", r.FinishReason,
			"HTTP error must not be reported as a successful 'stop' stream")
	}
}
