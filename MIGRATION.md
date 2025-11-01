# Migration Guide: Legacy to Multi-Cloud Architecture

## Overview

This guide helps you migrate from the legacy single-file AWS Lambda implementation to the new multi-cloud architecture.

## What Changed?

### Architecture

**Before:**
- Single monolithic file (`main.go`, `lambda.go`)
- AWS-only support
- Tightly coupled components
- Build tags required (`-tags lambda`)

**After:**
- Modular package structure
- Multi-cloud support (AWS, Azure, GCP)
- Clean separation of concerns
- Provider-specific binaries

### Directory Structure

```
Old:
├── main.go          (HEC runtime + AWS code)
├── lambda.go        (Lambda handler)
├── awsuri.go        (S3 URL parsing)
└── Dockerfile       (AWS Lambda only)

New:
├── cmd/
│   ├── aws-lambda/      (AWS Lambda handler)
│   ├── azure-function/  (Azure Functions handler)
│   └── gcp-function/    (GCP Cloud Functions handler)
├── pkg/
│   ├── models/          (Data models)
│   ├── hec/             (HEC client)
│   ├── provider/        (Cloud provider interfaces)
│   │   ├── aws/
│   │   ├── azure/
│   │   └── gcp/
│   └── storage/         (Storage backends)
│       └── s3/
├── Dockerfile.aws       (AWS Lambda container)
├── Dockerfile.azure     (Azure Functions container)
└── Dockerfile.gcp       (GCP Cloud Functions container)
```

## Migration Steps

### For AWS Lambda Users

#### Option 1: Use Pre-built Artifacts (Recommended)

Download the latest release:
```bash
wget https://github.com/mosajjal/whatthehec/releases/latest/download/hec-lambda-arm64.zip
```

The configuration is **100% backward compatible**. All environment variables work the same way.

#### Option 2: Build from Source

```bash
# Old way (with build tags)
go build -tags lambda -o bootstrap .

# New way (no build tags needed)
go build -o bootstrap ./cmd/aws-lambda
```

#### Option 3: Use Container Image

```bash
# Old container
docker pull ghcr.io/mosajjal/whatthehec:latest

# New container (provider-specific)
docker pull ghcr.io/mosajjal/whatthehec-aws:latest
```

### Configuration Changes

✅ **Good News:** All environment variables remain the same!

No configuration changes are required. The new implementation maintains full backward compatibility:

- `HEC_ENDPOINTS` - Still works
- `HEC_TOKEN` - Still works (including Secrets Manager ARN support)
- `HEC_INDEX`, `HEC_SOURCE`, `HEC_SOURCETYPE` - All unchanged
- `S3_URL`, `S3_COLD_STORAGE_URL` - Same as before
- `HEC_EXTRACT_LOG_EVENTS` - Still supported

### Differences

#### Build Process

**Old:**
```bash
# Required build tag
GOOS=linux GOARCH=arm64 go build -tags lambda -o bootstrap .
```

**New:**
```bash
# No build tag needed
GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/aws-lambda
```

#### Import Paths (If You Forked)

If you forked the repository and modified the code:

**Old:**
```go
// Everything was in the main package
```

**New:**
```go
import (
    "github.com/mosajjal/whatthehec/pkg/hec"
    "github.com/mosajjal/whatthehec/pkg/provider/aws"
    "github.com/mosajjal/whatthehec/pkg/models"
)
```

## Testing Your Migration

### 1. Test Lambda Function

```bash
# Create test event
cat > test-event.json << 'EOF'
{
  "awslogs": {
    "data": "H4sIAAAAAAAAAHWPwQqCQBCGX0Xm7EFtK+smZBEUgXoLEreV..."
  }
}
EOF

# Test locally (if using SAM or similar)
sam local invoke -e test-event.json
```

### 2. Verify Environment Variables

Ensure all your environment variables are set correctly:

```bash
aws lambda get-function-configuration \
  --function-name your-function-name \
  --query 'Environment.Variables'
```

### 3. Check CloudWatch Logs

After deployment, verify logs are being sent to Splunk:

```bash
aws logs tail /aws/lambda/your-function-name --follow
```

## Rollback Plan

If you need to rollback:

### For Container Deployments

```bash
# Revert to previous image version
aws lambda update-function-code \
  --function-name your-function-name \
  --image-uri <previous-image-uri>
```

### For ZIP Deployments

Use the previous release:
```bash
wget https://github.com/mosajjal/whatthehec/releases/download/v1.0.0/hec-lambda-arm64.zip
```

## Legacy File Status

The following files are now **deprecated** but kept for reference:

- ❌ `main.go` - Replaced by `cmd/aws-lambda/main.go`
- ❌ `lambda.go` - Merged into `cmd/aws-lambda/main.go`
- ❌ `Dockerfile` - Replaced by `Dockerfile.aws`

These files will be removed in a future major version (v2.0.0).

## Benefits of Migration

1. **Multi-Cloud Ready**: Easy to deploy to Azure or GCP
2. **Better Testing**: Modular architecture allows comprehensive testing
3. **Maintainability**: Clear separation of concerns
4. **Efficiency**: Optimized for each cloud provider
5. **Best Practices**: Follows cloud-native design patterns

## New Features Available

### Azure Functions Support

```bash
# Deploy to Azure
docker pull ghcr.io/mosajjal/whatthehec-azure:latest
```

### GCP Cloud Functions Support

```bash
# Deploy to GCP
docker pull ghcr.io/mosajjal/whatthehec-gcp:latest
```

### Enhanced Testing

```bash
# Run comprehensive tests
go test ./...
```

## Getting Help

- 📖 [New README](README.md)
- 🐛 [Report Issues](https://github.com/mosajjal/whatthehec/issues)
- 💬 [Discussions](https://github.com/mosajjal/whatthehec/discussions)

## FAQ

**Q: Do I need to change my Lambda function configuration?**  
A: No, all environment variables remain the same.

**Q: Will my CloudWatch Log subscriptions still work?**  
A: Yes, the event format handling is unchanged.

**Q: Can I still use the old ZIP files?**  
A: For now, yes. But we recommend migrating to the new artifacts.

**Q: What about Secrets Manager integration?**  
A: Fully supported, same as before.

**Q: Do I need to rebuild my IaC (Terraform, CloudFormation)?**  
A: Only if you want to use the new container images. ZIP deployments work as-is.

## Timeline

- **Now**: Both old and new implementations work
- **v1.x**: Old files marked as deprecated
- **v2.0**: Old files removed (planned for Q2 2025)
