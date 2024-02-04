# Splunk HEC tools for the cloud

This repository has a few handy tools for pushing events (mostly CloudWatch logs) to Splunk via different methods. 

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

in case of a failure, the function will retry to send the logs to the endpoint.
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


## Build as a lambda function

```bash
CGO_ENABLED=0 go build -ldflags='-s -w' -buildmode=pie -tags lambda
```

Note that Batching and load balancing is not supported in lambda mode due to the nature of lambda triggers. the recommended approach is also to not use the builtin DLQ and use
another Lambda function to deal with the failed attempts