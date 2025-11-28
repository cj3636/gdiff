.PHONY: build clean install test fmt help

# Binary name
BINARY_NAME=gdiff

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Display this help message
	@echo "gdiff - A beautiful terminal diff viewer"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	$(GOBUILD) -o $(BINARY_NAME) -v

clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

install: ## Install the application to $GOPATH/bin
	$(GOCMD) install

test: ## Run tests
	$(GOTEST) -v ./...

fmt: ## Format Go code
	$(GOFMT) ./...

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

run: build ## Build and run the application with example files
	@echo "Building and running gdiff..."
	@./$(BINARY_NAME) --version

all: deps fmt build test ## Run deps, fmt, build, and test

.DEFAULT_GOAL := help
