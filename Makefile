# Add the following 'help' target to your Makefile
# And add help text after each target name starting with '\#\#'
.DEFAULT_GOAL:=help

.PHONY: help test build-image check-go-version run-conformance local-tests build-report show-report

.EXPORT_ALL_VARIABLES:

# set default shell
SHELL=/bin/bash -o pipefail -o errexit

ifndef VERBOSE
.SILENT:
endif

DOCKER_CLI_EXPERIMENTAL ?= enabled

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

test: generate-bindata ## Run conformance tests using 'go test' (local development)
	@go test

build-image: generate-bindata ## Build image to run conformance test suite
	@go test -c
	@make -C images/conformance build

check-go-version:
	@hack/check-go-version.sh

generate-bindata:
	@hack/generate-bindata.sh

run-conformance: ## Run conformance tests using a pod
	@mkdir -p "/tmp/results"
	@RESULTS_DIR="/tmp/results" \
	./images/conformance/run_e2e.sh

build-report: generate-bindata ## Run tests and generate HTML report in directory
	echo "Running go tests with cucumber output..."
	go test --output-file "$(PWD)/reports/ingress-conformance.json" --format cucumber

	echo "Generating report..."
	@docker run --rm \
		--name build-report \
		-v "$(PWD)/reports/build":/usr/src/conformance \
		-v "$(PWD)/.m2":/var/maven/.m2 \
		-v "$(PWD)/reports/output":/report-output:rw \
		-v "$(PWD)/reports/ingress-conformance.json":/input.json:ro \
		-w /usr/src/conformance \
		-e MAVEN_CONFIG=/var/maven/.m2 \
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
		nginx:1.17.8-alpine
