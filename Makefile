BINARY := zoh
VERSION ?= dev

.PHONY: build test lint vet clean release-dry-run

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./... -race -v

lint: vet
	@echo "Lint passed (go vet)"

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

release-dry-run:
	goreleaser release --snapshot --clean
