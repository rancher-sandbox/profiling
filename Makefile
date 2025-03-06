.PHONY: build

build:
	go build -o bin/collector ./cmd/collector
	go build -o bin/operator ./cmd/operator