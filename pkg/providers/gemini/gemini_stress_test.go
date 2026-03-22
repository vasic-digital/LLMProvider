package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.llmprovider/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==============================================================================
// Stress Tests for the Gemini provider
// All stress tests use httptest servers -- no real API calls.
// ==============================================================================

// newMockGeminiServer creates an httptest server that returns a valid Gemini
// response with the given text content. An atomic counter tracks the number of
// requests served.
func newMockGeminiServer(
	content string,
	counter *int64,
) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if counter != nil {
				atomic.AddInt64(counter, 1)
			}
			resp := GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{
								{Text: content},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: &GeminiUsageMetadata{
					PromptTokenCount:     5,
					CandidatesTokenCount: 10,
					TotalTokenCount:      15,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		},
	))
}

// newMockStreamServer returns an httptest server that responds with a simple
// SSE stream containing the given content chunks.
func newMockStreamServer(
	chunks []string,
	counter *int64,
) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if counter != nil {
				atomic.AddInt64(counter, 1)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)

			for _, chunk := range chunks {
				streamResp := GeminiStreamResponse{
					Candidates: []GeminiCandidate{
						{
							Content: GeminiContent{
								Parts: []GeminiPart{
									{Text: chunk},
								},
							},
						},
					},
				}
				data, _ := json.Marshal(streamResp)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				if ok {
					flusher.Flush()
				}
			}

			// Final chunk with finish reason
			finalResp := GeminiStreamResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{
								{Text: ""},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: &GeminiUsageMetadata{
					TotalTokenCount: len(chunks) * 2,
				},
			}
			finalData, _ := json.Marshal(finalResp)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", finalData)
			if ok {
				flusher.Flush()
			}
		},
	))
}

// TestGeminiAPI_ConcurrentRequests launches 10 concurrent Complete() calls
// against an httptest server and verifies all succeed.
func TestGeminiAPI_ConcurrentRequests(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	var requestCount int64
	server := newMockGeminiServer("concurrent response", &requestCount)
	defer server.Close()

	baseURL := server.URL + "/v1beta/models/%s:generateContent"
	provider := NewGeminiAPIProvider("test-key", baseURL, "gemini-2.5-flash")

	const concurrency = 10
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	responses := make([]*models.LLMResponse, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(
				context.Background(), 10*time.Second,
			)
			defer cancel()

			req := &models.LLMRequest{
				ID: fmt.Sprintf("concurrent-%d", idx),
				Messages: []models.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("request %d", idx),
					},
				},
				ModelParams: models.ModelParameters{MaxTokens: 32},
			}

			resp, err := provider.Complete(ctx, req)
			errs[idx] = err
			responses[idx] = resp
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		assert.NoError(t, errs[i],
			"request %d should not error", i)
		require.NotNil(t, responses[i],
			"response %d should not be nil", i)
		assert.Equal(t, "concurrent response", responses[i].Content,
			"response %d should have correct content", i)
	}

	assert.Equal(t, int64(concurrency), atomic.LoadInt64(&requestCount),
		"server should have received exactly %d requests", concurrency)
}

// TestGeminiAPI_ConcurrentStreaming launches 5 concurrent CompleteStream()
// calls and verifies all streams complete without error.
func TestGeminiAPI_ConcurrentStreaming(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	chunks := []string{"chunk1 ", "chunk2 ", "chunk3"}
	var requestCount int64
	server := newMockStreamServer(chunks, &requestCount)
	defer server.Close()

	baseURL := server.URL + "/v1beta/models/%s:generateContent"
	streamURL := server.URL + "/v1beta/models/%s:streamGenerateContent"

	provider := NewGeminiAPIProviderWithRetry(
		"test-key", baseURL, "gemini-2.5-flash",
		RetryConfig{
			MaxRetries:   1,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     500 * time.Millisecond,
			Multiplier:   2.0,
		},
	)
	// Override stream URL
	provider.streamURL = streamURL

	const concurrency = 5
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	chunkCounts := make([]int, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(
				context.Background(), 10*time.Second,
			)
			defer cancel()

			req := &models.LLMRequest{
				ID: fmt.Sprintf("stream-%d", idx),
				Messages: []models.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("stream request %d", idx),
					},
				},
				ModelParams: models.ModelParameters{MaxTokens: 64},
			}

			ch, err := provider.CompleteStream(ctx, req)
			if err != nil {
				errs[idx] = err
				return
			}

			count := 0
			for range ch {
				count++
			}
			chunkCounts[idx] = count
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		assert.NoError(t, errs[i],
			"stream %d should not error", i)
		assert.Greater(t, chunkCounts[i], 0,
			"stream %d should have received chunks", i)
	}
}

