build:
	@goreleaser --snapshot --rm-dist

dev:
	@go run main.go -service

release:
	@goreleaser release

fix_dependencies:
	@go get -u golang.org/x/sys

.PHONY: build release fix_dependencies
