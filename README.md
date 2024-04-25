# Cloud Splunk HEC Tools Overview

This repository offers tools designed to facilitate the forwarding of events, with a focus on CloudWatch logs, to Splunk using various methods. It specializes in converting CloudWatch logs from their native base64 format and dispatching them to designated Splunk HTTP Event Collector (HEC) endpoints.

## Configuration via Environment Variables

The utility leverages environment variables for configuration, enabling straightforward setup and adaptability. The configurable options include:

- **AWS_REGION**: The AWS region for the IAM role and authentication.
- **HEC_ENDPOINTS**: A comma-separated list of Splunk HEC endpoints (e.g., `https://localhost:8088,https://localhost:8089`).
- **HEC_TLS_VERIFY**: Determines whether to verify the TLS certificate for HEC endpoints, with possible values of `true` or `false`.
- **HEC_PROXY**: The proxy server address for connecting to HEC endpoints (e.g., `socks5://username:password@localhost:1080`).
- **HEC_TOKEN**: Authentication token for HEC. if the token's value starts with `arn:aws:secretsmanager:`, it'll be considered to be a secret ARN and will be fetched from AWS Secrets. For authentication, lambda's role has priority, then S3_ACCESS_KEY_ID and S3_ACCESS_KEY_SECRET will be used. 
- **HEC_INDEX**: Target index for the logs.
- **HEC_SOURCE**: Source identifier for the logs.
- **HEC_SOURCETYPE**: Defines the type of source for the logs.
- **HEC_BATCH_SIZE** and **HEC_BATCH_TIMEOUT**: Configurations for batching logs.
- **HEC_BALANCE**: Load balancing strategy among `roundrobin`, `random`, or `sticky`.
- **HEC_STICKY_TTL**: Time-to-live for sticky sessions.

In case of delivery failure, the tool retries sending logs to the HEC endpoint. If persistent failures occur, it can redirect logs to an S3 bucket for storage, with additional options for cold storage. The related S3 settings include:

- **S3_URL**: URI for primary S3 storage in case of failure (e.g., `https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/`).
- **S3_ACCESS_KEY_ID** and **S3_ACCESS_KEY_SECRET**: Authentication credentials for S3. These can be left blank if assuming an AWS role.
- **S3_COLD_STORAGE_URL**: URI for S3 cold storage (e.g., `https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/`).
- **S3_COLD_STORAGE_ACCESS_KEY_ID** and **S3_COLD_STORAGE_ACCESS_KEY_SECRET**: Authentication credentials for S3 cold storage, optional if assuming an AWS role.

## Lambda Function Deployment

The tools can be compiled as a Lambda function using the provided command line. Batching and load balancing are not supported in Lambda mode due to the nature of Lambda triggers. A separate Lambda function is recommended for handling failed attempts instead of relying on the built-in Dead Letter Queue (DLQ).

### Deployment in Lambda using the container runtime

You must create a private Elastic Container Registry (ECR) repository, build the Dockerfile from this repository, and push the image to ECR. For S3 interactions, using AWS roles for authentication is advised over static keys.

### Deployment in Lambda using the zip archive

open a terminal and run the following commands:

```bash
wget https://github.com/mosajjal/whatthehec/releases/latest/download/hec-lambda-arm64.zip
aws --profile spk lambda create-function --function-name myFunction \
          --runtime provided.al2023 --handler bootstrap \
          --architectures arm64 \
          --role REPLACE_WITH_AWS_IAM_ROLE \
          --zip-file fileb://hec-lambda-arm64.zip
```


## Managing Retention in S3

For retention policies on S3 buckets, management is conducted through the AWS console. A provided link guides on automating the deletion of old files from S3, ensuring efficient storage management.

[Blog post](https://lepczynski.it/en/aws_en/automatically-delete-old-files-from-aws-s3/)
