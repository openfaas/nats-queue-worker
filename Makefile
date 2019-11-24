DOCKER_REPOSITORY=openfaas/queue-worker
BUILD_ARGS=--build-arg http_proxy="${http_proxy}" --build-arg https_proxy="${https_proxy}"
ARCHS=amd64 arm64 armhf ppc64le
TAG?=latest

# docker manifest command will work with Docker CLI 18.03 or newer
# but for now it's still experimental feature so we need to enable that
export DOCKER_CLI_EXPERIMENTAL=enabled

.PHONY: build
build: $(addprefix build-,$(ARCHS))  ## Build Docker images for all architectures 

.PHONY: build-%
build-%:
	docker build $(BUILD_ARGS) --build-arg go_opts="GOARCH=$*" -t $(DOCKER_REPOSITORY):$(TAG)-$* .

build-armhf:
	docker build $(BUILD_ARGS) --build-arg go_opts="GOARCH=arm GOARM=6" -t $(DOCKER_REPOSITORY):$(TAG)-armhf .

.PHONY: push
push: $(addprefix push-,$(ARCHS)) ## Push Docker images for all architectures

.PHONY: push-%
push-%:
	docker push $(DOCKER_REPOSITORY):$(TAG)-$* 

.PHONY: manifest
manifest: ## Create and push Docker manifest to combine all architectures in multi-arch Docker image
	docker manifest create --amend $(DOCKER_REPOSITORY):$(TAG) $(addprefix $(DOCKER_REPOSITORY):$(TAG)-,$(ARCHS))
	$(MAKE) $(addprefix manifest-annotate-,$(ARCHS))
	docker manifest push -p $(DOCKER_REPOSITORY):$(TAG)

.PHONY: manifest-annotate-%
manifest-annotate-%:
	docker manifest annotate $(DOCKER_REPOSITORY):$(TAG) $(DOCKER_REPOSITORY):$(TAG)-$* --os linux --arch $*

.PHONY: manifest-annotate-armhf
manifest-annotate-armhf:
	docker manifest annotate $(DOCKER_REPOSITORY):$(TAG) $(DOCKER_REPOSITORY):$(TAG)-armhf --os linux --arch arm --variant v6

.PHONY: test
test: ## Run tests
	go test -v ./...

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show help
	@echo "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:"
	@grep -E '^[a-zA-Z_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
