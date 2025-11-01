package aws

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(false)
	if provider.Name() != "aws" {
		t.Errorf("Expected provider name to be 'aws', got '%s'", provider.Name())
	}
}

func TestProvider_ParseEvent(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	// Test with empty event
	event := map[string]interface{}{
		"awslogs": map[string]interface{}{
			"data": "",
		},
	}

	cloudEvent, err := provider.ParseEvent(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cloudEvent.ProviderType != "aws" {
		t.Errorf("Expected provider type to be 'aws', got '%s'", cloudEvent.ProviderType)
	}
}

func TestDecodeCloudWatchData(t *testing.T) {
	// Test error cases
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "invalid base64",
			data:    "not-valid-base64!@#",
			wantErr: true,
		},
		{
			name:    "valid base64 but not gzip",
			data:    base64.StdEncoding.EncodeToString([]byte("not gzipped")),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeCloudWatchData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeCloudWatchData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_ParseBatch(t *testing.T) {
	provider := NewProvider(false)
	ctx := context.Background()

	// Test with Kinesis records structure
	event := map[string]interface{}{
		"records": []map[string]interface{}{},
	}

	events, err := provider.ParseBatch(ctx, event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

func TestProvider_ExtractLogEvents(t *testing.T) {
	provider := NewProvider(true).(*Provider)

	if !provider.extractLogEvents {
		t.Error("Expected extractLogEvents to be true")
	}
}
