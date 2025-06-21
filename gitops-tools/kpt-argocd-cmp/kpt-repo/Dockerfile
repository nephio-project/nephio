ARG BUILDER_IMAGE=golang:1.24.2
ARG BASE_IMAGE=quay.io/argoproj/argocd:v2.14.6

FROM --platform=$BUILDPLATFORM $BUILDER_IMAGE AS build
ENV CGO_ENABLED=0
WORKDIR /go/src/

RUN mkdir kpt-repo

COPY go.mod go.sum kpt-repo/
RUN cd kpt-repo && go mod download && cd ..

COPY . .
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /usr/local/bin/kpt-repo

#############################################

FROM $BASE_IMAGE
COPY --from=build /usr/local/bin/kpt-repo /usr/local/bin/kpt-repo

############################################

WORKDIR /home/argocd/cmp-server/config
COPY kpt-repo-plugin.yaml ./plugin.yaml
