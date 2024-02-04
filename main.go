/*
HEC serveless is an executable designed to be deployed as a lambda function.
It takes the input logs in base64 format (cloudwatch default), decodes them and sends them to a list of HEC endpoints.
the binary is configure only using environment variables.
configurable variables:
- HEC_ENDPOINTS: a comma separated list of HEC endpoints. Example: https://localhost:8088,https://localhost:8089
- HEC_TLS_VERIFY: whether to verify the TLS certificate of the HEC endpoints. Possible values: true, false
- HEC_PROXY: the proxy to use for all the endpoints. Example: socks5://username:password@localhost:1080
- HEC_TOKEN: the HEC token to use for all the endpoints
- HEC_INDEX: the index to use for all the endpoints
- HEC_SOURCE: the source to use for all the endpoints
- HEC_SOURCETYPE: the sourcetype to use for all the endpoints
- HEC_BATCH_SIZE: the batch size to use for all the endpoints
- HEC_BATCH_TIMEOUT: the batch timeout to use for all the endpoints
- HEC_BALANCE: the load balancing strategy to use. Possible values: roundrobin, random, sticky
- HEC_STICKY_TTL: the sticky ttl to use for all the endpoints

in case of a failure, the function will retry to send the logs to the same endpoint for 3 times.
after that, it will try to send the logs to an optional S3 bucket.
configurable variables:
- S3_DOMAIN: the S3 domain to use for storing the logs in case of a failure. Example: s3.eu-west-1.amazonaws.com
- S3_BUCKET: the S3 bucket to use for storing the logs in case of a failure
- S3_KEY_PREFIX: the S3 key prefix to use for storing the logs in case of a failure

there is also an ability to send the log to another S3 bucket, mainly for cold-storage purposes.
configurable variables:
- S3_COLD_STORAGE_BUCKET: the S3 bucket to use for storing the logs in case of a failure
- S3_COLD_STORAGE_KEY_PREFIX: the S3 key prefix to use for storing the logs in case of a failure

for managing retention on either S3 bucket, use the AWS console:
https://lepczynski.it/en/aws_en/automatically-delete-old-files-from-aws-s3/
*/

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	"github.com/alexflint/go-arg"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

var args struct {
	Endpoints                    []string      `arg:"env:HEC_ENDPOINTS,required"`
	TLSSkipVerify                bool          `arg:"env:HEC_TLS_SKIP_VERIFY" default:"true"`
	Proxy                        string        `arg:"env:HEC_PROXY"`
	Token                        string        `arg:"env:HEC_TOKEN"`
	Index                        string        `arg:"env:HEC_INDEX" default:"main"`
	Source                       string        `arg:"env:HEC_SOURCE" default:"hec_lambda"`
	Sourcetype                   string        `arg:"env:HEC_SOURCETYPE" default:"hec_lambda"`
	BatchSize                    int           `arg:"env:HEC_BATCH_SIZE" default:"1"`
	BatchTimeout                 time.Duration `arg:"env:HEC_BATCH_TIMEOUT" default:"2s"`
	Balance                      string        `arg:"env:HEC_BALANCE" default:"roundrobin"`
	StickyTTL                    time.Duration `arg:"env:HEC_STICKY_TTL" default:"5m"`
	S3URL                        string        `arg:"env:S3_URL" help:"example: https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/"`
	S3AccessKeyID                string        `arg:"env:S3_ACCESS_KEY_ID"`
	S3AccessKeySecret            string        `arg:"env:S3_ACCESS_KEY_SECRET"`
	S3ColdStorageURL             string        `arg:"env:S3_COLD_STORAGE_URL" help:"example: https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/"`
	S3ColdStorageAccessKeyID     string        `arg:"env:S3_COLD_STORAGE_ACCESS_KEY_ID"`
	S3ColdStorageAccessKeySecret string        `arg:"env:S3_COLD_STORAGE_ACCESS_KEY_SECRET"`
}

const (
	FirstAvailable = 1
	Sticky         = 2
	Random         = 3
	RoundRobin     = 4
)

