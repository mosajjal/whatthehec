package storage

import (
	"testing"
)

func TestStorageConfig(t *testing.T) {
	cfg := StorageConfig{
		Provider:        "s3",
		URL:             "https://bucket.s3.us-east-1.amazonaws.com/prefix/",
		AccessKey:       "test-key",
		SecretKey:       "test-secret",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		PathPrefix:      "logs/",
		CompressionType: "gzip",
	}

	if cfg.Provider != "s3" {
		t.Errorf("Expected provider to be 's3', got '%s'", cfg.Provider)
	}
	if cfg.CompressionType != "gzip" {
		t.Errorf("Expected compression type to be 'gzip', got '%s'", cfg.CompressionType)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Expected region to be 'us-east-1', got '%s'", cfg.Region)
	}
}

func TestStorageConfig_Providers(t *testing.T) {
	providers := []string{"s3", "azure-blob", "gcs"}
	
	for _, provider := range providers {
		cfg := StorageConfig{
			Provider: provider,
		}
		
		if cfg.Provider != provider {
			t.Errorf("Expected provider to be '%s', got '%s'", provider, cfg.Provider)
		}
	}
}
