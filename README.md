# easyeda-agent

AI-native automation layer for EasyEDA.

`easyeda-agent` turns the official EasyEDA extension API into a typed, observable, Skill-friendly system. The EasyEDA plugin stays thin: it connects to the local agent and executes approved actions. The Go CLI/daemon owns protocol, state, artifacts, validation, and user-facing workflows.

## Why This Exists

The upstream `run-api-gateway` proves the important entry point: code can run inside EasyEDA with access to the official `eda` object. Its rough edge is that it exposes raw JavaScript execution as the main workflow. That is powerful, but brittle for agents.

This project moves the system into a better shape:

- Skill describes expert workflow and guardrails.
- Go CLI/daemon exposes stable typed actions.
- EasyEDA connector plugin only bridges to official `eda.*` APIs.
- Artifacts, screenshots, DRC results, and audit logs are first-class outputs.

## Phase 1 Scope

Phase 1 focuses on schematic workflows:

- connect to an active EasyEDA window
- read project and current document context
- list schematic pages
- list, place, modify, and delete schematic components
- create wires, net labels, ports, power flags, and ground flags
- select and inspect primitives
- run schematic DRC
- save schematic changes
- export schematic netlist and BOM artifacts
- capture schematic viewport snapshots for verification

PCB, footprint, manufacturing, and library authoring are intentionally deferred.

## Repository Layout

```text
cmd/easyeda/                 CLI entrypoint used by humans and Skills
internal/app/                CLI command implementation
internal/daemon/             Future local daemon boundary
internal/protocol/           Typed action protocol shared with connector
internal/version/            Build/version metadata
extension/                   EasyEDA connector notes and future source
skills/easyeda-schematic/    Phase 1 Skill draft
docs/                        Architecture, protocol, roadmap, decisions
```

## Current Commands

```bash
go run ./cmd/easyeda version
go run ./cmd/easyeda phase1
go run ./cmd/easyeda actions
go run ./cmd/easyeda health
```

`health` scans `127.0.0.1:49620-49629` for an `easyeda-agent` daemon-style health endpoint. The daemon is not implemented yet, so a clean `not_found` result is expected at this stage.

## Design Position

Raw JavaScript execution remains useful for debugging, but not as the primary AI surface. The default surface should be typed actions with explicit inputs, predictable outputs, artifact handling, and verification hooks.

See:

- [Phase 1 schematic scope](docs/phase-1-schematic.md)
- [Architecture](docs/architecture.md)
- [Protocol](docs/protocol.md)
- [Skill design](docs/skill-design.md)
