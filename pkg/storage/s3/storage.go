package s3

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/storage"
)

// Storage implements S3 backend for storage
type Storage struct {
	config    storage.StorageConfig
	client    *s3.Client
	bucket    string
	keyPrefix string
}

// NewStorage creates a new S3 storage backend
func NewStorage(cfg storage.StorageConfig, awsCfg aws.Config) (*Storage, error) {
	client := s3.NewFromConfig(awsCfg)

	// Parse bucket and prefix from URL
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URL: %w", err)
	}

	bucket := ""
	keyPrefix := ""

	// Parse bucket from hostname or path
	if strings.Contains(u.Host, ".s3.") || strings.Contains(u.Host, ".s3-") {
		// Virtual-hosted-style URL: bucket.s3.region.amazonaws.com
		parts := strings.Split(u.Host, ".")
		if len(parts) > 0 {
			bucket = parts[0]
		}
		keyPrefix = strings.Trim(u.Path, "/")
	} else {
		// Path-style URL: s3.region.amazonaws.com/bucket
		pathParts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 2)
		if len(pathParts) > 0 {
			bucket = pathParts[0]
		}
		if len(pathParts) > 1 {
			keyPrefix = pathParts[1]
		}
	}

	if bucket == "" {
		return nil, fmt.Errorf("could not parse bucket name from URL: %s", cfg.URL)
	}

	return &Storage{
		config:    cfg,
		client:    client,
		bucket:    bucket,
		keyPrefix: keyPrefix,
	}, nil
}

// Store saves events to S3
func (s *Storage) Store(ctx context.Context, events []*models.Event) error {
	// Convert events to JSON and compress
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)

	for _, event := range events {
		var eventData []byte
		var err error

		// Handle different event types
		switch v := event.Event.(type) {
		case string:
			eventData = []byte(v)
		case []byte:
			eventData = v
		default:
			eventData, err = json.Marshal(v)
			if err != nil {
				log.Printf("Failed to marshal event: %v", err)
				continue
			}
		}

		if _, err := gz.Write(eventData); err != nil {
			log.Printf("Failed to write to gzip: %v", err)
		}
		gz.Write([]byte("\n")) // Add newline between events
	}
	gz.Close()

	// Generate key with timestamp and UUID
	now := time.Now()
	key := fmt.Sprintf("%s/%d/%02d/%02d/%02d/%s-%s.json.gz",
		s.keyPrefix,
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Format("2006-01-02T15:04:05.000Z"),
		uuid.New().String(),
	)

	// Upload to S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("Successfully stored %d events to S3: %s/%s", len(events), s.bucket, key)
	return nil
}

// Close cleans up resources
func (s *Storage) Close() error {
	return nil
}
