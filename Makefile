BINARY ?= containerd-driver
ifndef $(GOLANG)
    GOLANG=$(shell which go)
    export GOLANG
endif

export GO111MODULE=on

default: build

.PHONY: clean
clean:
	rm -f $(BINARY)

.PHONY: build
build:
	$(GOLANG) build -o $(BINARY) .

.PHONY: test
test:
	./tests/run_tests.sh
