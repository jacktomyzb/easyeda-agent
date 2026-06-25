.PHONY: test fmt actions build daemon eext connector

test:
	go test ./...

fmt:
	gofmt -w cmd internal

actions:
	go run ./cmd/easyeda actions

build:
	go build -o bin/easyeda ./cmd/easyeda

daemon:
	go run ./cmd/easyeda daemon

# Build the connector .eext at the CURRENT version (no bump).
connector:
	npm --prefix extension run build

# Bump the connector version + build a fresh, importable .eext.
# EasyEDA refuses to re-import an .eext whose (uuid, version) is already
# installed, so use this whenever the user needs to load new connector code.
eext:
	npm --prefix extension run release
