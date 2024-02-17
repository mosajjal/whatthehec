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
- S3_URL: the S3 URI to use for storing the logs in case of a failure. Example: `https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/`
- S3_ACCESS_KEY_ID: the S3 access key id to authenticate. leave empty if you're assuming a role
- S3_ACCESS_KEY_SECRET: the S3 access key secret to authenticate. leave empty if you're assuming a role

there is also the ability to send the log to another S3 bucket, mainly for cold-storage purposes.
configurable variables:
- S3_COLD_STORAGE_URL: the S3 URI to use for storing the logs as cold storage Example: `https://YOURBUCKET.s3.ap-southeast-2.amazonaws.com/YOURFOLDER/`
- S3_COLD_STORAGE_ACCESS_KEY_ID: the S3 access key id to authenticate. leave empty if you're assuming a role
- S3_COLD_STORAGE_ACCESS_KEY_SECRET: the S3 access key secret to authenticate. leave empty if you're assuming a role

for managing retention on either S3 bucket, use the AWS console:
https://lepczynski.it/en/aws_en/automatically-delete-old-files-from-aws-s3/


## Build as a lambda function

```bash
CGO_ENABLED=0 go build -ldflags='-s -w' -buildmode=pie -tags lambda
```

Note that Batching and load balancing is not supported in lambda mode due to the nature of lambda triggers. the recommended approach is also to not use the builtin DLQ and use
another Lambda function to deal with the failed attempts

the lambda function only works with the container runtime. Due to the current limitation of lambda not being able to directly use public registries, you need to create a private ECR repository, build this repositories' Dockerfile and push it. 

If you intend to use AWS roles rather than static key for authenticating to s3, you must keep both 