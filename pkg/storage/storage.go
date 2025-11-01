package storage

import (
	"context"
	"github.com/mosajjal/whatthehec/pkg/models"
)

// StorageBackend defines the interface for fallback storage
type StorageBackend interface {
	// Store saves events to storage when HEC delivery fails
	Store(ctx context.Context, events []*models.Event) error
	
	// Close cleans up resources
	Close() error
}

// StorageConfig holds common storage configuration
type StorageConfig struct {
	Provider        string // s3, azure-blob, gcs
	URL             string
	AccessKey       string
	SecretKey       string
	Region          string
	Bucket          string
	PathPrefix      string
	CompressionType string // gzip, none
}
