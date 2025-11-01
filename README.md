# Multi-Cloud Splunk HEC Forwarder

A cloud-native, multi-cloud log forwarding solution that sends logs from AWS, Azure, and GCP to Splunk HTTP Event Collector (HEC) endpoints.

## 🌟 Features

- **Multi-Cloud Support**: Native integrations for AWS Lambda, Azure Functions, and GCP Cloud Functions
- **Provider-Agnostic Architecture**: Clean separation between cloud providers and HEC delivery
- **Efficient Batching**: Configurable batching and load balancing across HEC endpoints
- **Failure Handling**: Automatic fallback to cloud storage (S3, Azure Blob, GCS) on delivery failures
- **Cold Storage**: Optional archival to long-term storage for compliance
- **Health Monitoring**: Automatic health checks for HEC endpoints
- **TLS Support**: Configurable TLS verification for secure connections
- **Secret Management**: Integration with cloud-native secret managers

## 📦 Architecture

### Directory Structure

```
whatthehec/
├── cmd/
│   ├── aws-lambda/       # AWS Lambda function entry point
│   ├── azure-function/   # Azure Functions entry point
│   └── gcp-function/     # GCP Cloud Functions entry point
├── pkg/
│   ├── models/           # Common data models
│   ├── hec/              # HEC client implementation
│   ├── provider/         # Cloud provider interfaces
│   │   ├── aws/         # AWS CloudWatch Logs parser
│   │   ├── azure/       # Azure Monitor parser
│   │   └── gcp/         # GCP Cloud Logging parser
│   └── storage/         # Storage backend interfaces
│       ├── s3/          # AWS S3 storage
│       ├── azure/       # Azure Blob storage (TODO)
│       └── gcs/         # GCP Cloud Storage (TODO)
├── Dockerfile.aws        # AWS Lambda container
├── Dockerfile.azure      # Azure Functions container
└── Dockerfile.gcp        # GCP Cloud Functions container
```

### Design Principles

1. **Separation of Concerns**: Cloud-specific logic isolated in provider packages
2. **Dependency Injection**: Components configured via constructors for testability
3. **Interface-Driven**: All major components implement interfaces for flexibility
4. **Best Practices**: Following cloud architecture frameworks and 12-factor app principles

## 🚀 Deployment

### AWS Lambda

#### Option 1: Container Image (Recommended)

```bash
# Build and push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account>.dkr.ecr.us-east-1.amazonaws.com
docker build -f Dockerfile.aws -t whatthehec-aws .
docker tag whatthehec-aws:latest <account>.dkr.ecr.us-east-1.amazonaws.com/whatthehec-aws:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/whatthehec-aws:latest

# Create Lambda function
aws lambda create-function \
  --function-name whatthehec-cloudwatch \
  --package-type Image \
  --code ImageUri=<account>.dkr.ecr.us-east-1.amazonaws.com/whatthehec-aws:latest \
  --role arn:aws:iam::<account>:role/lambda-execution-role \
  --timeout 60 \
  --memory-size 256
```

#### Option 2: ZIP Archive

