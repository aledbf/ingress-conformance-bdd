# Add the following 'help' target to your Makefile
# And add help text after each target name starting with '\#\#'
.DEFAULT_GOAL:=help

.PHONY: help test build-image check-go-version run-conformance local-tests build-report show-report local-cluster codegen verify-codegen

.EXPORT_ALL_VARIABLES:

# set default shell
SHELL=/bin/bash -o pipefail -o errexit

ifndef VERBOSE
.SILENT:
endif

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

K8S_VERSION ?= v1.18.0@sha256:0e20578828edd939d25eb98496a685c76c98d54084932f76069f886ec315d694

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

test: verify-codegen ## Run conformance tests using 'go test' (local development)
	@go test

build-image: verify-codegen ## Build image to run conformance test suite
	@go test -c
	@make -C images/conformance build

check-go-version:
	@hack/check-go-version.sh

run-conformance: ## Run conformance tests using a pod
	@mkdir -p "/tmp/results"
	@RESULTS_DIR="/tmp/results" \
	./images/conformance/run_e2e.sh

build-report: ## Run tests and generate HTML report in directory
	echo "Running go tests with cucumber output..."
	go test -v --format cucumber

	echo "Generating report..."
	@docker run --rm \
		--name build-report \
		-v "$(ROOT_DIR)/reports/build":/usr/src/conformance \
		-v "$(ROOT_DIR)/.m2":/var/maven/.m2 \
		-v "$(ROOT_DIR)/reports":/reports \
		-w /usr/src/conformance \
		-e MAVEN_CONFIG=/var/maven/.m2 \
		-e INPUT_JSON_FILES=/reports \
		-e OUTPUT_DIRECTORY=/reports/output \
		-u $(shell id -u):$(shell id -g) \
		maven:3.6.3-jdk-11-slim mvn -Duser.home=/var/maven clean compile exec:java

show-report: build-report ## Starts NGINX locally to access reports using http://localhost
	echo "Starting web server..."
	echo ""
	echo "Open http://localhost:8080"
	@docker run --rm \
		--name show-report \
		-p 8080:8080 \
		-v "$(PWD)/reports/output/cucumber-html-reports":/www:ro \
		-v "$(PWD)/reports/output/nginx.conf":/etc/nginx/nginx.conf:ro \
		nginx:1.17.9-alpine

local-cluster: ## Create local cluster using kind
ifeq ($(shell which kind >/dev/null 2>&1 && kind version),)
	$(error "kind is not installed. Use a package manager (i.e 'brew install kind') or visit the official site https://kind.sigs.k8s.io")
endif
ifeq ($(shell kind get clusters -q),)
	echo "Creating kind cluster..."
	kind create cluster --config .github/kind.yaml --image "kindest/node:${K8S_VERSION}" || true
	kubectl get nodes
else
	echo "Using existing kind cluster"
endif
	# Install ingress-nginx. THIS IS TEMPORAL
	curl -sSL https://gist.githubusercontent.com/aledbf/7e67bcb338fa6a1696eb5b101597224e/raw/6b106c9992c0f8937834113b8003be05950807d9/install-ingress-nginx.sh | bash

codegen: ## Generate or update missing Go code defined in feature files
	@go run hack/codegen.go -update -conformance-path=test/conformance features

verify-codegen: ## Verifies if generated Go code is in sync with feature files
	@go run hack/codegen.go -conformance-path=test/conformance features
