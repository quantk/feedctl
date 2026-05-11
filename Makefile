CGO_ENABLED ?= 0

.PHONY: build test fmt clean

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -o feedctl ./cmd/feedctl

test:
	CGO_ENABLED=$(CGO_ENABLED) go test ./...

fmt:
	gofmt -w cmd internal

clean:
	rm -f feedctl
