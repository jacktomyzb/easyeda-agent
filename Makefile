.PHONY: test fmt actions

test:
	go test ./...

fmt:
	gofmt -w cmd internal

actions:
	go run ./cmd/easyeda actions
