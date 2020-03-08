# Copyright 2020 The Kubernetes Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all: build push

KUBE_VERSION=$(shell curl -sS https://storage.googleapis.com/kubernetes-release/release/stable.txt)

REGISTRY ?= aledbf
IMAGE = $(REGISTRY)/ingress-conformance
TAG ?=v$(shell date +%m%d%Y)-$(shell git rev-parse --short HEAD)

build:
	cp    ../../ingress-conformance-bdd.test ingress-conformance-bdd.test
	cp -R ../../features .
	docker build \
		--build-arg KUBE_VERSION=$(KUBE_VERSION) \
		-t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)

clean:
	rm ingress-conformance-bdd.test
