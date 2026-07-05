.PHONY: build test fmt fmt-check tidy tidy-check clean

GO ?= go
GOFMT ?= gofmt

build:
	$(GO) build -o dratools ./cmd/dratools

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -w ./cmd ./internal

fmt-check:
	test -z "$$($(GOFMT) -l ./cmd ./internal)"

tidy:
	$(GO) mod tidy

tidy-check:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

clean:
	rm -rf dratools dist
