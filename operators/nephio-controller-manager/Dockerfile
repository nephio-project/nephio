# Build the manager binary
FROM golang:1.20 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src
# Copy the Go Modules manifests
COPY krm-functions/ ./krm-functions/
COPY controllers/pkg/ ./controllers/pkg/
COPY operators/nephio-controller-manager/ ./operators/nephio-controller-manager/
WORKDIR /go/src/operators/nephio-controller-manager

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o /usr/local/bin/manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
#FROM alpine:latest
WORKDIR /
COPY --from=builder /usr/local/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
