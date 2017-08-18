FROM golang:1.7.5
WORKDIR /go/src/github.com/alexellis/faas/queue-worker

COPY vendor     vendor
COPY main.go  .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:3.5
RUN apk add --no-cache ca-certificates
WORKDIR /root/

EXPOSE 8080
ENV http_proxy      ""
ENV https_proxy     ""

COPY --from=0 /go/src/github.com/alexellis/faas/queue-worker/app    .

CMD ["./app"]
