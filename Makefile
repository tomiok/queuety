# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=queuety

# Docker parameters
IMAGE_NAME=queuety
CONTAINER_NAME=queuety
VERSION=latest

# Linting and formatting tools
GOLANGCI_LINT_VERSION=v2.4.0

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color
PRINT=printf

.PHONY: all build clean test coverage help
.PHONY: install-tools install-linters install-formatters
.PHONY: lint lint-fix format format-check
.PHONY: deps deps-update deps-verify deps-clean
.PHONY: ci-setup ci-lint ci-test ci-build
.PHONY: run stop logs restart shell docker-build docker-run

# Default target
all: clean deps format lint test build

## Build
build: ## Build the Docker image
	@$(PRINT) "$(BLUE)Building Docker image $(IMAGE_NAME):$(VERSION)...$(NC)\n"
	@docker build -t $(IMAGE_NAME):$(VERSION) .

build-binary: ## Build the Go binary locally
	@$(PRINT) "$(BLUE)Building $(BINARY_NAME) binary...$(NC)\n"
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/$(BINARY_NAME)

build-all: ## Build for multiple platforms
	@$(PRINT) "$(BLUE)Building for multiple platforms...$(NC)\n"
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)

clean: ## Clean build files and Docker resources
	@$(PRINT) "$(BLUE)Cleaning...$(NC)\n"
	$(GOCLEAN)
	rm -f $(BINARY_NAME)*
	docker compose down -v || true
	docker rmi $(IMAGE_NAME):$(VERSION) || true
	docker system prune -f

## Testing
test: ## Run tests
	@$(PRINT) "$(BLUE)Running tests...$(NC)\n"
	$(GOTEST) -v -race ./...

coverage: ## Generate test coverage
	@$(PRINT) "$(BLUE)Generating test coverage...$(NC)\n"
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@$(PRINT) "$(GREEN)Coverage report generated: coverage.html$(NC)\n"

## Installation of tools
install-tools: install-linters install-formatters ## Install all development tools

install-linters: ## Install golangci-lint
	@$(PRINT) "$(BLUE)Installing golangci-lint $(GOLANGCI_LINT_VERSION)...$(NC)\n"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	}
	@$(PRINT) "$(GREEN)golangci-lint $(GOLANGCI_LINT_VERSION) installed$(NC)\n"

install-formatters: ## Install gofumpt and goimports
	@$(PRINT) "$(BLUE)Installing gofumpt and goimports...$(NC)\n"
	@command -v gofumpt >/dev/null 2>&1 || go install mvdan.cc/gofumpt@latest
	@command -v goimports >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	@$(PRINT) "$(GREEN)Formatters installed (gofumpt, goimports)$(NC)\n"

## Linting and Formatting
lint: ## Run golangci-lint
	@$(PRINT) "$(BLUE)Running golangci-lint...$(NC)\n"
	golangci-lint run --config .golangci.yml

format: ## Format code
	@$(PRINT) "$(BLUE)Formatting code with gofumpt + goimports...$(NC)\n"
	@gofumpt -w .
	@goimports -w .

format-check: ## Check if code is formatted
	@$(PRINT) "$(BLUE)Checking code formatting...$(NC)\n"
	@if [ -n "$$(gofumpt -d .)" ] || [ -n "$$(goimports -l .)" ]; then \
		$(PRINT) "$(RED)Code formatting issues found.$(NC)\n"; \
		$(PRINT) "$(YELLOW)Run 'make format' to fix formatting issues$(NC)\n"; \
		exit 1; \
	else \
		$(PRINT) "$(GREEN)All files are properly formatted$(NC)\n"; \
	fi

## Documentation
docs: ## Generate documentation
	@$(PRINT) "$(BLUE)Generating documentation...$(NC)\n"
	$(GOCMD) doc -all ./...

docs-serve: ## Serve documentation locally
	@$(PRINT) "$(BLUE)Serving documentation on http://localhost:6060$(NC)\n"
	godoc -http=:6060

## Docker support
run: build ## Build and run Docker container
	@$(PRINT) "$(BLUE)Running Docker container with Docker Compose...$(NC)\n"
	@docker compose up -d

stop: ## Stop and remove Docker containers
	@$(PRINT) "$(BLUE)Stopping Docker containers...$(NC)\n"
	docker compose down

logs: ## Show Docker container logs
	@$(PRINT) "$(BLUE)Showing container logs...$(NC)\n"
	docker compose logs -f queuety

restart: stop build run ## Restart the application (stop, build, run)

shell: ## Open shell in running container
	@$(PRINT) "$(BLUE)Opening shell in container...$(NC)\n"
	docker exec -it $(CONTAINER_NAME) /bin/sh

docker-build: build ## Alias for build (Docker image)
docker-run: run ## Alias for run (Docker container)

tools-version: ## Show tools versions
	@$(PRINT) "$(BLUE)Tools versions:$(NC)\n"
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint version || $(PRINT) "$(RED)golangci-lint not installed$(NC)\n"
	@command -v gofumpt >/dev/null 2>&1 && gofumpt --version || $(PRINT) "$(RED)gofumpt not installed$(NC)\n"
	@command -v goimports >/dev/null 2>&1 && $(PRINT) "goimports installed\n" || $(PRINT) "$(RED)goimports not installed$(NC)\n"

help: ## Show this help message
	@$(PRINT) "$(GREEN)Available targets:$(NC)\n\n"
	@$(PRINT) "$(YELLOW)Basic commands:$(NC)\n"
	@$(PRINT) "  $(BLUE)make build$(NC)          - Build Docker image\n"
	@$(PRINT) "  $(BLUE)make run$(NC)            - Run Docker container\n"
	@$(PRINT) "  $(BLUE)make test$(NC)           - Run tests\n"
	@$(PRINT) "  $(BLUE)make clean$(NC)          - Clean everything\n\n"
	@$(PRINT) "$(YELLOW)Development workflow:$(NC)\n"
	@$(PRINT) "  $(BLUE)make dev$(NC)            - Full development cycle (format + lint + test)\n"
	@$(PRINT) "  $(BLUE)make quick$(NC)          - Quick checks (format-check + lint)\n\n"
	@$(PRINT) "$(YELLOW)Formatting tools:$(NC)\n"
	@$(PRINT) "  $(BLUE)make format$(NC)         - Format all Go files\n"
	@$(PRINT) "  $(BLUE)make format-check$(NC)   - Check if code is formatted\n\n"
	@$(PRINT) "$(YELLOW)Setup:$(NC)\n"
	@$(PRINT) "  $(BLUE)make install-tools$(NC)  - Install all development tools\n"
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(YELLOW)All available targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Check if tools are installed
check-tools:
	@$(PRINT) "$(BLUE)Checking if required tools are installed...$(NC)\n"
	@command -v golangci-lint >/dev/null 2>&1 || ($(PRINT) "$(RED)golangci-lint not found. Run 'make install-linters'$(NC)\n" && exit 1)
	@command -v gofumpt >/dev/null 2>&1 || ($(PRINT) "$(RED)gofumpt not found. Run 'make install-formatters'$(NC)\n" && exit 1)
	@command -v goimports >/dev/null 2>&1 || ($(PRINT) "$(RED)goimports not found. Run 'make install-formatters'$(NC)\n" && exit 1)
	@$(PRINT) "$(GREEN)All required tools are installed$(NC)\n"
