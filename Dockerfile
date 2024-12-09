
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23-alpine as build

ENV GO111MODULE=on
ENV CGO_ENABLED=0

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git

WORKDIR /go/src/github.com/openfaas/nats-queue-worker

COPY vendor     vendor
COPY handler    handler
COPY version    version
COPY nats       nats
COPY go.mod     .
COPY go.sum     .
COPY main.go    .
COPY types.go   .
COPY readconfig.go      .
COPY readconfig_test.go .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"
RUN go test $(go list ./... | grep -v integration | grep -v /vendor/ | grep -v /template/) -cover

RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build --ldflags "-s -w \
    -X \"github.com/openfaas/nats-queue-worker/version.GitCommit=${GIT_COMMIT}\" \
    -X \"github.com/openfaas/nats-queue-worker/version.Version=${VERSION}\"" \
    -a -installsuffix cgo -o worker .

# we can't add user in next stage because it's from scratch
# ca-certificates and tmp folder are also missing in scratch
# so we add all of it here and copy files in next stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot

WORKDIR /
USER nonroot:nonroot
COPY --from=build /go/src/github.com/openfaas/nats-queue-worker/worker    .
CMD ["/worker"]
