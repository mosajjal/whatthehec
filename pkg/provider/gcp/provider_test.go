package gcp

import (
	"context"
	"testing"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(false)
	if provider.Name() != "gcp" {
		t.Errorf("Expected provider name to be 'gcp', got '%s'", provider.Name())
	}
}

func TestProvider_ParseEvent(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	event := map[string]interface{}{
		"insertId":   "test-id",
		"logName":    "projects/test-project/logs/test-log",
		"timestamp":  "2025-01-01T00:00:00Z",
		"textPayload": "test message",
	}

	cloudEvent, err := provider.ParseEvent(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cloudEvent.ProviderType != "gcp" {
		t.Errorf("Expected provider type to be 'gcp', got '%s'", cloudEvent.ProviderType)
	}

	if len(cloudEvent.RawData) == 0 {
		t.Error("Expected raw data to be populated")
	}
}

func TestProvider_ParseBatch(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	event := map[string]interface{}{
		"entries": []interface{}{},
	}

	events, err := provider.ParseBatch(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
