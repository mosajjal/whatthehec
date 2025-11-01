package aws

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/provider"
)

// Provider implements the CloudProvider interface for AWS
type Provider struct {
	extractLogEvents bool
}

// NewProvider creates a new AWS provider
func NewProvider(extractLogEvents bool) provider.CloudProvider {
	return &Provider{
		extractLogEvents: extractLogEvents,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "aws"
}

// CloudWatchLogs represents AWS CloudWatch Logs event structure
type CloudWatchLogs struct {
	AWSLogs struct {
		Data string `json:"data"`
	} `json:"awslogs"`
	Records []struct {
		RecordID string `json:"recordId"`
		Data     string `json:"data"`
	} `json:"records"`
}

// CloudWatchLogsData represents the decoded CloudWatch Logs data
type CloudWatchLogsData struct {
	MessageType         string     `json:"messageType"`
	Owner               string     `json:"owner"`
	LogGroup            string     `json:"logGroup"`
	LogStream           string     `json:"logStream"`
	SubscriptionFilters []string   `json:"subscriptionFilters"`
	LogEvents           []LogEvent `json:"logEvents"`
}

// LogEvent represents a single log event
type LogEvent struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// ParseEvent parses an AWS CloudWatch Logs event
func (p *Provider) ParseEvent(ctx context.Context, rawEvent interface{}) (*models.CloudEvent, error) {
	data, err := json.Marshal(rawEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	var cwLogs CloudWatchLogs
	if err := json.Unmarshal(data, &cwLogs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudWatch Logs: %w", err)
	}

	// Decode the base64 + gzip data
	var decodedData []byte
	if cwLogs.AWSLogs.Data != "" {
		decodedData, err = decodeCloudWatchData(cwLogs.AWSLogs.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CloudWatch data: %w", err)
		}
	}

	return &models.CloudEvent{
		ProviderType: "aws",
		RawData:      decodedData,
	}, nil
}

// ParseBatch parses a batch of AWS events (for Kinesis)
func (p *Provider) ParseBatch(ctx context.Context, rawEvent interface{}) ([]*models.CloudEvent, error) {
	data, err := json.Marshal(rawEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	var cwLogs CloudWatchLogs
	if err := json.Unmarshal(data, &cwLogs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudWatch Logs: %w", err)
	}

	var events []*models.CloudEvent

	// Handle Kinesis records
	if len(cwLogs.Records) > 0 {
		for _, record := range cwLogs.Records {
			decodedData, err := decodeCloudWatchData(record.Data)
			if err != nil {
				continue
			}
			events = append(events, &models.CloudEvent{
				ProviderType: "aws",
				RawData:      decodedData,
			})
		}
		return events, nil
	}

	// Handle single CloudWatch Log event
	if cwLogs.AWSLogs.Data != "" {
		decodedData, err := decodeCloudWatchData(cwLogs.AWSLogs.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CloudWatch data: %w", err)
		}

		// If extractLogEvents is enabled, parse individual log events
		if p.extractLogEvents {
			var cwData CloudWatchLogsData
			if err := json.Unmarshal(decodedData, &cwData); err == nil && len(cwData.LogEvents) > 0 {
				for _, logEvent := range cwData.LogEvents {
					eventData, _ := json.Marshal(logEvent)
					events = append(events, &models.CloudEvent{
						ProviderType: "aws",
						Timestamp:    logEvent.Timestamp,
						LogGroup:     cwData.LogGroup,
						LogStream:    cwData.LogStream,
						Message:      logEvent.Message,
						RawData:      eventData,
					})
				}
				return events, nil
			}
		}

		events = append(events, &models.CloudEvent{
			ProviderType: "aws",
			RawData:      decodedData,
		})
	}

	return events, nil
}

func decodeCloudWatchData(data string) ([]byte, error) {
	// Decode base64
	base64Decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decompress gzip
	gz, err := gzip.NewReader(bytes.NewReader(base64Decoded))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip: %w", err)
	}

	return decompressed, nil
}
