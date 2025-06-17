VERSION := $(shell git describe --tags --always --dirty="-dev" --match "v*.*.*" || echo "development" )
VERSION := $(VERSION:v%=%)

.PHONY: build
build:
	@CGO_ENABLED=0 go build \
			-ldflags "-X main.version=${VERSION}" \
			-o ./bin/gh-artifacts-sync \
		github.com/flashbots/gh-artifacts-sync/cmd

.PHONY: snapshot
snapshot:
	@goreleaser release --snapshot --clean

.PHONY: help
help:
	@go run github.com/flashbots/gh-artifacts-sync/cmd serve --help

.PHONY: serve
serve:
	@go run github.com/flashbots/gh-artifacts-sync/cmd serve
