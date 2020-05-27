BINARY ?= containerd-driver
GOLANG ?= /usr/local/go/bin/go

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
