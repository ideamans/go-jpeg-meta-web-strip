.PHONY: data clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

data: ## Generate test data using ImageMagick
	@echo "Generating test data..."
	@go run datacreator/cmd/main.go

clean: ## Clean generated test data
	@echo "Cleaning test data..."
	@rm -rf testdata/

test: ## Run tests
	@echo "Running tests..."
	@go test ./...

build: ## Build the package
	@echo "Building..."
	@go build ./...