// Package llmprovider provides LLM provider abstractions and utilities for Go applications.
//
// This package defines the core LLMProvider interface and common utilities for
// building LLM provider implementations, including circuit breakers, health monitoring,
// retry logic, and lazy loading.
//
// # Core Components
//
//   - LLMProvider: Interface that all provider implementations must satisfy
//   - CircuitBreaker: Fault tolerance for provider failures
//   - HealthMonitor: Health monitoring and status tracking
//   - RetryConfig: Configurable retry logic with exponential backoff
//   - LazyProvider: Lazy initialization of providers
//
// # Provider Interface
//
// All providers implement the LLMProvider interface:
//
//	type LLMProvider interface {
//	    Complete(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error)
//	    CompleteStream(ctx context.Context, req *models.LLMRequest) (<-chan *models.LLMResponse, error)
//	    HealthCheck() error
//	    GetCapabilities() *models.ProviderCapabilities
//	    ValidateConfig(config map[string]interface{}) (bool, []string)
//	}
//
// # Circuit Breaker Pattern
//
// The circuit breaker prevents cascading failures when providers are unhealthy:
//
//   - Closed: Normal operation, requests pass through
//   - Open: Provider is failing, requests are short-circuited
//   - Half-Open: Testing if provider has recovered
//
// # Health Monitoring
//
// HealthMonitor tracks provider health with configurable thresholds and intervals,
// providing real-time health status and automatic failure detection.
//
// # Retry Logic
//
// Configurable retry with exponential backoff, jitter, and HTTP status code detection
// for retryable errors (429, 500, 502, 503, 504).
//
// # Example Usage
//
//	import (
//	    "context"
//	    "digital.vasic.llmprovider"
//	    "digital.vasic.models"
//	)
//
//	func main() {
//	    provider := // create your provider implementation
//	    cb := llmprovider.NewDefaultCircuitBreaker("my-provider", provider)
//
//	    req := &models.LLMRequest{
//	        Prompt: "Hello, world!",
//	        MaxTokens: 100,
//	    }
//
//	    resp, err := cb.Complete(context.Background(), req)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    fmt.Println(resp.Text)
//	}
//
// # Dependencies
//
// This module depends on:
//
//   - digital.vasic.models: For LLMRequest, LLMResponse, and ProviderCapabilities types
//   - github.com/sirupsen/logrus: For logging in circuit breaker
//   - Standard library: context, sync, time, net/http, etc.
//
// # Module Information
//
// Module path: digital.vasic.llmprovider
// License: MIT
// Repository: https://github.com/vasic-digital/llmprovider
package llmprovider
