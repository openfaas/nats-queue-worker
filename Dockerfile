
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.16-alpine as build

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
COPY auth.go    .
COPY .git       .
COPY readconfig.go      .
COPY readconfig_test.go .

ARG go_opts

RUN  VERSION=$(git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///') \
    && GIT_COMMIT=$(git rev-list -1 HEAD) \
    && env $go_opts CGO_ENABLED=0 go build \
        --ldflags "-s -w \
        -X github.com/openfaas/nats-queue-worker/version.GitCommit=${GIT_COMMIT}\
        -X github.com/openfaas/nats-queue-worker/version.Version=${VERSION}" \
        -a -installsuffix cgo -o worker .

# we can't add user in next stage because it's from scratch
# ca-certificates and tmp folder are also missing in scratch
# so we add all of it here and copy files in next stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot

WORKDIR /
USER nonroot:nonroot
COPY --from=build /go/src/github.com/openfaas/nats-queue-worker/worker    .
CMD ["/worker"]
