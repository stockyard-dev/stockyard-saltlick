VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)

.PHONY: build run clean

build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o saltlick ./cmd/saltlick/

run: build
	./saltlick

clean:
	rm -f saltlick
