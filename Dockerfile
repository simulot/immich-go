FROM golang:1.23 AS builder

WORKDIR /go/src/app
COPY . .
RUN go mod download &&  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/immich-go -ldflags="-s -w -extldflags=-static" main.go

FROM gcr.io/distroless/base-debian12

COPY --from=builder  /go/bin/immich-go /
CMD ["/immich-go"]