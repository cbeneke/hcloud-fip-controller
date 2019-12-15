FROM golang:alpine as builder
RUN mkdir /app
WORKDIR /app
ADD . /app
RUN apk add git \
  && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' ./cmd/fip-controller

FROM alpine:latest
RUN adduser -S -D -H -h /app runuser && \
  apk add --no-cache ca-certificates
WORKDIR /app
USER runuser
COPY --from=builder /app/fip-controller /app/fip-controller
ENTRYPOINT ./fip-controller
