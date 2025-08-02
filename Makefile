WORK_DIR := $(shell pwd)
DIST_DIR := $(WORK_DIR)/dist

VERSION ?= local
COMMIT_ID ?= local

BINARY_NAME := main

BUILD_CMD_DEV := go build -ldflags="-X main.mode=dev -X main.version=$(VERSION) -X main.commitId=$(COMMIT_ID)"
BUILD_CMD_PROD := go build -tags prod -ldflags="-s -w -X main.mode=prod -X main.version=$(VERSION) -X main.commitId=$(COMMIT_ID)"

.PHONY: clean dev build

clean:
	go clean
	rm -rf $(DIST_DIR)

dev:
	$(BUILD_CMD_DEV) -o $(DIST_DIR)/$(BINARY_NAME)
	$(DIST_DIR)/$(BINARY_NAME)

build:
	GOOS=linux GOARCH=amd64 $(BUILD_CMD_PROD) -o $(DIST_DIR)/$(BINARY_NAME)
