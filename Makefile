.PHONY: run build test lint fmt check

run:
	set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/bot

build:
	go build -o bin/bot ./cmd/bot

test:
	go test -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w . && go mod tidy

check: fmt test lint
