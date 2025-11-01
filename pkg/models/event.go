package models

import "time"

// Event represents a log event to be sent to Splunk HEC
type Event struct {
	Time       time.Time
	Host       string
	Source     string
	SourceType string
	Index      string
	Event      interface{}
}

// CloudEvent represents a cloud provider-agnostic log event
type CloudEvent struct {
	ProviderType string      // aws, azure, gcp
	Timestamp    int64       // Unix timestamp
	LogGroup     string      // Source log group/stream
	LogStream    string      // Specific log stream
	Message      string      // Log message
	Metadata     map[string]string // Additional metadata
	RawData      []byte      // Original raw data
}
