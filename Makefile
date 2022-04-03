VERSION ?= 1.0.0

build:
	@go build -ldflags "-s -w -X 'main.Version=$(VERSION)'" -o dist/mailbiter main.go

.PHONY: build
