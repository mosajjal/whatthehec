FROM --platform=${BUILDPLATFORM:-linux/amd64} alpine:1.21.6-alpine3.19
RUN apk add --no-cache ca-certificates
COPY  ./hec-serverless /hec-serverless
ENTRYPOINT ["/hec-serverless"]
