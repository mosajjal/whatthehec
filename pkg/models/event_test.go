package models

import (
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	event := &Event{
		Time:       time.Now(),
		Host:       "test-host",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		Index:      "test-index",
		Event:      "test event data",
	}

	if event.Host != "test-host" {
		t.Errorf("Expected host to be 'test-host', got '%s'", event.Host)
	}
	if event.Source != "test-source" {
		t.Errorf("Expected source to be 'test-source', got '%s'", event.Source)
	}
	if event.Index != "test-index" {
		t.Errorf("Expected index to be 'test-index', got '%s'", event.Index)
	}
}

func TestCloudEvent(t *testing.T) {
	cloudEvent := &CloudEvent{
		ProviderType: "aws",
		Timestamp:    time.Now().Unix(),
		LogGroup:     "test-log-group",
		LogStream:    "test-log-stream",
		Message:      "test message",
		Metadata:     map[string]string{"key": "value"},
		RawData:      []byte("raw data"),
	}

	if cloudEvent.ProviderType != "aws" {
		t.Errorf("Expected provider type to be 'aws', got '%s'", cloudEvent.ProviderType)
	}
	if cloudEvent.LogGroup != "test-log-group" {
		t.Errorf("Expected log group to be 'test-log-group', got '%s'", cloudEvent.LogGroup)
	}
	if string(cloudEvent.RawData) != "raw data" {
		t.Errorf("Expected raw data to be 'raw data', got '%s'", string(cloudEvent.RawData))
	}
}
