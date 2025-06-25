ARG BUILDER_IMAGE=golang:1.23.5
ARG BASE_IMAGE=quay.io/argoproj/argocd:v3.0.6

FROM --platform=$BUILDPLATFORM ${BUILDER_IMAGE} AS build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/kpt-render

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /usr/local/bin/kpt-render

FROM ${BASE_IMAGE}

COPY --from=build /usr/local/bin/kpt-render /usr/local/bin/kpt-render

WORKDIR /home/argocd/cmp-server/config
COPY kpt-render-plugin.yaml ./plugin.yaml