type HECRuntime struct {
	Conns     []*HECConn
	Balance   uint8 // 1: first available, 2: sticky, 3: random, 4: roundrobin
	StickyTTL time.Duration
	BatchSize int
	count     int
	FailureS3 *S3
	ColdS3    *S3
	Events    chan string
	Done      chan struct{}
}

var hecRuntime *HECRuntime

type HECConfig struct {
	endpoint     string
	tlsVerify    bool
	proxy        string
	token        string
	index        string
	source       string
	sourcetype   string
	batchTimeout time.Duration
}

type HECConn struct {
	HECConfig
	IsHealthy bool
	Client    *splunk.Client
}

func (h *HECConn) UpdateHealthStatus() {
	h.IsHealthy = h.Client.CheckHealth() == nil
}

func (h *HECConn) SendEvent(events ...*splunk.Event) error {
	for _, event := range events {
		event.Index = h.index
		event.Source = h.source
		event.SourceType = h.sourcetype
	}
	return h.Client.LogEvents(events)
}

// Start is meant to be run as a goroutine and will update the health status of the HEC connection
func (h *HECConn) Start() {
	for {
		h.UpdateHealthStatus()
		time.Sleep(10 * time.Second)
	}
}

func NewHEC(conf HECConfig) *HECConn {
	rt := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.tlsVerify}}
	httpClient := &http.Client{Timeout: conf.batchTimeout, Transport: rt}
	if conf.proxy != "" {
		proxyURL, err := url.Parse(conf.proxy)
		if err != nil {
			panic(err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	if !strings.HasSuffix(conf.endpoint, "/services/collector") {
		conf.endpoint = fmt.Sprintf("%s/services/collector", conf.endpoint)
	}

	client := splunk.NewClient(httpClient, conf.endpoint, conf.token, conf.source, conf.sourcetype, conf.index)
	conn := &HECConn{conf, false, client}
	conn.UpdateHealthStatus()
	return conn
}

func (hec *HECRuntime) GetFirstAvailable() *HECConn {
	for _, conn := range hec.Conns {
		if conn.IsHealthy {
			return conn
		}
	}
	return nil
}
func (hec *HECRuntime) GetSticky() *HECConn {
	if hec.count >= len(hec.Conns) {
		hec.count = 0
	}
	conn := hec.Conns[hec.count]
	if conn.IsHealthy {
		return conn
	}
	return nil
}

func (hec *HECRuntime) GetRandom() *HECConn {
	// TODO: implement
	return hec.Conns[0]
}

func (hec *HECRuntime) GetRoundRobin() *HECConn {
	// TODO: implement
	return hec.Conns[0]
}

func (hec *HECRuntime) GetConn() *HECConn {
	switch hec.Balance {
	case FirstAvailable:
		return hec.GetFirstAvailable()
	case Sticky:
		return hec.GetSticky()
	case Random:
		return hec.GetRandom()
	case RoundRobin:
		return hec.GetRoundRobin()
	default:
		return hec.GetFirstAvailable()
	}
}

func (hec *HECRuntime) SendEvents(events ...*splunk.Event) error {
	// send to cold storage
	if hec.ColdS3 != nil {
		err := hec.ColdS3.Send(events...)
		if err != nil {
			log.Printf("error sending events to cold storage: %v", err)
		}
	}

	conn := hec.GetConn()
	if conn == nil {
		log.Printf("no healthy connection available.. sending events to S3")
		if hec.FailureS3 != nil {
			return hec.FailureS3.Send(events...)
		}
	}
	return conn.SendEvent(events...)
}

type S3 struct {
	URL             string
	AccessKeyID     string
	AccessKeySecret string
}

func (s3Bucket *S3) Send(events ...*splunk.Event) error {
	// converts events back to JSON and send them to the configured S3 bucket

	// parse the S3 URL to get the bucket name and the folder
	u, err := url.Parse(s3Bucket.URL)
	if err != nil {
		log.Printf("error parsing S3 URL: %v", err)
		return err
	}
	AmazonS3URL := ParseAmazonS3URL(u)
	// s3Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
	// 	return aws.Endpoint{
	// 		URL: fmt.Sprintf("%s://%s", u.Scheme, strings.TrimPrefix(u.Host, AmazonS3URL.Bucket+".")),
	// 	}, nil
	// })

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		// config.WithRegion(AmazonS3URL.Region),
		config.WithRegion("ap-southeast-2"),
		// config.WithEndpointResolverWithOptions(s3Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3Bucket.AccessKeyID, s3Bucket.AccessKeySecret, "")),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg)

	// convert the events back to JSON as IO
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	for _, event := range events {
		// b, err := json.Marshal(event.Event)
		// if err != nil {
		// 	log.Printf("Couldn't marshal event to JSON. Here's why: %v\n", err)
		// 	continue
		// }
		if _, err := gz.Write([]byte(event.Event.(string))); err != nil {
			log.Printf("Couldn't gzip event. Here's why: %v\n", err)
		}
	}
	gz.Close()

	// filename is path/year/month/day/hour/timestamp-uuid.json.gz
	now := time.Now()
	filename := fmt.Sprintf("%s/%d/%d/%d/%d/%v-%s.json.gz", strings.Trim(u.Path, "/"), now.Year(), now.Month(), now.Day(), now.Hour(), now.Format("2006-01-02T15:04:05.000Z"), uuid.New().String())

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(AmazonS3URL.Bucket),
		Key:    aws.String(filename),
		Body:   bytes.NewReader(buf.Bytes()),
		// ContentLength: int64(buf.Len()),
	})
	if err != nil {
		log.Printf("Couldn't upload events to %v. Here's why: %v\n", AmazonS3URL.Bucket, err)
	}

	return nil
}

