BINARY  := cordis
PKG     := ./...
LDFLAGS := -s -w

.PHONY: help build run test vet fmt tidy docker-build docker-run clean

help: ## Show the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

build: ## Build the server binary for the host platform
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/server

run: ## Run the server against the environment of the shell
	go run ./cmd/server

test: ## Run the test suite
	go test -race $(PKG)

vet: ## Report suspicious constructs
	go vet $(PKG)

fmt: ## Format every source file
	go fmt $(PKG)

tidy: ## Synchronise go.mod and go.sum
	go mod tidy

docker-build: ## Build the multi stage image
	docker build -t cordis/backend:latest .

docker-run: ## Run the image with the variables of .env
	docker run --rm --env-file ../.env -p 8080:8080 cordis/backend:latest

clean: ## Remove build output
	rm -rf bin data/uploads