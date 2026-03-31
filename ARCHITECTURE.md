# Architecture -- LLMProvider

## Purpose

Standalone Go module providing a unified interface for 43+ LLM providers with built-in fault tolerance. Includes circuit breaker for automatic fault isolation, health monitoring with latency tracking, retry with exponential backoff and jitter, 3-tier dynamic model discovery, streaming support, and tool/function calling.

## Structure

```
pkg/
  provider/    LLMProvider interface: Complete, Stream, Available, Name
  models/      LLMRequest, LLMResponse, ProviderCapabilities, ModelParameters
  retry/       RetryConfig, ExecuteWithRetry with exponential backoff and jitter
  circuit/     CircuitBreaker (closed/open/half-open), CircuitBreakerManager
  health/      HealthMonitor with periodic checks, latency tracking, status history
  http/        HTTP client with retry for provider API calls
  discovery/   3-tier model discovery: provider API -> models.dev -> fallback list
  providers/   43+ provider implementations (OpenAI, Anthropic, Gemini, Groq, Mistral, DeepSeek, Ollama, etc.)
```

## Key Components

- **`provider.LLMProvider`** -- Interface: Complete(ctx, request), Stream(ctx, request), Available(), Name()
- **`models.LLMRequest`** -- Prompt, ModelParams (Temperature, MaxTokens, TopP), Tools, SystemPrompt
- **`circuit.CircuitBreaker`** -- 3 consecutive failures = open; configurable timeout for half-open transition
- **`circuit.CircuitBreakerManager`** -- Per-provider circuit breaker registry
- **`health.HealthMonitor`** -- Periodic availability checks, latency percentile tracking, status history
- **`discovery.ModelDiscovery`** -- 3-tier: provider's native API -> models.dev aggregator -> hardcoded fallback; TTL-cached

## Data Flow

```
LLMProvider.Complete(ctx, request)
    |
    CircuitBreaker.Execute() -> check state
        |
        Closed -> retry.ExecuteWithRetry(fn, config)
            |
            provider-specific HTTP call -> parse response -> LLMResponse
        |
        Open -> ErrCircuitOpen
    |
    HealthMonitor tracks latency and success/failure

ModelDiscovery.ListModels(provider)
    |
    tier 1: provider.ListModels() API call
    tier 2: models.dev API query (fallback)
    tier 3: hardcoded model list (final fallback)
    |
    cache with configurable TTL
```

## Dependencies

- `github.com/sirupsen/logrus` -- Structured logging
- `github.com/stretchr/testify` -- Test assertions

## Testing Strategy

Table-driven tests with `testify` and race detection. Core tests (models, retry, circuit, health) run without network calls. Provider tests verify request construction and response parsing. Circuit breaker tests verify state machine transitions.