func init() {
	arg.MustParse(&args)

	// set up each HEC connection
	var hecConns []*HECConn
	for _, hec := range args.Endpoints {
		hecConfig := HECConfig{
			endpoint:     hec,
			tlsVerify:    args.TLSSkipVerify,
			proxy:        args.Proxy,
			token:        args.Token,
			index:        args.Index,
			source:       args.Source,
			sourcetype:   args.Sourcetype,
			batchTimeout: args.BatchTimeout,
		}
		hecConns = append(hecConns, NewHEC(hecConfig))
	}
	// set up the S3 buckets
	var coldS3 *S3
	if args.S3ColdStorageURL != "" {
		coldS3 = &S3{
			URL:             args.S3ColdStorageURL,
			AccessKeyID:     args.S3ColdStorageAccessKeyID,
			AccessKeySecret: args.S3ColdStorageAccessKeySecret,
		}
	} else {
		log.Printf("No cold storage S3 URL is provided. If you want to send events to S3 for cold storage, please provide a S3 URL")
	}
	var failureS3 *S3
	if args.S3URL != "" {
		failureS3 = &S3{
			URL:             args.S3URL,
			AccessKeyID:     args.S3AccessKeyID,
			AccessKeySecret: args.S3AccessKeySecret,
		}
	} else {
		log.Printf("No S3 URL is provided. If you want to send events to S3 in case of failure, please provide a S3 URL")
	}

	// translate load balance strategy
	var balanceStrategy uint8
	switch args.Balance {
	case "first_available":
		balanceStrategy = FirstAvailable
	case "sticky":
		balanceStrategy = Sticky
	case "random":
		balanceStrategy = Random
	case "roundrobin":
		balanceStrategy = RoundRobin
	default:
		log.Printf("Unknown load balance strategy: %v. Using first_available", args.Balance)
		balanceStrategy = FirstAvailable
	}

	// set up the HEC runtime
	hecRuntime = &HECRuntime{
		Conns:     hecConns,
		Balance:   balanceStrategy,
		StickyTTL: args.StickyTTL,
		FailureS3: failureS3,
		ColdS3:    coldS3,
	}
	hecRuntime.Events = make(chan string)
	// start the runtime
	go hecRuntime.Start()
}

func main() {
	lambda.Start(HandleRequest)
}

