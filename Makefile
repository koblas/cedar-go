BIN := .tmp/bin

.PHONY: test
test: ## Run unit tests
	#go test -vet=off -race -cover -short ./...
	go test -vet=off -cover -short ./...

.PHONY: lint
lint: $(BIN)/golangci-lint ## Lint Go and protobuf
	go vet ./...
	golangci-lint run --modules-download-mode=readonly --timeout=3m0s

$(BIN)/golangci-lint: Makefile
	@mkdir -p $(@D)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
