package models

import (
	"testing"
	"time"
)

func TestLLMRequest_Creation(t *testing.T) {
	req := &LLMRequest{
		ID:        "test-request-123",
		SessionID: "test-session-456",
		UserID:    "test-user-789",
		Prompt:    "Write a simple Go function that adds two numbers",
		Messages:  []Message{},
		ModelParams: ModelParameters{
			Model:            "test-model",
			Temperature:      0.7,
			MaxTokens:        1000,
			TopP:             1.0,
			StopSequences:    []string{},
			ProviderSpecific: map[string]interface{}{},
		},
		EnsembleConfig: &EnsembleConfig{
			Strategy:            "confidence_weighted",
			MinProviders:        2,
			ConfidenceThreshold: 0.8,
			FallbackToBest:      true,
			Timeout:             30,
			PreferredProviders:  []string{"test-provider"},
		},
		MemoryEnhanced: false,
		Memory:         map[string]string{},
		Status:         "pending",
		CreatedAt:      time.Now(),
		RequestType:    "code_generation",
	}

	if req == nil {
		t.Fatal("Expected non-nil request")
	}
	if req.ID != "test-request-123" {
		t.Errorf("Expected ID 'test-request-123', got %q", req.ID)
	}
	if req.SessionID != "test-session-456" {
		t.Errorf("Expected SessionID 'test-session-456', got %q", req.SessionID)
	}
	if req.ModelParams.Model != "test-model" {
		t.Errorf("Expected Model 'test-model', got %q", req.ModelParams.Model)
	}
	if req.ModelParams.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", req.ModelParams.Temperature)
	}
	if req.EnsembleConfig.Strategy != "confidence_weighted" {
		t.Errorf("Expected Strategy 'confidence_weighted', got %q", req.EnsembleConfig.Strategy)
	}
}

func TestLLMResponse_Creation(t *testing.T) {
	resp := &LLMResponse{
		ID:             "test-response-123",
		RequestID:      "test-request-123",
		ProviderID:     "test-provider",
		ProviderName:   "Test Provider",
		Content:        "func add(a, b int) int { return a + b }",
		Confidence:     0.95,
		TokensUsed:     50,
		ResponseTime:   500,
		FinishReason:   "stop",
		Metadata:       map[string]interface{}{},
		Selected:       true,
		SelectionScore: 0.95,
		CreatedAt:      time.Now(),
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.ID != "test-response-123" {
		t.Errorf("Expected ID 'test-response-123', got %q", resp.ID)
	}
	if resp.Confidence != 0.95 {
		t.Errorf("Expected Confidence 0.95, got %f", resp.Confidence)
	}
	if resp.TokensUsed != 50 {
		t.Errorf("Expected TokensUsed 50, got %d", resp.TokensUsed)
	}
	if !resp.Selected {
		t.Error("Expected Selected to be true")
	}
}

func TestProviderCapabilities_DefaultValues(t *testing.T) {
	cap := &ProviderCapabilities{}

	if cap.SupportedModels != nil {
		t.Errorf("SupportedModels should be nil, got %v", cap.SupportedModels)
	}
	if cap.SupportsStreaming {
		t.Error("SupportsStreaming should be false")
	}
	if cap.SupportsFunctionCalling {
		t.Error("SupportsFunctionCalling should be false")
	}
	if cap.SupportsVision {
		t.Error("SupportsVision should be false")
	}
	if cap.SupportsTools {
		t.Error("SupportsTools should be false")
	}
	if cap.Limits.MaxTokens != 0 {
		t.Errorf("MaxTokens should be 0, got %d", cap.Limits.MaxTokens)
	}
}

func TestProviderCapabilities_Full(t *testing.T) {
	cap := &ProviderCapabilities{
		SupportedModels:         []string{"gpt-4"},
		SupportedFeatures:       []string{"chat"},
		SupportedRequestTypes:   []string{"text_completion"},
		SupportsStreaming:       true,
		SupportsFunctionCalling: true,
		SupportsTools:           true,
		SupportsReasoning:       true,
		Limits: ModelLimits{
			MaxTokens:             4096,
			MaxInputLength:        8192,
			MaxOutputLength:       4096,
			MaxConcurrentRequests: 10,
		},
		Metadata: map[string]string{"provider": "test"},
	}

	if len(cap.SupportedModels) != 1 {
		t.Errorf("Expected 1 model, got %d", len(cap.SupportedModels))
	}
	if !cap.SupportsStreaming {
		t.Error("Expected SupportsStreaming to be true")
	}
	if cap.Limits.MaxTokens != 4096 {
		t.Errorf("Expected MaxTokens 4096, got %d", cap.Limits.MaxTokens)
	}
}

func TestModelParameters_Defaults(t *testing.T) {
	params := ModelParameters{
		Model:            "default-model",
		Temperature:      0.7,
		MaxTokens:        1000,
		TopP:             1.0,
		StopSequences:    []string{},
		ProviderSpecific: map[string]interface{}{},
	}

	if params.Model != "default-model" {
		t.Errorf("Expected Model 'default-model', got %q", params.Model)
	}
	if params.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", params.Temperature)
	}
	if len(params.StopSequences) != 0 {
		t.Errorf("Expected 0 StopSequences, got %d", len(params.StopSequences))
	}
}

func TestEnsembleConfig_Validation(t *testing.T) {
	config := EnsembleConfig{
		Strategy:            "confidence_weighted",
		MinProviders:        2,
		ConfidenceThreshold: 0.8,
		FallbackToBest:      true,
		Timeout:             30,
		PreferredProviders:  []string{"provider1", "provider2"},
	}

	if config.Strategy != "confidence_weighted" {
		t.Errorf("Expected Strategy 'confidence_weighted', got %q", config.Strategy)
	}
	if config.MinProviders != 2 {
		t.Errorf("Expected MinProviders 2, got %d", config.MinProviders)
	}
	if len(config.PreferredProviders) != 2 {
		t.Errorf("Expected 2 PreferredProviders, got %d", len(config.PreferredProviders))
	}
}

func TestMessage_Creation(t *testing.T) {
	msg := Message{
		Role:      "user",
		Content:   "Hello, world!",
		Name:      nil,
		ToolCalls: map[string]interface{}{},
	}

	if msg.Role != "user" {
		t.Errorf("Expected Role 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Expected Content 'Hello, world!', got %q", msg.Content)
	}
	if msg.Name != nil {
		t.Error("Expected Name to be nil")
	}
}

func TestToolCall_Creation(t *testing.T) {
	tc := ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "get_weather",
			Arguments: `{"location": "NYC"}`,
		},
	}

	if tc.ID != "call-123" {
		t.Errorf("Expected ID 'call-123', got %q", tc.ID)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Expected Name 'get_weather', got %q", tc.Function.Name)
	}
}

func BenchmarkLLMRequestCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &LLMRequest{
			ID:        "test-request-123",
			SessionID: "test-session-456",
			UserID:    "test-user-789",
			Prompt:    "Write a simple Go function",
			Messages:  []Message{},
			ModelParams: ModelParameters{
				Model:       "test-model",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			Status:    "pending",
			CreatedAt: time.Now(),
		}
	}
}

func BenchmarkLLMResponseCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &LLMResponse{
			ID:           "test-response-123",
			RequestID:    "test-request-123",
			ProviderID:   "test-provider",
			ProviderName: "Test Provider",
			Content:      "func add(a, b int) int { return a + b }",
			Confidence:   0.95,
			TokensUsed:   50,
			ResponseTime: 500,
			FinishReason: "stop",
			Metadata:     map[string]interface{}{},
			CreatedAt:    time.Now(),
		}
	}
}
