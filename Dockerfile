# DEPRECATED: This Dockerfile is deprecated and will be removed in v2.0.0
# Please use the provider-specific Dockerfiles:
#   - Dockerfile.aws   for AWS Lambda
#   - Dockerfile.azure for Azure Functions
#   - Dockerfile.gcp   for GCP Cloud Functions
# See MIGRATION.md for migration instructions.

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23.4-alpine3.21
LABEL maintainer="Ali Mosajjal <hi@n0p.me>"


ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates
RUN mkdir /app
ADD . /app/
WORKDIR /app/
ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOFLAGS=-buildvcs=false go build -ldflags "-s -w" -o lambda -tags lambda .
CMD ["/app/lambda"]

FROM alpine:edge
RUN apk add --no-cache ca-certificates
COPY --from=0 /app/lambda /lambda
ENTRYPOINT ["/lambda"]

