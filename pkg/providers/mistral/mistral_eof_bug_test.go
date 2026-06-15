package mistral

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"digital.vasic.llmprovider/pkg/models"
	"github.com/stretchr/testify/require"
)

// TestCompleteStream_EOFLastChunkNoNewline reproduces the EOF-last-chunk data-loss
// bug in the Mistral SSE read loop: `line, err := reader.ReadBytes('\n'); if err
// != nil { if err == io.EOF { break } }` DROPS the final partial line that
// ReadBytes returns alongside io.EOF when the last `data: {...}` chunk has no
// trailing newline.
func TestCompleteStream_EOFLastChunkNoNewline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// First chunk WITH trailing newline.
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"Hello "}}]}` + "\n"))
		// Final chunk WITHOUT trailing newline -> returned alongside io.EOF.
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"LAST"}}]}`))
	}))
	defer server.Close()

	provider := NewMistralProvider("test-key", server.URL, "mistral-large-latest")

	ch, err := provider.CompleteStream(context.Background(), &models.LLMRequest{ID: "req-eof"})
	require.NoError(t, err)

	var agg strings.Builder
	for resp := range ch {
		agg.WriteString(resp.Content)
	}

	require.Contains(t, agg.String(), "LAST",
		"final delta chunk without trailing newline was dropped on io.EOF")
}
