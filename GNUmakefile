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

# Run acceptance tests locally (requires .env.development file with: TF_ACC, TF_LOG, PEER_ACCOUNT_ID, PEER_VPC_ID, PEER_TGW_ID, PEER_REGION, TF_VAR_ts_project_id, TF_VAR_ts_access_key, TF_VAR_ts_secret_key, TIMESCALE_DEV_URL)
testacc-local:
	set -a && source .env.development && set +a && \
	TF_ACC=1 go test ./internal/provider/ -v -cover -timeout 120m

.PHONY: fmt lint test testacc sweep build install generate
