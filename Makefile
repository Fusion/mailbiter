VERSION ?= 1.0.0

build:
	@go build -ldflags "-s -w -X 'main.Version=$(VERSION)'" -o dist/mailbiter main.go

fix_dependencies:
	@go get -u golang.org/x/sys

.PHONY: build fix_dependencies
