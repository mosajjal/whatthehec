package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/mosajjal/whatthehec/pkg/hec"
	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/provider/aws"
	"github.com/mosajjal/whatthehec/pkg/storage"
	s3storage "github.com/mosajjal/whatthehec/pkg/storage/s3"
)

var (
	hecClient   *hec.Client
	awsProvider *aws.Provider
	awsConfig   awssdk.Config
)

func init() {
	var err error

	// Load AWS config
	region := getEnv("AWS_REGION", "us-east-1")
	if getEnv("S3_ACCESS_KEY_ID", "") != "" && getEnv("S3_ACCESS_KEY_SECRET", "") != "" {
		awsConfig, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					getEnv("S3_ACCESS_KEY_ID", ""),
					getEnv("S3_ACCESS_KEY_SECRET", ""),
					"",
				),
			),
		)
	} else {
		awsConfig, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
		)
	}
	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}

	// Get HEC token (potentially from Secrets Manager)
	hecToken := getEnv("HEC_TOKEN", "")
	if strings.HasPrefix(hecToken, "arn:aws:secretsmanager:") {
		log.Println("Fetching HEC token from AWS Secrets Manager")
		secretMgr := secretsmanager.NewFromConfig(awsConfig)
		secret, err := secretMgr.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
			SecretId: awssdk.String(hecToken),
		})
		if err != nil {
			log.Fatalf("Failed to get secret from Secrets Manager: %v", err)
		}
		hecToken = *secret.SecretString
	}

	// Parse HEC endpoints
	endpoints := strings.Split(getEnv("HEC_ENDPOINTS", ""), ",")
	if len(endpoints) == 0 || endpoints[0] == "" {
		log.Fatal("HEC_ENDPOINTS is required")
	}

	// Configure HEC client
	hecConfig := hec.Config{
		Endpoints:        endpoints,
		TLSSkipVerify:    getEnvBool("HEC_TLS_SKIP_VERIFY", true),
		Proxy:            getEnv("HEC_PROXY", ""),
		Token:            hecToken,
		ChannelID:        getEnv("HEC_CHANNEL_ID", ""),
		Index:            getEnv("HEC_INDEX", "main"),
		Source:           getEnv("HEC_SOURCE", "aws-lambda"),
		SourceType:       getEnv("HEC_SOURCETYPE", "aws:cloudwatch"),
		Host:             getEnv("HEC_HOST", "lambda"),
		BatchSize:        1,
		BatchTimeout:     parseDuration(getEnv("HEC_BATCH_TIMEOUT", "2s")),
		BalanceStrategy:  getEnv("HEC_BALANCE", "roundrobin"),
		ExtractLogEvents: getEnvBool("HEC_EXTRACT_LOG_EVENTS", false),
	}

	// Setup storage backends
	var failureStorage, coldStorage storage.StorageBackend

	if s3URL := getEnv("S3_URL", ""); s3URL != "" {
		storageConfig := storage.StorageConfig{
			Provider: "s3",
			URL:      s3URL,
		}
		failureStorage, err = s3storage.NewStorage(storageConfig, awsConfig)
		if err != nil {
			log.Printf("Failed to setup failure storage: %v", err)
		}
	}

	if s3ColdURL := getEnv("S3_COLD_STORAGE_URL", ""); s3ColdURL != "" {
		storageConfig := storage.StorageConfig{
			Provider: "s3",
			URL:      s3ColdURL,
		}
		coldStorage, err = s3storage.NewStorage(storageConfig, awsConfig)
		if err != nil {
			log.Printf("Failed to setup cold storage: %v", err)
		}
	}

	// Create HEC client
	hecClient, err = hec.NewClient(hecConfig, failureStorage, coldStorage)
	if err != nil {
		log.Fatalf("Failed to create HEC client: %v", err)
	}

	// Create AWS provider
	awsProvider = aws.NewProvider(hecConfig.ExtractLogEvents).(*aws.Provider)

	log.Println("AWS Lambda handler initialized successfully")
}

func HandleRequest(ctx context.Context, event interface{}) (string, error) {
	// Parse events using AWS provider
	cloudEvents, err := awsProvider.ParseBatch(ctx, event)
	if err != nil {
		return "", err
	}

	// Convert to HEC events
	hecEvents := make([]*models.Event, 0, len(cloudEvents))
	for _, cloudEvent := range cloudEvents {
		hecEvents = append(hecEvents, &models.Event{
			Time:       time.Now(),
			Host:       getEnv("HEC_HOST", "lambda"),
			Source:     getEnv("HEC_SOURCE", "aws-lambda"),
			SourceType: getEnv("HEC_SOURCETYPE", "aws:cloudwatch"),
			Index:      getEnv("HEC_INDEX", "main"),
			Event:      string(cloudEvent.RawData),
		})
	}

	// Send to HEC
	if err := hecClient.SendEvents(ctx, hecEvents); err != nil {
		log.Printf("Failed to send events to HEC: %v", err)
		return "", err
	}

	log.Printf("Successfully processed %d events", len(hecEvents))
	return "OK", nil
}

func main() {
	lambda.Start(HandleRequest)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 2 * time.Second
	}
	return d
}
