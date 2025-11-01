package azure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/provider"
)

// Provider implements the CloudProvider interface for Azure
type Provider struct {
	extractLogEvents bool
}

// NewProvider creates a new Azure provider
func NewProvider(extractLogEvents bool) provider.CloudProvider {
	return &Provider{
		extractLogEvents: extractLogEvents,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "azure"
}

// ParseEvent parses an Azure Monitor Logs event
func (p *Provider) ParseEvent(ctx context.Context, rawEvent interface{}) (*models.CloudEvent, error) {
	data, err := json.Marshal(rawEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	return &models.CloudEvent{
		ProviderType: "azure",
		RawData:      data,
	}, nil
}

// ParseBatch parses a batch of Azure events
func (p *Provider) ParseBatch(ctx context.Context, rawEvent interface{}) ([]*models.CloudEvent, error) {
	event, err := p.ParseEvent(ctx, rawEvent)
	if err != nil {
		return nil, err
	}
	return []*models.CloudEvent{event}, nil
}
