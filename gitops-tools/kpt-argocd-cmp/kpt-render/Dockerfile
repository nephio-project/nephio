ARG BUILDER_IMAGE=golang:1.24.2
ARG BASE_IMAGE=quay.io/argoproj/argocd:v2.14.6

FROM --platform=$BUILDPLATFORM $BUILDER_IMAGE AS build
ENV CGO_ENABLED=0
WORKDIR /go/src/

RUN mkdir kpt-render

COPY go.mod go.sum kpt-render/
RUN cd kpt-render && go mod download && cd ..

COPY . .
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /usr/local/bin/kpt-render

#############################################

FROM $BASE_IMAGE
COPY --from=build /usr/local/bin/kpt-render /usr/local/bin/kpt-render

############################################

WORKDIR /home/argocd/cmp-server/config
COPY kpt-render-plugin.yaml ./plugin.yaml