// CloudwatchLogs is the event structure for Cloudwatch Logs. the Data is base64 encoded
type CloudwatchLogs struct {
	// Cloudwatch Logs look like this and the Data is base64 and gzip. a custom marshaller is built to turn it into raw json
	AWSLogs struct {
		Data string `json:"data"`
	} `json:"awslogs"`
	// Kinesis cloudwatch logs processor looks like this and the Data is a base64 and gzip
	Records []struct {
		RecordID string `json:"recordId"`
		Data     string `json:"data"`
	} `json:"records"`
	raw []byte `json:"-"` // raw JSON
}

func (s CloudwatchLogs) MarshalJSON() ([]byte, error) {
	if s.raw == nil {
		if s.AWSLogs.Data != "" {
			base64DecodedGZ, err := base64.StdEncoding.DecodeString(s.AWSLogs.Data)
			if err != nil {
				fmt.Printf("Couldn't decode base64 data: %v", err)
				return nil, err
			}
			gz, err := gzip.NewReader(bytes.NewReader(base64DecodedGZ))
			if err != nil {
				fmt.Printf("Couldn't decompress gzipped data: %v", err)
				return nil, err
			}
			base64Decoded, err := io.ReadAll(gz)
			return base64Decoded, err
		}
		if len(s.Records) > 0 {
			out := []byte{}
			// create a JSON array because it's harder to parse otherwise
			out = append(out[:], '[')
			for _, record := range s.Records {
				base64DecodedGZ, err := base64.StdEncoding.DecodeString(record.Data)
				if err != nil {
					fmt.Printf("Couldn't decode base64 data: %v", err)
					return nil, err
				}
				gz, err := gzip.NewReader(bytes.NewReader(base64DecodedGZ))
				if err != nil {
					fmt.Printf("Couldn't decompress gzipped data: %v", err)
					return nil, err
				}
				if base64Decoded, err := io.ReadAll(gz); err == nil {
					out = append(out[:], base64Decoded[:]...)
					out = append(out[:], ',')
				}
			}
			// remove the last , and replace it with ]
			out = out[:len(out)-1]
			out = append(out[:], ']')
			return out, nil
		}
	}
	return s.raw, nil
}

func (s *CloudwatchLogs) UnmarshalJSON(data []byte) error {
	// try to unmarshal JSON
	type Alias CloudwatchLogs
	var alias Alias

	if err := json.Unmarshal(data, &alias); err == nil {
		*s = CloudwatchLogs(alias)
	}
	if alias.AWSLogs.Data == "" && len(alias.Records) == 0 {
		s.raw = data
	}

	return nil
}

func HandleRequest(ctx context.Context, event CloudwatchLogs) (string, error) {
	s, err := event.MarshalJSON()
	if err != nil {
		return "", err
	}
	hecRuntime.SendSingleEvent(string(s))
	// hecRuntime.Events <- string(s)
	// time.Sleep(time.Second * 2)
	// <-hecRuntime.Done
	return "OK", nil
}

func (h *HECRuntime) SendSingleEvent(event string) {
	eventBatch := make([]*splunk.Event, h.BatchSize)
	eventBatch = append(eventBatch, &splunk.Event{
		Time:  splunk.EventTime{Time: time.Now()},
		Host:  "lambda",
		Event: event,
	})
	err := h.SendEvents(eventBatch...)
	if err != nil {
		log.Printf("Couldn't send events to HEC. Here's why: %v\n", err)
		h.FailureS3.Send(eventBatch...)
	}
}

func (h *HECRuntime) Start() {
	// start the ticker
	ticker := time.NewTicker(time.Second * h.StickyTTL)
	defer ticker.Stop()

	eventBatch := make([]*splunk.Event, h.BatchSize)

	// start the ticker
	for {
		select {
		case <-ticker.C:
			// TODO: implement sticky TTL
		case event := <-h.Events:
			// batch them up and send them to the HEC
			eventBatch = append(eventBatch, &splunk.Event{
				Time:  splunk.EventTime{Time: time.Now()},
				Host:  "lambda",
				Event: event,
			})
			// if len(eventBatch) == h.BatchSize {
			err := h.SendEvents(eventBatch...)
			if err != nil {
				log.Printf("Couldn't send events to HEC. Here's why: %v\n", err)
			}
			// }
			eventBatch = make([]*splunk.Event, h.BatchSize)
		}
	}
}
