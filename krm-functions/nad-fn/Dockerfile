FROM golang:1.19.2-alpine3.15
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY go.mod go.sum ./
COPY ./mutator ./mutator
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/function ./
FROM alpine:3.15
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]