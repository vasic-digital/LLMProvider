// Package models provides shared data types for LLM provider communication.
//
// These types define the common request/response structures used across all
// LLM provider implementations.
package models

import "time"

// LLMRequest represents a request to an LLM provider
type LLMRequest struct {
	ID             string            `json:"id" db:"id"`
	SessionID      string            `json:"session_id" db:"session_id"`
	UserID         string            `json:"user_id" db:"user_id"`
	Prompt         string            `json:"prompt" db:"prompt"`
	Messages       []Message         `json:"messages" db:"messages"`
	ModelParams    ModelParameters   `json:"model_params" db:"model_params"`
	EnsembleConfig *EnsembleConfig   `json:"ensemble_config" db:"ensemble_config"`
	MemoryEnhanced bool              `json:"memory_enhanced" db:"memory_enhanced"`
	Memory         map[string]string `json:"memory" db:"memory"`
	Status         string            `json:"status" db:"status"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	StartedAt      *time.Time        `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time        `json:"completed_at" db:"completed_at"`
	RequestType    string            `json:"request_type" db:"request_type"`
	// Tools available for the LLM to call (OpenAI format)
	Tools []Tool `json:"tools,omitempty"`
	// ToolChoice specifies how the model should use tools ("none", "auto", "required", or specific tool)
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// Tool represents a tool available for the LLM to call
type Tool struct {
	Type     string       `json:"type"` // Always "function" for now
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function that can be called
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// LLMResponse represents a response from an LLM provider
type LLMResponse struct {
	ID             string                 `json:"id" db:"id"`
	RequestID      string                 `json:"request_id" db:"request_id"`
	ProviderID     string                 `json:"provider_id" db:"provider_id"`
	ProviderName   string                 `json:"provider_name" db:"provider_name"`
	Content        string                 `json:"content" db:"content"`
	Confidence     float64                `json:"confidence" db:"confidence"`
	TokensUsed     int                    `json:"tokens_used" db:"tokens_used"`
	ResponseTime   int64                  `json:"response_time" db:"response_time"`
	FinishReason   string                 `json:"finish_reason" db:"finish_reason"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Selected       bool                   `json:"selected" db:"selected"`
	SelectionScore float64                `json:"selection_score" db:"selection_score"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	// ToolCalls returned by the LLM when it wants to use tools
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call requested by the LLM
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // Always "function" for now
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the function name and arguments to call
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of arguments
}

// Message represents a message in a conversation
type Message struct {
	Role      string                 `json:"role" db:"role"`
	Content   string                 `json:"content" db:"content"`
	Name      *string                `json:"name" db:"name"`
	ToolCalls map[string]interface{} `json:"tool_calls" db:"tool_calls"`
}

// ModelParameters defines model-specific parameters for LLM requests
type ModelParameters struct {
	Model            string                 `json:"model" db:"model"`
	Temperature      float64                `json:"temperature" db:"temperature"`
	MaxTokens        int                    `json:"max_tokens" db:"max_tokens"`
	TopP             float64                `json:"top_p" db:"top_p"`
	StopSequences    []string               `json:"stop_sequences" db:"stop_sequences"`
	ProviderSpecific map[string]interface{} `json:"provider_specific" db:"provider_specific"`
}

// EnsembleConfig configures ensemble behavior for multi-provider requests
type EnsembleConfig struct {
	Strategy            string   `json:"strategy" db:"strategy"`
	MinProviders        int      `json:"min_providers" db:"min_providers"`
	ConfidenceThreshold float64  `json:"confidence_threshold" db:"confidence_threshold"`
	FallbackToBest      bool     `json:"fallback_to_best" db:"fallback_to_best"`
	Timeout             int      `json:"timeout" db:"timeout"`
	PreferredProviders  []string `json:"preferred_providers" db:"preferred_providers"`
}

// ProviderCapabilities describes capabilities exposed by an LLM provider.
type ProviderCapabilities struct {
	SupportedModels         []string          `json:"supported_models"`
	SupportedFeatures       []string          `json:"supported_features"`
	SupportedRequestTypes   []string          `json:"supported_request_types"`
	SupportsStreaming       bool              `json:"supports_streaming"`
	SupportsFunctionCalling bool              `json:"supports_function_calling"`
	SupportsVision          bool              `json:"supports_vision"`
	Limits                  ModelLimits       `json:"limits"`
	Metadata                map[string]string `json:"metadata"`

	// LSP specific capabilities
	SupportsTools          bool `json:"supports_tools"`
	SupportsSearch         bool `json:"supports_search"`
	SupportsReasoning      bool `json:"supports_reasoning"`
	SupportsCodeCompletion bool `json:"supports_code_completion"`
	SupportsCodeAnalysis   bool `json:"supports_code_analysis"`
	SupportsRefactoring    bool `json:"supports_refactoring"`
}

// ModelLimits defines the operational limits of an LLM model.
type ModelLimits struct {
	MaxTokens             int `json:"max_tokens"`
	MaxInputLength        int `json:"max_input_length"`
	MaxOutputLength       int `json:"max_output_length"`
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}

// LLMProviderRecord represents a provider configuration record (for storage/DB use)
type LLMProviderRecord struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Type         string                 `json:"type" db:"type"`
	APIKey       string                 `json:"-" db:"api_key"`
	BaseURL      string                 `json:"base_url" db:"base_url"`
	Model        string                 `json:"model" db:"model"`
	Weight       float64                `json:"weight" db:"weight"`
	Enabled      bool                   `json:"enabled" db:"enabled"`
	Config       map[string]interface{} `json:"config" db:"config"`
	HealthStatus string                 `json:"health_status" db:"health_status"`
	ResponseTime int64                  `json:"response_time" db:"response_time"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}
