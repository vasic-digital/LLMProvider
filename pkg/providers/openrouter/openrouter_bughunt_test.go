package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digital.vasic.llmprovider/pkg/models"
)

// TestStream_LargeSSELineNotDropped reproduces a streaming data-loss bug.
//
// CompleteStream reads the SSE body with a fixed 4096-byte buffer via
// reader.Read(buf) and parses each Read() result line-by-line. An SSE event
// whose single "data: {...}\n" line is larger than the buffer is split across
// two Read() calls, so neither half is valid JSON, json.Unmarshal fails, and
// the content of that chunk is silently dropped. A real upstream can emit a
// content delta larger than 4096 bytes (long token bursts / code blocks).
//
// Every other provider in this module reads with bufio.Reader.ReadBytes('\n'),
// which respects line boundaries and does not have this defect.
func TestStream_LargeSSELineNotDropped(t *testing.T) {
	// One content delta whose JSON line exceeds the 4096-byte read buffer.
	bigContent := strings.Repeat("A", 8000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"" + bigContent + "\"}}]}\n"))
		_, _ = w.Write([]byte("data: [DONE]\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := NewSimpleOpenRouterProviderWithBaseURL("test-key", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &models.LLMRequest{ID: "req-big", ModelParams: models.ModelParameters{Model: "test-model"}}

	ch, err := p.CompleteStream(ctx, req)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}

	var assembled strings.Builder
	for resp := range ch {
		// Only count incremental chunk frames; the [DONE] final frame repeats
		// the accumulated content and would double-count.
		if isChunk, _ := resp.Metadata["is_chunk"].(bool); isChunk {
			assembled.WriteString(resp.Content)
		}
	}

	if assembled.Len() != len(bigContent) {
		t.Fatalf("streamed content length = %d, want %d (large SSE line dropped/truncated)",
			assembled.Len(), len(bigContent))
	}
}
