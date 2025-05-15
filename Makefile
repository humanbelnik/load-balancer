MAIN_PKG=./cmd/app
BIN_NAME=lb
BIN_DIR=bin
CONFIG=./config/config.yaml

.PHONY: all build run fmt lint test clean

all: build

build:
	@echo "Building..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BIN_NAME) $(MAIN_PKG)

run:
	@echo "Running..."
	@go run $(MAIN_PKG) -config=$(CONFIG)

fmt:
	@echo "Formatting..."
	@go fmt ./...

lint:
	@echo "Linting..."
	@go vet ./...
	@which golint >/dev/null && golint ./... || echo "golint not installed"

test: mock
	@echo "Testing..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -rf ./internal/mocks


mock:
	mockery --name=Server --dir=internal/server/server --output=internal/mocks --outpkg=mocks --filename=mock_server.go
	mockery --name=Pool --dir=internal/balancer --output=internal/mocks --outpkg=mocks --filename=mock_pool.go
	mockery --name=Policy --dir=internal/balancer --output=internal/mocks --outpkg=mocks --filename=mock_policy.go

reload:
	@if [ -z "$(PORT)" ]; then \
		echo "Usage: make reload PORT=<port>"; \
		exit 1; \
	fi && \
	PID=$$(lsof -t -i :$(PORT)) && \
	if [ -z "$$PID" ]; then \
		echo "No process found on port $(PORT)"; \
		exit 1; \
	fi && \
	echo "Sending SIGHUP to PID $$PID (port $(PORT))" && \
	kill -SIGHUP $$PID