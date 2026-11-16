# TAP build tool. On machines without make, run the go commands directly.
# Owner: Role C
#
# Override any variable on the command line, for example:
#   make run-client-gui SERVER=10.0.0.5:4242

BIN         := bin
PORT        ?= 4242
GUIPORT     ?= 8080
SERVER      ?= 127.0.0.1:$(PORT)
WORLD       ?= data/world.yaml
LINT_VERSION := v2.15.0

.PHONY: install build run-server run-client run-client-gui lint test fmt clean

install: ## Download dependencies and the linter (pinned so everyone gets the same checks)
	go mod download
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION)

build: ## Build all three programs into bin/
	go build -o $(BIN)/ ./cmd/...

run-server: ## Start the game server on PORT
	go run ./cmd/server -port $(PORT) -world $(WORLD)

run-client: ## Start the CLI client, connected to SERVER
	go run ./cmd/cli -addr $(SERVER)

run-client-gui: ## Start the GUI bridge to SERVER, then open http://127.0.0.1:GUIPORT
	go run ./cmd/gui -port $(GUIPORT) -server $(SERVER)

lint: ## Run golangci-lint (includes go vet and staticcheck)
	golangci-lint run ./...

test: ## Run all tests with the race detector
	go test -race ./...

fmt: ## Format all Go files
	gofmt -w .

clean: ## Remove build output
	rm -rf $(BIN)
	go clean
