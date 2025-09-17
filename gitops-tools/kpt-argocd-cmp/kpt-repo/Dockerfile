# Build
ARG BUILDER_IMAGE=golang:1.23.8@sha256:ec5612bbd9e96d5b80a8b968cea06a4a9b985fe200ff6da784bf607063273c59
ARG BASE_IMAGE=quay.io/argoproj/argocd:v3.0.6

FROM --platform=$BUILDPLATFORM ${BUILDER_IMAGE} AS build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/kpt-repo

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

ARG TARGETOS
ARG TARGETARCH
RUN GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /usr/local/bin/kpt-repo

# Final Image
FROM ${BASE_IMAGE}

COPY --from=build /usr/local/bin/kpt-repo /usr/local/bin/kpt-repo

WORKDIR /home/argocd/cmp-server/config
COPY kpt-repo-plugin.yaml ./plugin.yaml
