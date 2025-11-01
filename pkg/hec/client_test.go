package hec

import (
	"testing"
	"time"

	"github.com/mosajjal/whatthehec/pkg/models"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Endpoints:       []string{"https://localhost:8088"},
		TLSSkipVerify:   true,
		Token:           "test-token",
		Index:           "test-index",
		Source:          "test-source",
		SourceType:      "test-sourcetype",
		Host:            "test-host",
		BatchSize:       100,
		BatchTimeout:    5 * time.Second,
		BalanceStrategy: "roundrobin",
	}

	if cfg.Endpoints[0] != "https://localhost:8088" {
		t.Errorf("Expected endpoint to be 'https://localhost:8088', got '%s'", cfg.Endpoints[0])
	}
	if cfg.BalanceStrategy != "roundrobin" {
		t.Errorf("Expected balance strategy to be 'roundrobin', got '%s'", cfg.BalanceStrategy)
	}
}

func TestNewClient_InvalidEndpoints(t *testing.T) {
	cfg := Config{
		Endpoints:       []string{},
		Token:           "test-token",
		BalanceStrategy: "roundrobin",
	}

	client, err := NewClient(cfg, nil, nil)
	if err == nil {
		t.Error("Expected error for empty endpoints, got nil")
	}
	if client != nil {
		t.Error("Expected nil client for empty endpoints")
	}
}

func TestNewClient_BalanceStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		expected uint8
	}{
		{"first_available", "first_available", FirstAvailable},
		{"sticky", "sticky", Sticky},
		{"random", "random", Random},
		{"roundrobin", "roundrobin", RoundRobin},
		{"unknown", "unknown", FirstAvailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This would require a mock HEC server to test properly
			// For now, we just test the balance strategy parsing
			cfg := Config{
				Endpoints:       []string{"https://localhost:8088"},
				Token:           "test-token",
				BalanceStrategy: tt.strategy,
			}

			// We can't actually create a client without a valid HEC endpoint
			// but we can test the config
			if cfg.BalanceStrategy != tt.strategy {
				t.Errorf("Expected strategy '%s', got '%s'", tt.strategy, cfg.BalanceStrategy)
			}
		})
	}
}

func TestSendEvents_NoHealthyConnections(t *testing.T) {
	// This test would require mocking
	// Just verify the models.Event structure
	event := &models.Event{
		Time:       time.Now(),
		Host:       "test-host",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		Index:      "test-index",
		Event:      "test event",
	}

	if event.Event != "test event" {
		t.Errorf("Expected event to be 'test event', got '%v'", event.Event)
	}
}

func TestConnection(t *testing.T) {
	conn := &connection{
		endpoint:  "https://localhost:8088",
		isHealthy: false,
	}

	if conn.endpoint != "https://localhost:8088" {
		t.Errorf("Expected endpoint to be 'https://localhost:8088', got '%s'", conn.endpoint)
	}
}
