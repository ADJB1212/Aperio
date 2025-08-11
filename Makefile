BIN_DIR = bin
BINARY = $(BIN_DIR)/aperio
SRC = cmd/aperio/main.go
VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: all build clean tidy

all: build

build:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) $(SRC)

clean:
	@rm -f $(BINARY)

tidy:
	@go mod tidy
