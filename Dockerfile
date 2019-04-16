FROM golang:alpine as builder
RUN mkdir /out
WORKDIR /out
RUN apk add git && \
  go get -v -d github.com/cbeneke/hcloud-fip-controller
ADD . ${GOPATH}/src/github.com/cbeneke/hcloud-fip-controller
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' github.com/cbeneke/hcloud-fip-controller

FROM alpine:latest
RUN adduser -S -D -H -h /app runuser && \
  apk add --no-cache ca-certificates
WORKDIR /app
USER runuser
COPY --from=builder /out/hcloud-fip-controller /app/fip-controller
ENTRYPOINT ./fip-controller
