# Makefile

# Binary name
BINARY_NAME := solizard

# Installation directory (GOPATH/bin)
INSTALL_DIR := $(shell go env GOPATH)/bin

# Main package to build
MAIN_PACKAGE := ./cmd/solizard

# Go commands
GO := go
GOBUILD := $(GO) build
GOINSTALL := $(GO) install

# Default target (build)
all: build

# Build binary
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	chmod +x $(BINARY_NAME)

# Install
install:
	$(GOBUILD) -o $(INSTALL_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)

# Clean (remove built files)
clean:
	rm -f $(BINARY_NAME)

# Help (display available commands)
help:
	@echo "Available commands:"
	@echo "  make          - Default build (build)"
	@echo "  make build    - Build binary"
	@echo "  make install  - Install binary"
	@echo "  make clean    - Remove built files"
	@echo "  make help     - Display available commands"

.PHONY: all build install clean help