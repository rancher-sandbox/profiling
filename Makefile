.PHONY: build
.PHONY: build-collector
.PHONY: build-operator

build: build-collector build-operator

build-collector:
	go build -o bin/collector ./cmd/collector
build-operator:
	go build -o bin/operator ./cmd/operator

local: build-collector
	./bin/collector -c ./examples/local/config.yaml