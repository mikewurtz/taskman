.PHONY: all lint build build-race generate-proto test clean

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

test:
	go test -race -v ./...

clean:
	rm -rf $(BINDIR) $(GEN_DIR)
