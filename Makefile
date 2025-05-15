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

test:
	@echo "Testing..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
