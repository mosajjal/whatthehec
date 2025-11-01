package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mosajjal/whatthehec/pkg/hec"
	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/provider/azure"
)

var (
	hecClient     *hec.Client
	azureProvider *azure.Provider
)

func init() {
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
		Token:            getEnv("HEC_TOKEN", ""),
		ChannelID:        getEnv("HEC_CHANNEL_ID", ""),
		Index:            getEnv("HEC_INDEX", "main"),
		Source:           getEnv("HEC_SOURCE", "azure-function"),
		SourceType:       getEnv("HEC_SOURCETYPE", "azure:monitor"),
		Host:             getEnv("HEC_HOST", "azure-function"),
		BatchSize:        1,
		BatchTimeout:     parseDuration(getEnv("HEC_BATCH_TIMEOUT", "2s")),
		BalanceStrategy:  getEnv("HEC_BALANCE", "roundrobin"),
		ExtractLogEvents: getEnvBool("HEC_EXTRACT_LOG_EVENTS", false),
	}

	var err error
	hecClient, err = hec.NewClient(hecConfig, nil, nil)
	if err != nil {
		log.Fatalf("Failed to create HEC client: %v", err)
	}

	azureProvider = azure.NewProvider(hecConfig.ExtractLogEvents).(*azure.Provider)
	log.Println("Azure Function handler initialized successfully")
}

// HandleRequest processes Azure Monitor events
func HandleRequest(ctx context.Context, event interface{}) (string, error) {
	cloudEvents, err := azureProvider.ParseBatch(ctx, event)
	if err != nil {
		return "", err
	}

	hecEvents := make([]*models.Event, 0, len(cloudEvents))
	for _, cloudEvent := range cloudEvents {
		hecEvents = append(hecEvents, &models.Event{
			Time:       time.Now(),
			Host:       getEnv("HEC_HOST", "azure-function"),
			Source:     getEnv("HEC_SOURCE", "azure-function"),
			SourceType: getEnv("HEC_SOURCETYPE", "azure:monitor"),
			Index:      getEnv("HEC_INDEX", "main"),
			Event:      string(cloudEvent.RawData),
		})
	}

	if err := hecClient.SendEvents(ctx, hecEvents); err != nil {
		log.Printf("Failed to send events to HEC: %v", err)
		return "", err
	}

	log.Printf("Successfully processed %d events", len(hecEvents))
	return "OK", nil
}

func main() {
	// Azure Functions runtime will call HandleRequest
	log.Println("Azure Function for Splunk HEC ready")
}

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
