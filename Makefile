PACKAGES := $(shell go list ./... | grep -v /vendor/)
TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=registry.terraform.io
NAMESPACE=martian-cloud
NAME=tharsis
BINARY=terraform-provider-${NAME}
VERSION?=0.0.0-$(shell git rev-parse --short HEAD)
GCFLAGS:=-gcflags all=-trimpath=${PWD}
LDFLAGS:=-ldflags "-s -w -X internal/provider.Version=${VERSION}"
OS_ARCH=linux_amd64

default: install

.PHONY: build install default test testacc

build:
	CGO_ENABLED=0 go build ${GCFLAGS} ${LDFLAGS} -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

.PHONY: vet
vet: ## run golint on all Go package
	@go vet $(PACKAGES)

.PHONY: fmt
fmt: ## run "go fmt" on all Go packages
	@go fmt $(PACKAGES)

test:
	go test $(TEST) || exit 1
	echo $(TEST) | xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

.PHONY: generate
generate: ## run go generate
	go generate $(PACKAGES)
