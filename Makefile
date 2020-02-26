.DEFAULT_GOAL := help

##
## Global ENV vars
##

GIT_SHA ?= $(shell git rev-parse --short=8 HEAD)
GIT_TAG ?= $(shell git describe --tags --abbrev=0)

##
## Helpful Help
##

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


##
## Building
##

.PHONY: ios_framework
ios_framework: ## Build iOS Framework for mobile
	gomobile bind -target=ios github.com/phoreproject/pm-go/mobile

.PHONY: android_framework
android_framework: ## Build Android Framework for mobile
	gomobile bind -target=android github.com/phoreproject/pm-go/mobile

##
## Protobuf compilation
##
P_TIMESTAMP = Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp
P_ANY = Mgoogle/protobuf/any.proto=github.com/golang/protobuf/ptypes/any

PKGMAP = $(P_TIMESTAMP),$(P_ANY)

.PHONY: protos
protos: ## Build go files for proto definitions
	cd pb/protos && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=$(PKGMAP):.. *.proto


##
## Testing
##
OPENBAZAARD_NAME ?= marketplaced-$(GIT_SHA)
BITCOIND_PATH ?= .

.PHONY: marketplaced
marketplaced: ## Build daemon
	$(info "Building marketplace daemon...")
	go build -o ./$(OPENBAZAARD_NAME) .

.PHONY: qa_test
qa_test: marketplaced ## Run QA test suite against current working copy
	$(info "Running QA... (marketplaced: ../$(OPENBAZAARD_NAME) bitcoind: $(BITCOIND_PATH)/bin/bitcoind)")
	(cd qa && ./runtests.sh ../$(OPENBAZAARD_NAME) $(BITCOIND_PATH)/bin/bitcoind)

##
## Docker
##
PUBLIC_DOCKER_REGISTRY ?= phoremarketplace
QA_DEV_TAG ?= 0.10

DOCKER_SERVER_IMAGE_NAME ?= $(PUBLIC_DOCKER_REGISTRY)/server:$(GIT_TAG)
DOCKER_QA_IMAGE_NAME ?= $(PUBLIC_DOCKER_REGISTRY)/server-qa:$(QA_DEV_TAG)
DOCKER_DEV_IMAGE_NAME ?= $(PUBLIC_DOCKER_REGISTRY)/server-dev:$(QA_DEV_TAG)


.PHONY: docker_build
docker_build: ## Build container for daemon
	docker build -t $(DOCKER_SERVER_IMAGE_NAME) .

.PHONY: docker_push
docker_push: docker ## Push container for daemon
	docker push $(DOCKER_SERVER_IMAGE_NAME)

.PHONY: qa_docker_build
qa_docker_build: ## Build container with QA test dependencies included
	docker build -t $(DOCKER_QA_IMAGE_NAME) -f ./Dockerfile.qa .

.PHONY: qa_docker_push
qa_docker_push: qa_docker_build ## Push container for daemon QA test environment
	docker push $(DOCKER_QA_IMAGE_NAME)

.PHONY: dev_docker_build
dev_docker: ## Build container with dev dependencies included
	docker build -t $(DOCKER_DEV_IMAGE_NAME) -f ./Dockerfile.dev .

.PHONY: dev_docker_push
dev_docker_push: dev_docker_build ## Push container for daemon dev environment
	docker push $(DOCKER_DEV_IMAGE_NAME)