Download pre-built artifacts from [releases](https://github.com/mosajjal/whatthehec/releases):

```bash
wget https://github.com/mosajjal/whatthehec/releases/latest/download/hec-lambda-arm64.zip

aws lambda create-function \
  --function-name whatthehec-cloudwatch \
  --runtime provided.al2023 \
  --handler bootstrap \
  --architectures arm64 \
  --role arn:aws:iam::<account>:role/lambda-execution-role \
  --zip-file fileb://hec-lambda-arm64.zip
```

### Azure Functions

```bash
# Build container
docker build -f Dockerfile.azure -t whatthehec-azure .

# Push to Azure Container Registry
az acr login --name <registry>
docker tag whatthehec-azure <registry>.azurecr.io/whatthehec-azure:latest
docker push <registry>.azurecr.io/whatthehec-azure:latest

# Deploy to Azure Functions
az functionapp create \
  --resource-group <resource-group> \
  --name whatthehec-monitor \
  --storage-account <storage> \
  --deployment-container-image-name <registry>.azurecr.io/whatthehec-azure:latest
```

### GCP Cloud Functions

```bash
# Build container
docker build -f Dockerfile.gcp -t whatthehec-gcp .

# Push to Google Container Registry
docker tag whatthehec-gcp gcr.io/<project>/whatthehec-gcp:latest
docker push gcr.io/<project>/whatthehec-gcp:latest

# Deploy to Cloud Functions
gcloud functions deploy whatthehec-logging \
  --gen2 \
  --runtime=go123 \
  --region=us-central1 \
  --source=./cmd/gcp-function \
  --entry-point=HandleRequest
```

## ⚙️ Configuration

All deployment options use environment variables for configuration:

### Core HEC Settings

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `HEC_ENDPOINTS` | Comma-separated list of HEC URLs | - | Yes |
| `HEC_TOKEN` | HEC authentication token or secret ARN (AWS) | - | Yes |
| `HEC_TLS_SKIP_VERIFY` | Skip TLS certificate verification | `true` | No |
| `HEC_INDEX` | Target Splunk index | `main` | No |
| `HEC_SOURCE` | Source identifier | `{provider}-function` | No |
| `HEC_SOURCETYPE` | Source type | `{provider}:logs` | No |
| `HEC_HOST` | Host field value | `function` | No |

### Advanced HEC Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `HEC_CHANNEL_ID` | HEC channel ID (UUID) | Auto-generated |
| `HEC_PROXY` | Proxy URL (e.g., `socks5://user:pass@host:port`) | - |
| `HEC_BATCH_TIMEOUT` | Batch timeout duration | `2s` |
| `HEC_BALANCE` | Load balancing: `first_available`, `sticky`, `random`, `roundrobin` | `roundrobin` |
| `HEC_EXTRACT_LOG_EVENTS` | Extract individual log events (AWS only) | `false` |

### Storage Backends (AWS)

| Variable | Description |
|----------|-------------|
| `S3_URL` | Failure storage S3 URL |
| `S3_ACCESS_KEY_ID` | S3 access key (optional if using IAM role) |
| `S3_ACCESS_KEY_SECRET` | S3 secret key (optional if using IAM role) |
| `S3_COLD_STORAGE_URL` | Cold storage S3 URL |
| `AWS_REGION` | AWS region | 

### Example Configuration

```bash
# AWS Lambda environment variables
HEC_ENDPOINTS=https://splunk1.example.com:8088,https://splunk2.example.com:8088
HEC_TOKEN=arn:aws:secretsmanager:us-east-1:123456789:secret:splunk-hec-token
HEC_INDEX=cloudwatch
HEC_SOURCE=aws-lambda
HEC_SOURCETYPE=aws:cloudwatch
HEC_BALANCE=roundrobin
S3_URL=https://mybucket.s3.us-east-1.amazonaws.com/failed-logs/
AWS_REGION=us-east-1
```

## 🏗️ Building from Source

### Prerequisites

- Go 1.23.4 or later
- Docker (for container builds)

### Build All Providers

```bash
# Get dependencies
go mod download

# Build AWS Lambda
go build -o aws-lambda ./cmd/aws-lambda

# Build Azure Functions
go build -o azure-function ./cmd/azure-function

# Build GCP Cloud Functions
go build -o gcp-function ./cmd/gcp-function

# Build all Docker images
docker build -f Dockerfile.aws -t whatthehec-aws .
docker build -f Dockerfile.azure -t whatthehec-azure .
docker build -f Dockerfile.gcp -t whatthehec-gcp .
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

## 📊 Artifacts

The project produces the following artifacts:

### Container Images (via GitHub Actions)

- `ghcr.io/mosajjal/whatthehec-aws:latest` - AWS Lambda container
- `ghcr.io/mosajjal/whatthehec-azure:latest` - Azure Functions container
- `ghcr.io/mosajjal/whatthehec-gcp:latest` - GCP Cloud Functions container

### Lambda ZIP Archives (via GitHub Releases)

- `hec-lambda-amd64.zip` - x86_64 Lambda function
- `hec-lambda-arm64.zip` - ARM64 Lambda function

## 🔒 Security Best Practices

1. **Use Secret Managers**: Store HEC tokens in AWS Secrets Manager, Azure Key Vault, or GCP Secret Manager
2. **Enable TLS Verification**: Set `HEC_TLS_SKIP_VERIFY=false` in production
3. **Use IAM Roles**: Prefer cloud-native authentication over static credentials
4. **Least Privilege**: Grant minimal required permissions to function execution roles
5. **Network Security**: Deploy functions in private subnets with VPC endpoints

### Required IAM Permissions (AWS)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::your-bucket/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:region:account:secret:your-secret"
    }
  ]
}
```

## 🎯 Performance Tuning

### Memory and Timeout

- **AWS Lambda**: 256-512 MB recommended, 60s timeout
- **Azure Functions**: Premium plan for production workloads
- **GCP**: 2nd generation functions with 512 MB memory

### Batch Processing

For high-volume scenarios, consider:
- Kinesis Data Firehose (AWS) with batching
- Event Hubs (Azure) with batch processing
- Pub/Sub (GCP) with subscription batching

### HEC Endpoint Optimization

- Use multiple HEC endpoints with `roundrobin` load balancing
- Deploy HEC indexers close to function regions
- Enable HEC acknowledgments for delivery guarantees

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📝 License

Apache License 2.0 - See [LICENSE](LICENSE) for details

## 👤 Maintainer

Ali Mosajjal <hi@n0p.me>

## 🔗 Links

- [GitHub Repository](https://github.com/mosajjal/whatthehec)
- [Issue Tracker](https://github.com/mosajjal/whatthehec/issues)
- [Releases](https://github.com/mosajjal/whatthehec/releases)
- [Splunk HEC Documentation](https://docs.splunk.com/Documentation/Splunk/latest/Data/UsetheHTTPEventCollector)
