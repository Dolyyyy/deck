BINARY_NAME=deck
BUILD_DIR=bin
VERSION?=0.2.0
BUILD_DATE?=$(shell date -u +"%Y-%m-%d")

LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -s -w"

.PHONY: all build test lint clean run cross-build

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/deck

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -v -race ./...

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 0)
	golangci-lint run ./...

cross-build:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/deck
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/deck
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/deck
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/deck
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/deck

clean:
	rm -rf $(BUILD_DIR)
