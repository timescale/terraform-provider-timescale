default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=30m -parallel=10 ./...

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	TF_ACC=1 go test ./internal/provider/ -v -timeout 10m -sweep=all

# Run acceptance tests
testacc:
	TF_ACC=1 go test ./internal/provider/ -v -cover -timeout 120m

.PHONY: fmt lint test testacc sweep build install generate
