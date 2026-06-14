package zen

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"digital.vasic.llmprovider/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RED: zen_http Complete drops req.Prompt (system instruction) when req.Messages
// is non-empty. The prompt builder only consults req.Messages and the
// `if prompt == "" && req.Prompt != ""` fallback never fires when messages exist,
// so the system prompt is silently never sent to the server.
func TestZenHTTP_Complete_DropsSystemPromptWhenMessagesPresent(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/messages") {
			body, _ := io.ReadAll(r.Body)
			var mr messageRequest
			_ = json.Unmarshal(body, &mr)
			received = mr.Content
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(messageResponse{
				ID: "m1", Role: "assistant", Content: "ok", Model: "big-pickle",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sessionResponse{ID: "s1"})
	}))
	defer server.Close()

	p := NewZenHTTPProvider(ZenHTTPConfig{BaseURL: server.URL, AutoStart: false})
	p.serverStarted = true

	_, err := p.Complete(context.Background(), &models.LLMRequest{
		Prompt: "SYSTEM_INSTRUCTION_MARKER",
		Messages: []models.Message{
			{Role: "user", Content: "hello there"},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, received, "SYSTEM_INSTRUCTION_MARKER",
		"system prompt (req.Prompt) MUST be forwarded to the server even when req.Messages is non-empty")
}

// RED: zen_http Complete reads/writes p.sessionID without synchronization.
// Concurrent Complete calls (the provider advertises MaxConcurrentRequests=10)
// race on p.sessionID. Run with -race.
func TestZenHTTP_Complete_ConcurrentSessionRace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/messages") {
			_ = json.NewEncoder(w).Encode(messageResponse{ID: "m1", Role: "assistant", Content: "ok", Model: "big-pickle"})
			return
		}
		_ = json.NewEncoder(w).Encode(sessionResponse{ID: "s1"})
	}))
	defer server.Close()

	p := NewZenHTTPProvider(ZenHTTPConfig{BaseURL: server.URL, AutoStart: false})
	p.serverStarted = true

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = p.Complete(context.Background(), &models.LLMRequest{Prompt: "hi"})
		}()
	}
	wg.Wait()
}
