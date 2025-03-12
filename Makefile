.PHONY: build
.PHONY: build-collector
.PHONY: build-operator
REGISTRY?=docker.io
REPO?=alex7285
TAG?=dev

build: build-collector build-operator

build-collector:
	go build -o bin/collector ./cmd/collector
build-operator:
	go build -o bin/operator ./cmd/operator

local: build-collector
	./bin/collector -c ./examples/local/config.yaml

image:
	docker build -f ./package/collector/Dockerfile -t $(REGISTRY)/$(REPO)/collector:$(TAG) .

push: image
	docker push $(REGISTRY)/$(REPO)/collector:$(TAG)
