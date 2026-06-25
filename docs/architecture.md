# Architecture

## Components

```text
AI Agent / Human
  |
  | calls CLI commands or future local HTTP API
  v
Go CLI / Daemon
  |
  | typed action protocol over WebSocket
  v
EasyEDA Connector Extension
  |
  | official JavaScript extension API
  v
EasyEDA eda object
```

## Go CLI / Daemon

Responsibilities:

- expose ergonomic commands for Skills and humans
- scan and manage local daemon ports
- track connected EasyEDA windows
- own the typed action schema
- validate action inputs before dispatch
- normalize responses and errors
- persist artifacts such as snapshots, netlists, and BOM files
- write audit logs
- keep raw JavaScript execution behind a debug command

The CLI starts as a direct command surface. A daemon will be added when we need long-lived WebSocket sessions, artifact storage, and multi-window state.

## EasyEDA Connector Extension

Responsibilities:

- connect to the Go daemon
- register a stable `windowId`
- report active project/document context
- translate typed actions to `eda.*` API calls
- serialize primitive objects into JSON-friendly state
- stream or upload `File`/`Blob` results as artifacts
- return structured errors with original EasyEDA messages

The connector should stay small. Business process belongs in Skills and Go action orchestration.

## Skill Layer

Responsibilities:

- choose the right action sequence for a schematic task
- ask for confirmation before destructive operations
- verify after mutations
- interpret DRC results and propose fixes
- avoid raw JavaScript unless a typed action is missing

## Why Not Raw JS First

The upstream gateway executes arbitrary JavaScript inside EasyEDA. That proves feasibility, but it makes AI calls fragile:

- generated code can use wrong method names or wrong current context
- `File` and `Blob` results do not survive JSON serialization well
- errors lack domain-specific recovery guidance
- destructive actions have no standard confirmation or audit path

Typed actions solve those problems without removing the debug escape hatch.
