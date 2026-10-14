VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test lint integration snapshot docker clean

build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o govno ./cmd/govno

test:
	go test -race ./...

lint:
	golangci-lint run

# Runs standalone (fake S3 + stub mysqldump). Set the GOVNO_IT_* vars from
# internal/backup/integration_test.go to use a real MySQL and/or S3.
integration:
	go test -tags integration -run EndToEnd -v ./internal/backup/

snapshot:
	goreleaser release --snapshot --clean

docker:
	docker build --build-arg VERSION=$(VERSION) -t govno:$(VERSION) .

clean:
	rm -rf dist govno
