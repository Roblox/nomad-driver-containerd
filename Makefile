PLUGIN_BINARY=containerd-driver
export GO111MODULE=on

default: build

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf ${PLUGIN_BINARY}

.PHONY: build
build:
	go build -o ${PLUGIN_BINARY} .
