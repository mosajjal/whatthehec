package hec

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
	"github.com/mosajjal/whatthehec/pkg/models"
	"github.com/mosajjal/whatthehec/pkg/storage"
)

// Config holds HEC client configuration
type Config struct {
	Endpoints        []string
	TLSSkipVerify    bool
	Proxy            string
	Token            string
	ChannelID        string
	Index            string
	Source           string
	SourceType       string
	Host             string
	BatchSize        int
	BatchTimeout     time.Duration
	BalanceStrategy  string // first_available, sticky, random, roundrobin
	StickyTTL        time.Duration
	ExtractLogEvents bool
}

// Client manages HEC connections and event delivery
type Client struct {
	config          Config
	connections     []*connection
	failureStorage  storage.StorageBackend
	coldStorage     storage.StorageBackend
	balanceStrategy uint8
	count           int
}

const (
	FirstAvailable = 1
	Sticky         = 2
	Random         = 3
	RoundRobin     = 4
)

type connection struct {
	endpoint  string
	client    *splunk.Client
	isHealthy bool
}

// NewClient creates a new HEC client
func NewClient(cfg Config, failureStorage, coldStorage storage.StorageBackend) (*Client, error) {
	client := &Client{
		config:         cfg,
		connections:    make([]*connection, 0),
		failureStorage: failureStorage,
		coldStorage:    coldStorage,
	}

	// Parse balance strategy
	switch cfg.BalanceStrategy {
	case "first_available":
		client.balanceStrategy = FirstAvailable
	case "sticky":
		client.balanceStrategy = Sticky
	case "random":
		client.balanceStrategy = Random
	case "roundrobin":
		client.balanceStrategy = RoundRobin
	default:
		log.Printf("Unknown load balance strategy: %v. Using first_available", cfg.BalanceStrategy)
		client.balanceStrategy = FirstAvailable
	}

	// Create connections for each endpoint
	for _, endpoint := range cfg.Endpoints {
		conn, err := newConnection(endpoint, cfg)
		if err != nil {
			log.Printf("Failed to create connection to %s: %v", endpoint, err)
			continue
		}
		client.connections = append(client.connections, conn)
		go conn.healthCheck()
	}

	if len(client.connections) == 0 {
		return nil, fmt.Errorf("no valid HEC endpoints configured")
	}

	return client, nil
}

func newConnection(endpoint string, cfg Config) (*connection, error) {
	rt := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.TLSSkipVerify},
	}
	httpClient := &http.Client{
		Timeout:   cfg.BatchTimeout,
		Transport: rt,
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		httpClient.Transport = &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.TLSSkipVerify},
		}
	}

	if !strings.HasSuffix(endpoint, "/services/collector") {
		endpoint = fmt.Sprintf("%s/services/collector", endpoint)
	}

	channelID := cfg.ChannelID
	if channelID == "" {
		channelID = uuid.New().String()
	} else {
		if _, err := uuid.Parse(channelID); err != nil {
			channelID = uuid.New().String()
		}
	}

	splunkClient := splunk.NewClient(
		httpClient,
		endpoint,
		cfg.Token,
		channelID,
		cfg.Source,
		cfg.SourceType,
		cfg.Index,
	)

	conn := &connection{
		endpoint: endpoint,
		client:   splunkClient,
	}
	conn.updateHealth()

	return conn, nil
}

func (c *connection) updateHealth() {
	c.isHealthy = c.client.CheckHealth() == nil
}

func (c *connection) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.updateHealth()
	}
}

// SendEvents sends events to HEC with fallback to storage
func (c *Client) SendEvents(ctx context.Context, events []*models.Event) error {
	// Send to cold storage if configured
	if c.coldStorage != nil {
		if err := c.coldStorage.Store(ctx, events); err != nil {
			log.Printf("Failed to send events to cold storage: %v", err)
		}
	}

	// Convert to splunk events
	splunkEvents := make([]*splunk.Event, len(events))
	for i, event := range events {
		splunkEvents[i] = &splunk.Event{
			Time:       splunk.EventTime{Time: event.Time},
			Host:       event.Host,
			Source:     event.Source,
			SourceType: event.SourceType,
			Index:      event.Index,
			Event:      event.Event,
		}
	}

	// Get a healthy connection
	conn := c.getConnection()
	if conn == nil {
		log.Printf("No healthy HEC connection available, sending to failure storage")
		if c.failureStorage != nil {
			return c.failureStorage.Store(ctx, events)
		}
		return fmt.Errorf("no healthy connections and no failure storage configured")
	}

	// Send to HEC
	return conn.client.LogEvents(splunkEvents)
}

func (c *Client) getConnection() *connection {
	switch c.balanceStrategy {
	case FirstAvailable:
		return c.getFirstAvailable()
	case Sticky:
		return c.getSticky()
	case Random:
		return c.getRandom()
	case RoundRobin:
		return c.getRoundRobin()
	default:
		return c.getFirstAvailable()
	}
}

func (c *Client) getFirstAvailable() *connection {
	for _, conn := range c.connections {
		if conn.isHealthy {
			return conn
		}
	}
	return nil
}

func (c *Client) getSticky() *connection {
	if c.count >= len(c.connections) {
		c.count = 0
	}
	conn := c.connections[c.count]
	if conn.isHealthy {
		return conn
	}
	return nil
}

func (c *Client) getRandom() *connection {
	// TODO: implement random selection
	return c.getFirstAvailable()
}

func (c *Client) getRoundRobin() *connection {
	// TODO: implement round-robin
	return c.getFirstAvailable()
}

// Close closes all connections
func (c *Client) Close() error {
	// Connections are closed automatically
	return nil
}
