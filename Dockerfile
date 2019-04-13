FROM golang:alpine as builder
RUN mkdir /build && \
  apk add git && \
  go get -u -v -d github.com/cbeneke/hcloud-fip-controller
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM alpine:latest
RUN adduser -S -D -H -h /app user
USER user
COPY --from=builder /build/main /app/
WORKDIR /app
RUN chmod +x ./main
CMD ["./main"]
