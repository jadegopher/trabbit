.PHONY: all build run run-dev profile-cpu view-cpu test clean help

# Binary name
BINARY_NAME=trabbit
PACKAGE_PATH=./cmd/trabbit

all: build ## Build the binary (default)

build: ## Build the binary
	go build -o $(BINARY_NAME) $(PACKAGE_PATH)

run: ## Run the app
	go run $(PACKAGE_PATH)

run-dev: ## Run the app in dev mode
	go run -tags dev $(PACKAGE_PATH)

profile-cpu: ## Run the app with CPU profiling enabled
	go run -tags dev $(PACKAGE_PATH) -cpuprofile cpu.prof
	@echo "CPU profile saved to cpu.prof. Run 'make view-cpu' to analyze."

view-cpu: ## Open the CPU profile in a web browser
	go tool pprof -http=:8080 cpu.prof

test: ## Run tests
	go test -v ./...

clean: ## Remove build artifacts and profiles
	go clean
	rm -f $(BINARY_NAME)
	rm -f cpu.prof

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