// TestGeminiAPI_RapidHealthChecks performs 50 rapid health checks against an
// httptest server to verify no resource leaks or panics under load.
func TestGeminiAPI_RapidHealthChecks(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models": [{"name": "gemini-2.5-flash"}]}`))
		},
	))
	defer server.Close()

	provider := NewGeminiAPIProvider("test-key", "", "gemini-2.5-flash")
	provider.healthURL = server.URL + "/v1beta/models"

	const iterations = 50
	var wg sync.WaitGroup
	errs := make([]error, iterations)

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = provider.HealthCheck()
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		}
	}

	assert.Equal(t, iterations, successCount,
		"all %d health checks should succeed", iterations)
	assert.Equal(t, int64(iterations), atomic.LoadInt64(&requestCount),
		"server should have received exactly %d requests", iterations)
}

// TestGeminiUnified_ConcurrentInitialization verifies that 20 goroutines
// calling Initialize() simultaneously do not cause a race condition or panic.
// sync.Once inside the provider should ensure initialization happens exactly
// once.
func TestGeminiUnified_ConcurrentInitialization(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	const concurrency = 20
	var wg sync.WaitGroup
	errs := make([]error, concurrency)

	config := GeminiUnifiedConfig{
		APIKey:          "test-key-concurrent",
		Model:           "gemini-2.5-flash",
		PreferredMethod: "auto",
	}
	provider := NewGeminiUnifiedProvider(config)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = provider.Initialize()
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		assert.NoError(t, errs[i],
			"initialization %d should not error", i)
	}

	// Verify the provider is actually initialized and consistent
	assert.Equal(t, "gemini-2.5-flash", provider.GetCurrentModel())
	assert.Equal(t, "auto", provider.GetPreferredMethod())
}

// TestGeminiAPI_SequentialRetries creates a server that returns 429 for the
// first 2 requests and then returns 200, verifying retry logic under load.
func TestGeminiAPI_SequentialRetries(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt64(&requestCount, 1)

			if count <= 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error": "rate limited"}`))
				return
			}

			resp := GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{
								{Text: "success after retries"},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: &GeminiUsageMetadata{
					TotalTokenCount: 10,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		},
	))
	defer server.Close()

	baseURL := server.URL + "/v1beta/models/%s:generateContent"

	retryConfig := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
	}
	provider := NewGeminiAPIProviderWithRetry(
		"test-key", baseURL, "gemini-2.5-flash", retryConfig,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &models.LLMRequest{
		ID: "test-retry",
		Messages: []models.Message{
			{Role: "user", Content: "test retry"},
		},
		ModelParams: models.ModelParameters{MaxTokens: 32},
	}

	resp, err := provider.Complete(ctx, req)
	require.NoError(t, err, "should succeed after retries")
	require.NotNil(t, resp)
	assert.Equal(t, "success after retries", resp.Content)
	assert.GreaterOrEqual(t, atomic.LoadInt64(&requestCount), int64(3),
		"should have made at least 3 requests (2 failures + 1 success)")
}

// TestGeminiCLI_ConcurrentModelDiscovery launches 10 goroutines calling
// DiscoverModels() simultaneously to verify thread safety of the once-based
// discovery mechanism.
func TestGeminiCLI_ConcurrentModelDiscovery(t *testing.T) {
	runtime.GOMAXPROCS(2)
	t.Parallel()

	const concurrency = 10
	var wg sync.WaitGroup

	provider := NewGeminiCLIProvider(GeminiCLIConfig{
		Model:   "gemini-2.5-flash",
		Timeout: 5 * time.Second,
	})

	results := make([][]string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = provider.DiscoverModels()
		}(i)
	}

	wg.Wait()

	// All goroutines should get the same result (sync.Once guarantees this)
	require.NotEmpty(t, results[0],
		"first result should have models (at least fallback)")

	for i := 1; i < concurrency; i++ {
		assert.Equal(t, results[0], results[i],
			"all goroutines should get the same model list")
	}
}
