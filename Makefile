.PHONY: build typecheck test lint check

build:
	go build -o bin/cleanup ./cmd/cleanup

typecheck:
	go build ./...

test:
	go test -race ./...

lint:
	go vet ./...
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)

check: typecheck test lint
