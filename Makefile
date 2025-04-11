.PHONY: all lint build build-race generate-proto test clean unit-test test-integration test-all test-integration-specific

BINDIR      := bin
CMD_CLI     := cmd/cli/main.go
CMD_SERVER  := cmd/server/main.go
PROTO_DIR   := proto
PROTO_FILE  := $(PROTO_DIR)/task.proto
GEN_DIR     := gen

all: lint build test

lint:
	golangci-lint run

build:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/taskman $(CMD_CLI)
	go build -o $(BINDIR)/taskman-server $(CMD_SERVER)

build-race:
	@mkdir -p $(BINDIR)
	go build -race -o $(BINDIR)/taskman $(CMD_CLI)
	go build -race -o $(BINDIR)/taskman-server $(CMD_SERVER)

generate-proto:
	@mkdir -p $(GEN_DIR)
	protoc --go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_FILE)

unit-test:
	# Run unit tests as non-root
	@echo "==> Running unit tests (non-root)"
	go test -v -race -cover $$(go list ./... | grep -v 'github.com/mikewurtz/taskman/tests$$')

test-integration:
	# Run integration tests as root since cgroup creation is privileged
	@echo "==> Running privileged tests (as root)"
	go test -c -race -o tests/testbin ./tests/integration && \
	sudo ./tests/testbin -test.v

test-all: unit-test test-integration

test-integration-specific:
	@if [ -z "$(FUNC)" ]; then \
		echo "Error: specify the test function with FUNC=<TestName>"; \
		exit 1; \
	fi; \
	go test -c -o tests/testbin ./tests/integration && \
	sudo ./tests/testbin -test.v -test.run "^$(FUNC)$$"

clean:
	rm -rf $(BINDIR) $(GEN_DIR)
