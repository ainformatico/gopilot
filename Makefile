MAIN_PACKAGE_PATH := .
BINARY_NAME := gopilot

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: build the application
.PHONY: build
build:
	go build -ldflags="-s -w" -o=${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## run: run the application
.PHONY: run
run: build
	${BINARY_NAME}
