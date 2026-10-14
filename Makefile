VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test lint integration snapshot docker clean

build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o govno ./cmd/govno

test:
	go test -race ./...

lint:
	golangci-lint run

# Needs MySQL + an S3-compatible server; see the GOVNO_IT_* vars in
# internal/backup/integration_test.go and .github/workflows/ci.yml.
integration:
	go test -tags integration -run EndToEnd -v ./internal/backup/

snapshot:
	goreleaser release --snapshot --clean

docker:
	docker build --build-arg VERSION=$(VERSION) -t govno:$(VERSION) .

clean:
	rm -rf dist govno
