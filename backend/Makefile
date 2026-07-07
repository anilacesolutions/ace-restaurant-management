.PHONY: run build test tidy lint

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

lint:
	go vet ./...
