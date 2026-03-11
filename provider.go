package llmprovider

import (
	"context"
	"digital.vasic.models"
)

// LLMProvider defines an interface for LLM providers to integrate with the facade.
type LLMProvider interface {
	Complete(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error)
	CompleteStream(ctx context.Context, req *models.LLMRequest) (<-chan *models.LLMResponse, error)
	HealthCheck() error
	GetCapabilities() *models.ProviderCapabilities
	ValidateConfig(config map[string]interface{}) (bool, []string)
}
