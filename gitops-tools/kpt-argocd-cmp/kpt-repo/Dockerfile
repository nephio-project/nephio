# Build
ARG BUILDER_IMAGE=golang:1.23.5
ARG BASE_IMAGE=quay.io/argoproj/argocd:v3.0.9

FROM --platform=$BUILDPLATFORM ${BUILDER_IMAGE} AS build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/kpt-repo

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /usr/local/bin/kpt-repo

# Final Image
FROM ${BASE_IMAGE}

COPY --from=build /usr/local/bin/kpt-repo /usr/local/bin/kpt-repo

WORKDIR /home/argocd/cmp-server/config
COPY kpt-repo-plugin.yaml ./plugin.yaml
