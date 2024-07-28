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
	rm gopilot 2>/dev/null || true
	go build -ldflags="-s -w" -o=${BINARY_NAME} ${MAIN_PACKAGE_PATH}

install:
	mkdir -p ${GOPATH}/bin
	ln -s ${PWD}/${BINARY_NAME} ${GOPATH}/bin/${BINARY_NAME}

## run: run the application
.PHONY: run
run: build
	${BINARY_NAME}

## debug: debug the application
.PHONY: debug
debug: build
	${BINARY_NAME} -d
