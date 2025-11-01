package azure

import (
	"context"
	"testing"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(false)
	if provider.Name() != "azure" {
		t.Errorf("Expected provider name to be 'azure', got '%s'", provider.Name())
	}
}

func TestProvider_ParseEvent(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	event := map[string]interface{}{
		"records": []map[string]interface{}{
			{
				"time":         "2025-01-01T00:00:00Z",
				"category":     "Administrative",
				"operationName": "Microsoft.Compute/virtualMachines/write",
			},
		},
	}

	cloudEvent, err := provider.ParseEvent(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cloudEvent.ProviderType != "azure" {
		t.Errorf("Expected provider type to be 'azure', got '%s'", cloudEvent.ProviderType)
	}

	if len(cloudEvent.RawData) == 0 {
		t.Error("Expected raw data to be populated")
	}
}

func TestProvider_ParseBatch(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	event := map[string]interface{}{
		"records": []interface{}{},
	}

	events, err := provider.ParseBatch(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
