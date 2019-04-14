FROM golang:alpine as builder
RUN apk add git && \
  go get -v -d github.com/cbeneke/hcloud-fip-controller && \
  go build github.com/cbeneke/hcloud-fip-controller

FROM alpine:latest
RUN adduser -S -D -H -h /app runuser
WORKDIR /app
USER runuser
COPY --from=builder /hcloud-fip-controller /app/fip-controller
ENTRYPOINT ./fip-controller
