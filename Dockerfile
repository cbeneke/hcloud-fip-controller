FROM golang:latest
WORKDIR /go/src/github.com/cbeneke/hcloud-fip-controller
RUN go get -v github.com/cbeneke/hcloud-fip-controller

FROM alpine:latest
RUN apk --no-cache add ca-certificates && mkdir /app
COPY --from=0 /go/src/github.com/cbeneke/hcloud-fip-controller/hcloud-fip-controller /app/
CMD ["/app/hcloud-fip-controller"]
