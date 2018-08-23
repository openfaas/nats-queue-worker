TAG?=latest

build:
	docker build --build-arg http_proxy="${http_proxy}" --build-arg https_proxy="${https_proxy}" -t openfaas/queue-worker:$(TAG) .

push:
	docker push openfaas/queue-worker:$(TAG)

all: build

ci-armhf-build:
	docker build --build-arg http_proxy="${http_proxy}" --build-arg https_proxy="${https_proxy}" -t openfaas/queue-worker:$(TAG)-armhf . -f Dockerfile.armhf
ci-armhf-push:
	docker push openfaas/queue-worker:$(TAG)-armhf


