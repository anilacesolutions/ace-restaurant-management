.PHONY: run build test tidy

run:
	go run ./cmd/bridge

build:
	go build -o bin/bridge ./cmd/bridge

test:
	go test ./...

tidy:
	go mod tidy
