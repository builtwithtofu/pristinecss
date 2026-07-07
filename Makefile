# Variables
PROJECT_NAME := stylocss
GO_FILES := $(shell find . -type f -name '*.go')
GO := $(shell command -v go 2> /dev/null)
TEST_PATH ?= ./...

# Default target
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build the project"
	@echo "  build            Build the project"
	@echo "  test [path]      Runs the tests for the project. Default: ./..."
	@echo "  clean            Clean the project"
	@echo "  deps             Install dependencies"
	@echo "  help             Display this help message"

# Build target
.PHONY: build
build: check-deps clean
	$(GO) build -o ./bin/main ./cmd/stylocss

# Clean target
.PHONY: clean
clean:
	$(GO) clean

# Run tests
.PHONY: test
test: check-deps
	@echo "Running tests for: $(TEST_PATH)"
	@$(GO) test -json $(TEST_PATH) | gotestfmt

# Install dependencies
.PHONY: deps
deps: check-deps
	$(GO) mod tidy

# Check for dependencies
.PHONY: check-deps
check-deps:
ifndef GO
	$(error "Go is not installed.")
endif

# Default target to display help message
.PHONY: all
all: help


