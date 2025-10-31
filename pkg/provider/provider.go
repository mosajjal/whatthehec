package provider

import (
	"context"
	"github.com/mosajjal/whatthehec/pkg/models"
)

// CloudProvider defines the interface for cloud-specific implementations
type CloudProvider interface {
	// Name returns the provider name (aws, azure, gcp)
	Name() string
	
	// ParseEvent parses cloud-specific event format into CloudEvent
	ParseEvent(ctx context.Context, rawEvent interface{}) (*models.CloudEvent, error)
	
	// ParseBatch parses multiple events (for streaming services)
	ParseBatch(ctx context.Context, rawEvent interface{}) ([]*models.CloudEvent, error)
}

// FunctionHandler defines the interface for cloud function entry points
type FunctionHandler interface {
	// Handle processes the cloud function invocation
	Handle(ctx context.Context, event interface{}) error
}
