package stepfun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"digital.vasic.llmprovider/pkg/discovery"
	"digital.vasic.llmprovider/pkg/models"
)

const (
	StepfunAPIURL     = "https://api.stepfun.com/v1/chat/completions"
	StepfunModel      = "step-1.5v-mini"
	StepfunModelsURL  = "https://api.stepfun.com/v1/models"
	StepfunMaxContext = 32768
	StepfunMaxOutput  = 8192
)

type StepfunProvider struct {
	apiKey      string
	baseURL     string
	model       string
	httpClient  *http.Client
	retryConfig RetryConfig
	discoverer  *discovery.Discoverer
}

type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

type StepfunRequest struct {
	Model       string           `json:"model"`
	Messages    []StepfunMessage `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type StepfunMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StepfunResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []StepfunChoice `json:"choices"`
	Usage   StepfunUsage    `json:"usage"`
}

type StepfunChoice struct {
	Index        int            `json:"index"`
	Message      StepfunMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type StepfunUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StepfunStreamResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []StepfunStreamChoice `json:"choices"`
}

type StepfunStreamChoice struct {
	Index        int            `json:"index"`
	Delta        StepfunMessage `json:"delta"`
	FinishReason *string        `json:"finish_reason"`
}

type StepfunErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

func NewStepfunProvider(apiKey, baseURL, model string) *StepfunProvider {
	return NewStepfunProviderWithRetry(apiKey, baseURL, model, DefaultRetryConfig())
}

func NewStepfunProviderWithRetry(apiKey, baseURL, model string, retryConfig RetryConfig) *StepfunProvider {
	if baseURL == "" {
		baseURL = StepfunAPIURL
	}
	if model == "" {
		model = StepfunModel
	}

	p := &StepfunProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		retryConfig: retryConfig,
	}

	p.discoverer = discovery.NewDiscoverer(discovery.ProviderConfig{
		ProviderName:   "stepfun",
		ModelsEndpoint: StepfunModelsURL,
		ModelsDevID:    "stepfun",
		APIKey:         apiKey,
		FallbackModels: []string{
			"step-1.5v-mini",
			"step-2-16k",
			"step-1-8k",
			"step-1-32k",
			"step-1-128k",
			"step-1-256k",
		},
	})

	return p
}

func (p *StepfunProvider) Complete(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	startTime := time.Now()
	requestID := req.ID
	if requestID == "" {
		requestID = fmt.Sprintf("stepfun-%d", time.Now().UnixNano())
	}

	sReq := p.convertRequest(req)

	resp, err := p.makeAPICall(ctx, sReq)
	if err != nil {
		return nil, fmt.Errorf("Stepfun API call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp StepfunErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Stepfun API error: %d - %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Stepfun API error: %d - %s", resp.StatusCode, string(body))
	}

	var sResp StepfunResponse
	if err := json.Unmarshal(body, &sResp); err != nil {
		return nil, fmt.Errorf("failed to parse Stepfun response: %w", err)
	}

	if len(sResp.Choices) == 0 {
		return nil, fmt.Errorf("Stepfun API returned no choices")
	}

	return p.convertResponse(req, &sResp, startTime), nil
}

func (p *StepfunProvider) CompleteStream(ctx context.Context, req *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	startTime := time.Now()

	sReq := p.convertRequest(req)
	sReq.Stream = true

	resp, err := p.makeAPICall(ctx, sReq)
	if err != nil {
		return nil, fmt.Errorf("Stepfun streaming API call failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Stepfun API error: HTTP %d - failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("Stepfun API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	ch := make(chan *models.LLMResponse)

	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		var fullContent string

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				ch <- &models.LLMResponse{
					ID:           "stream-error-" + req.ID,
					RequestID:    req.ID,
					ProviderID:   "stepfun",
					ProviderName: "Stepfun",
					FinishReason: "error",
					CreatedAt:    time.Now(),
				}
				return
			}

			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}
			line = bytes.TrimPrefix(line, []byte("data: "))

			if bytes.Equal(line, []byte("[DONE]")) {
				break
			}

			var streamResp StepfunStreamResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta.Content
				if delta != "" {
					fullContent += delta
					ch <- &models.LLMResponse{
						ID:           streamResp.ID,
						RequestID:    req.ID,
						ProviderID:   "stepfun",
						ProviderName: "Stepfun",
						Content:      delta,
						Confidence:   0.8,
						TokensUsed:   1,
						ResponseTime: time.Since(startTime).Milliseconds(),
						CreatedAt:    time.Now(),
					}
				}

				if streamResp.Choices[0].FinishReason != nil {
					break
				}
			}
		}

		ch <- &models.LLMResponse{
			ID:           "stream-final-" + req.ID,
			RequestID:    req.ID,
			ProviderID:   "stepfun",
			ProviderName: "Stepfun",
			Content:      "",
			Confidence:   0.8,
			TokensUsed:   len(fullContent) / 4,
			ResponseTime: time.Since(startTime).Milliseconds(),
			FinishReason: "stop",
			CreatedAt:    time.Now(),
		}
	}()

	return ch, nil
}

func (p *StepfunProvider) convertRequest(req *models.LLMRequest) StepfunRequest {
	messages := make([]StepfunMessage, 0, len(req.Messages)+1)

	if req.Prompt != "" {
		messages = append(messages, StepfunMessage{Role: "system", Content: req.Prompt})
	}

	for _, msg := range req.Messages {
		messages = append(messages, StepfunMessage{Role: msg.Role, Content: msg.Content})
	}

	maxTokens := req.ModelParams.MaxTokens
	if maxTokens <= 0 {
		maxTokens = StepfunMaxOutput
	} else if maxTokens > StepfunMaxOutput {
		maxTokens = StepfunMaxOutput
	}

	return StepfunRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: req.ModelParams.Temperature,
		MaxTokens:   maxTokens,
		TopP:        req.ModelParams.TopP,
		Stream:      false,
	}
}

func (p *StepfunProvider) convertResponse(req *models.LLMRequest, sResp *StepfunResponse, startTime time.Time) *models.LLMResponse {
	var content, finishReason string
	if len(sResp.Choices) > 0 {
		content = sResp.Choices[0].Message.Content
		finishReason = sResp.Choices[0].FinishReason
	}

	confidence := 0.8
	if finishReason == "stop" {
		confidence += 0.1
	}
	if len(content) > 100 {
		confidence += 0.05
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return &models.LLMResponse{
		ID:           sResp.ID,
		RequestID:    req.ID,
		ProviderID:   "stepfun",
		ProviderName: "Stepfun",
		Content:      content,
		Confidence:   confidence,
		TokensUsed:   sResp.Usage.TotalTokens,
		ResponseTime: time.Since(startTime).Milliseconds(),
		FinishReason: finishReason,
		Metadata: map[string]any{
			"model":             sResp.Model,
			"prompt_tokens":     sResp.Usage.PromptTokens,
			"completion_tokens": sResp.Usage.CompletionTokens,
		},
		CreatedAt: time.Now(),
	}
}

func (p *StepfunProvider) makeAPICall(ctx context.Context, req StepfunRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	delay := p.retryConfig.InitialDelay

	for attempt := 0; attempt <= p.retryConfig.MaxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("User-Agent", "LLMProvider/1.0")

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			if attempt < p.retryConfig.MaxRetries {
				jitter := time.Duration(rand.Float64() * 0.1 * float64(delay))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay + jitter):
				}
				delay = time.Duration(float64(delay) * p.retryConfig.Multiplier)
				if delay > p.retryConfig.MaxDelay {
					delay = p.retryConfig.MaxDelay
				}
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			if attempt < p.retryConfig.MaxRetries {
				_ = resp.Body.Close()
				lastErr = fmt.Errorf("HTTP %d: retryable error", resp.StatusCode)
				jitter := time.Duration(rand.Float64() * 0.1 * float64(delay))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay + jitter):
				}
				delay = time.Duration(float64(delay) * p.retryConfig.Multiplier)
				if delay > p.retryConfig.MaxDelay {
					delay = p.retryConfig.MaxDelay
				}
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// modelsURL derives the models endpoint URL from the base URL.
func (p *StepfunProvider) modelsURL() string {
	// If baseURL is the default chat completions URL, use the known models URL
	if p.baseURL == StepfunAPIURL {
		return StepfunModelsURL
	}
	// For custom base URLs (e.g., test servers), append /models
	return p.baseURL + "/models"
}

func (p *StepfunProvider) GetCapabilities() *models.ProviderCapabilities {
	return &models.ProviderCapabilities{
		SupportedModels:        p.discoverer.DiscoverModels(),
		SupportedFeatures:      []string{"text_completion", "chat", "streaming", "vision", "gui_grounding"},
		SupportsStreaming:      true,
		SupportsVision:         true,
		SupportsCodeCompletion: true,
		SupportsCodeAnalysis:   true,
		Limits: models.ModelLimits{
			MaxTokens:       StepfunMaxOutput,
			MaxInputLength:  StepfunMaxContext,
			MaxOutputLength: StepfunMaxOutput,
		},
		Metadata: map[string]string{
			"provider": "Stepfun",
			"note":     "Stepfun Step-GUI vision and language models",
		},
	}
}

func (p *StepfunProvider) ValidateConfig(config map[string]interface{}) (bool, []string) {
	var errors []string
	if p.apiKey == "" {
		errors = append(errors, "API key is required")
	}
	return len(errors) == 0, errors
}

func (p *StepfunProvider) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Derive the models URL from the base URL (strip /chat/completions, add /models)
	modelsURL := p.modelsURL()
	req, _ := http.NewRequestWithContext(ctx, "GET", modelsURL, nil) //nolint:errcheck
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	return nil
}
