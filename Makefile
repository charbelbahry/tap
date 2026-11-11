# TAP build tool. On machines without make, run the go commands directly.
# Owner: Role C

BIN := bin

.PHONY: install build run-server run-client run-client-gui lint test fmt clean

install: ## Download dependencies and the linter
	go mod download
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

build: ## Build all three programs into bin/
	go build -o $(BIN)/ ./cmd/...

run-server: ## Start the game server on port 4242
	go run ./cmd/server

run-client: ## Start the CLI client
	go run ./cmd/cli

run-client-gui: ## Start the GUI bridge, then open http://127.0.0.1:8080
	go run ./cmd/gui

lint: ## Run golangci-lint (includes go vet and staticcheck)
	golangci-lint run ./...

test: ## Run all tests with the race detector
	go test -race ./...

fmt: ## Format all Go files
	gofmt -w .

clean: ## Remove build output
	rm -rf $(BIN)
	go clean
