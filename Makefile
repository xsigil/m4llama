BIN_DIR := bin
APP_NAME := $(shell basename $(CURDIR))

.PHONY: all build run test tidy clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run:
	@go run ./cmd/$(APP_NAME)

test:
	@go test -v ./...

tidy:
	@go mod tidy

clean:
	@rm -rf $(BIN_DIR) *.db *.db-journal
