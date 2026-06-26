.PHONY: test fmt actions build daemon dev eext eext-fresh connector lint-test

test:
	go test ./...

# Rule-trust harness for the schematic linter: orientation-table consistency
# (orientation.json derives to its frozenTable; matches the connector) +
# fixture goldens (known-good board stays clean, known-bad cases still fire).
lint-test:
	python3 tools/schematic-lint/tests/run.py

fmt:
	gofmt -w cmd internal

actions:
	go run ./cmd/easyeda actions

build:
	go build -o bin/easyeda ./cmd/easyeda

daemon:
	go run ./cmd/easyeda daemon

# Live-reload the daemon for development (.air.toml): rebuilds + restarts on any
# .go change; the connector auto-reconnects (it port-scans 49620-49629). Keep
# this running in a terminal while developing so the daemon is always up.
dev:
	@command -v air >/dev/null 2>&1 || { echo "air not found — install: go install github.com/air-verse/air@latest"; exit 1; }
	air

# Build the connector .eext at the CURRENT version (no bump).
connector:
	npm --prefix extension run build

# Cut an importable connector .eext (default: STABLE uuid). Bump PATCH + typecheck
# + build. EasyEDA dedups installed extensions by uuid, so to load this you update
# in place: uninstall the old one in EasyEDA's 已安装 tab, then import the printed
# .eext. Keeps ONE extension entry. Use `make eext-fresh` only if the installed
# one won't uninstall.
eext:
	node extension/scripts/bump.mjs patch
	npm --prefix extension run typecheck
	npm --prefix extension run build
	@printf '\n✅ uninstall old in 已安装, then import → extension/build/dist/easyeda-agent-connector_v%s.eext\n' "$$(node -p "require('./extension/extension.json').version")"

# Fallback only: mint a FRESH uuid so it imports as a NEW extension with no
# uninstall — but it leaves a duplicate "EasyEDA Agent" entry you must delete
# afterward (else multiple connectors fight over the daemon).
eext-fresh:
	node extension/scripts/bump.mjs patch --uuid
	npm --prefix extension run typecheck
	npm --prefix extension run build
	@printf '\n✅ fresh-uuid build → import extension/build/dist/easyeda-agent-connector_v%s.eext, then DELETE the old entry\n' "$$(node -p "require('./extension/extension.json').version")"
