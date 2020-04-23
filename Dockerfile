FROM golang:1.13-alpine as golang
ENV CGO_ENABLED=0
ENV GO111MODULE=off

WORKDIR /go/src/github.com/openfaas/nats-queue-worker

COPY vendor     vendor
COPY handler    handler
COPY version    version
COPY nats       nats
COPY main.go  .
COPY types.go .
COPY readconfig.go .
COPY readconfig_test.go .
COPY auth.go .
COPY .git     .

ARG go_opts

RUN apk add --no-cache git

RUN  VERSION=$(git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///') \
    && GIT_COMMIT=$(git rev-list -1 HEAD) \
    && env $go_opts CGO_ENABLED=0 go build \
        --ldflags "-s -w \
        -X github.com/openfaas/nats-queue-worker/version.GitCommit=${GIT_COMMIT}\
        -X github.com/openfaas/nats-queue-worker/version.Version=${VERSION}" \
        -a -installsuffix cgo -o app . \
    && addgroup -S app \
    && adduser -S -g app app \
    && mkdir /scratch-tmp

# we can't add user in next stage because it's from scratch
# ca-certificates and tmp folder are also missing in scratch
# so we add all of it here and copy files in next stage

FROM scratch

EXPOSE 8080
ENV http_proxy      ""
ENV https_proxy     ""
USER app

COPY --from=golang /etc/passwd /etc/group /etc/
COPY --from=golang /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=golang --chown=app:app /scratch-tmp /tmp
COPY --from=golang /go/src/github.com/openfaas/nats-queue-worker/app    .

CMD ["./app"]
