
lint:
	golangci-lint run

build:
	go build -o bin/taskman cmd/cli/main.go
	go build -o bin/taskman-server cmd/server/main.go

build-race:
	go build -race -o bin/taskman cmd/cli/main.go
	go build -race -o bin/taskman-server cmd/server/main.go

generate-proto:
	protoc --go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		proto/task.proto

test:
	go test -race -v ./...