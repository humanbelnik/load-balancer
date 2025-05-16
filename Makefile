MAIN_PKG=./cmd/app
BIN_NAME=lb
BIN_DIR=bin
CONFIG=./config/config.yaml

PORT=8888
HOST=localhost
RLIMIT=true
RLSTORE=ratelimiter.db

.PHONY: all build run fmt lint test clean

all: build

build:
	@echo "Building..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BIN_NAME) $(MAIN_PKG)

run:
	@echo "Running..."
	@go run $(MAIN_PKG) -port=$(PORT) -host=$(HOST) -config=$(CONFIG) -rlimit=$(RLIMIT) -rlstore=$(RLSTORE)

test: mock
	@echo "Testing..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -rf ./internal/mocks

mock:
	mockery --name=Server --dir=internal/balancer/server/server --output=internal/mocks --outpkg=mocks --filename=mock_server.go
	mockery --name=Pool --dir=internal/balancer/balancer --output=internal/mocks --outpkg=mocks --filename=mock_pool.go
	mockery --name=Policy --dir=internal/balancer/balancer --output=internal/mocks --outpkg=mocks --filename=mock_policy.go
