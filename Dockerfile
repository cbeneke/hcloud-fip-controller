FROM golang:alpine as builder
RUN mkdir /build && \
  apk add git && \
  go get -u -v -d github.com/cbeneke/hcloud-fip-controller
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM alpine:latest
RUN adduser -S -D -H -h /app user
COPY --from=builder /build/main /app/fip-controller
RUN chmod +x /app/fip-controller && chown user: /app/fip-controller
USER user
CMD ["/app/fip-controller"]
