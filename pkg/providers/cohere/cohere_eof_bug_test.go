package cohere

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
// bug: the SSE read loop does `line, err := reader.ReadBytes('\n'); if err != nil
// { if err == io.EOF { break } }` which DROPS the final partial line that
// ReadBytes returns ALONGSIDE io.EOF when the last data chunk has no trailing \n.
//
// The handler writes a normal content-delta WITH a trailing \n, then the final
// content-delta carrying "LAST" WITHOUT a trailing newline, then closes (EOF).
// A correct implementation aggregates "Hello LAST"; the buggy one drops "LAST".
func TestCompleteStream_EOFLastChunkNoNewline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// First chunk WITH trailing newline.
		_, _ = w.Write([]byte(`data: {"type":"content-delta","delta":{"message":{"content":{"type":"text","text":"Hello "}}}}` + "\n"))
		// Final chunk WITHOUT trailing newline -> returned alongside io.EOF.
		_, _ = w.Write([]byte(`data: {"type":"content-delta","delta":{"message":{"content":{"type":"text","text":"LAST"}}}}`))
		// Handler returns -> body closes -> reader hits io.EOF on the last line.
	}))
	defer server.Close()

	provider := NewProvider("test-api-key", server.URL, "command-r-plus")

	ch, err := provider.CompleteStream(context.Background(), &models.LLMRequest{ID: "req-eof"})
	require.NoError(t, err)

	var agg strings.Builder
	for resp := range ch {
		agg.WriteString(resp.Content)
	}

	require.Contains(t, agg.String(), "LAST",
		"final content-delta chunk without trailing newline was dropped on io.EOF")
}
