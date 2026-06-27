# EasyEDA Agent Connector

A real, buildable **EasyEDA Pro extension** (嘉立创EDA Pro / 立创EDA Pro / EasyEDA Pro)
that bridges the **easyeda-agent Go daemon** to the official `eda.*` API over a
local WebSocket. It is the only component allowed to call `eda.*`.

```text
Skill workflow -> Go CLI/daemon (typed actions) -> THIS connector -> official eda.* API
```

Business logic, confirmation, verification, unit handling, and multi-step
orchestration live in the **Go layer and Skills**, not here. This connector only
does: transport (port-scan + handshake + register + context + heartbeat),
typed-action dispatch (one `eda.*` cluster per action), result serialization,
binary artifact transfer, and structured errors.

This extension **adapts the proven transport** from
[`eext-run-api-gateway`](https://github.com/easyeda/eext-run-api-gateway) and
replaces its raw-JS `execute` path with our typed-action dispatcher. It uses
`eda.sys_WebSocket` (NOT browser `WebSocket`/`fetch`) and type-checks against
[`@jlceda/pro-api-types`](https://www.npmjs.com/package/@jlceda/pro-api-types).

---

## File layout

```
extension/
  src/
    index.ts        # Entry point. Exports activate/deactivate + menu fns.
    transport.ts    # eda.sys_WebSocket port-scan, handshake, register, heartbeat, dispatch loop.
    actions.ts      # typed action handlers → eda.* calls + JSON serialization + artifacts.
    eda-context.ts  # Reads project/document context; document-type label mapping; editor version.
    protocol.ts     # Wire-frame types, error codes, ActionError/ActionResult.
    util.ts         # Uint8Array→base64 (no Node Buffer), Blob→base64, payload field helpers.
  config/
    esbuild.common.ts   # Shared esbuild config (format: 'iife', bundle: true, platform: 'browser').
    esbuild.prod.ts     # Build runner (supports --watch).
  build/
    packaged.ts     # Zips the extension into build/dist/<name>_v<version>.eext.
  extension.json    # EasyEDA manifest (uuid, engines.eda, activationEvents, headerMenus).
  tsconfig.json     # Strict TS, targets the @jlceda/pro-api-types ambient globals.
  package.json      # Scripts + devDependencies.
  .edaignore        # Files excluded from the packaged .eext.
```

Source is modular but esbuild **bundles every `src/*.ts` into a single IIFE**
`dist/index.js` (entry `src/index.ts`, manifest `entry: "./dist/index"`).
`dist/` is gitignored and produced by the build.

---

## Build / typecheck / package

```bash
cd extension
npm install            # esbuild, typescript, @jlceda/pro-api-types, fs-extra, jszip, ignore, ts-node, ...
npm run typecheck      # tsc --noEmit against @jlceda/pro-api-types (proves eda.* call shapes)
npm run compile        # esbuild → dist/index.js (IIFE)
npm run build          # compile + package → build/dist/easyeda-agent-connector_v<version>.eext
```

Node >= 20.17.0 is required.

### Re-importing new connector code

EasyEDA local `.eext` import is **install-only per UUID**. Once a UUID is
installed you cannot re-import a newer build with the same UUID, and a stuck one
can fail to uninstall ("无法卸载 + 装不上"). **Most code changes don't need a
re-import at all — use the `debug.exec_js` escape hatch.** Only manifest /
typed-handler changes require a rebuild; for those:

1. Stop the connector (kill the daemon so it disconnects), then **uninstall** the
   old version and import the new `.eext` — the clean path, when uninstall works.
2. If it won't uninstall, ship a **fresh UUID** (installs as a brand-new
   extension, bypassing the conflict), then fully restart EasyEDA to clear the
   old stuck one:
   ```bash
   node -e "console.log(crypto.randomUUID().replaceAll('-',''))"   # → extension.json "uuid"
   npm run release   # bump + typecheck + build
   ```

```bash
make eext            # bump patch + typecheck + build a fresh .eext
npm run bump minor   # 0.4.x -> 0.5.0
```

`scripts/bump.mjs` keeps `extension.json` and `package.json` in lock-step.

> Proven root cause (2026-06): `0.4.0` (old uuid) wouldn't install; `0.4.1`,
> byte-identical except a **fresh UUID**, installed instantly. Earlier theories —
> "patch vs minor version", "PNG vs JPG logo", "scripts/ in the zip" — were all
> **wrong**; the only blocker was the same-UUID conflict. (JPG logo + clean
> package via `.edaignore` are still good hygiene, just not the cause.)

---

## Sideloading into EasyEDA Pro

1. Run `npm run build` to produce `build/dist/easyeda-agent-connector_v0.1.0.eext`.
2. In EasyEDA Pro: open the **Extension manager** and load/import the `.eext`
   file (or point it at this `extension/` directory in a dev install).
3. **Enable required permissions** for the extension:
   - **允许外部交互 / Allow external interaction** — **REQUIRED.**
     `eda.sys_WebSocket.register/send/close` throw if this is off, so the
     connector cannot reach the daemon without it.
   - **Show in top menu** — so the `EasyEDA Agent` header menu (Reconnect / Stop /
     Toggle Auto-Connect / About) is visible.
4. Start the easyeda-agent Go daemon (it listens on one of ports 49620-49629).
5. The extension auto-connects on startup (`onStartupFinished`) when
   auto-connect is enabled; otherwise use **EasyEDA Agent → Reconnect**.

---

## WebSocket wire protocol

Transport: `eda.sys_WebSocket.register("easyeda-agent", "ws://127.0.0.1:<port>/eda", onMessage, onConnected)`.
Ports 49620-49629 are scanned; for each, the connector registers and waits
~1500ms for the daemon's `handshake`. On success the connection is kept;
otherwise it is closed and the next port is tried. All frames are JSON text
(`event.data` is a raw JSON string that we `JSON.parse` ourselves).

Frame sequence:

1. **Daemon → connector (on connect):**
   `{"type":"handshake","service":"easyeda-agent","version":"<daemon ver>"}`
   — validated: `service` must equal `"easyeda-agent"`.
2. **Connector → daemon (after valid handshake):**
   `{"type":"register","windowId":"<uuid>","connectorVersion":"0.1.0","easyedaVersion":"<eda>","capabilities":["schematic.v1"]}`
   (`windowId` via `crypto.randomUUID()`).
3. **Connector → daemon (best-effort):**
   `{"type":"context","windowId":"...","projectUuid":"...","projectName":"...","documentUuid":"...","documentType":"schematic","tabId":"..."}`
   — empty fields are omitted.
4. **Heartbeat:** connector sends `{"type":"ping","id":"hb-<n>"}` every 3s.
   Liveness is **consecutive-miss based**, not a single round-trip deadline: only
   after 3 pings go unanswered in a row (~9s of true silence) is the socket
   considered dead → reconnect. A single lagged pong (EasyEDA's webview stalls on
   canvas redraw / GC) does NOT tear the socket down. A daemon-initiated
   `{"type":"ping","id":...}` is answered with `{"type":"pong","id":...}`.
5. **Daemon → connector:**
   `{"type":"request","id":"req_N","version":"v1","action":"<action>","payload":{...},"windowId":"..."}`.
6. **Connector → daemon (echoing `id`):**
   `{"type":"response","id":"req_N","version":"v1","ok":true,"result":{...},"context":{...},"artifacts":[...],"warnings":[...]}`
   or on failure
   `{"type":"response","id":"req_N","version":"v1","ok":false,"error":{"code":"...","message":"...","detail":"<original eda error>"}}`.
7. **Connector → daemon (diagnostics, best-effort):**
   `{"type":"log","msg":"<connection-lifecycle event>"}` — reconnect reasons and
   register attempts, surfaced in the daemon log as `connector LOG: …`. Deliberately
   NOT emitted per ping/pong, to keep the log readable.

Auto-reconnect **never permanently gives up**. Up to 5 fast retries 3s apart, then
it falls back to a quiet 10s background poll (announced once, then silent) so a
daemon started/restarted later auto-connects with no manual **Reconnect**. The
liveness loop also reconnects on a `send` throw (the underlying socket is gone).

---

## Action → `eda.*` mapping (19 connector-dispatched actions)

All `eda.*` calls are `await`ed. Component fields are read via `getState_*()`
accessors. Coordinates are passed through from the payload unchanged (unit
handling is the daemon/skill's concern).

| Action | `eda.*` call(s) |
| --- | --- |
| `project.current` | `dmt_Project.getCurrentProjectInfo()` → `{uuid,name,friendlyName,teamUuid,description}` |
| `document.current` | `dmt_SelectControl.getCurrentDocumentInfo()` → maps numeric `documentType` → label |
| `schematic.pages.list` | `dmt_Schematic.getAllSchematicsInfo()` + `getAllSchematicPagesInfo()` |
| `schematic.page.open` | `dmt_EditorControl.openDocument(schematicPageUuid)` → `{tabId}` |
| `schematic.components.list` | `sch_PrimitiveComponent.getAll(undefined, allPages)`; optional `getAllPinsByPrimitiveId(id)` when `includePins:true` |
| `schematic.component.place` | `sch_PrimitiveComponent.create({libraryUuid,uuid}, x, y, subPartName?, rotation?, mirror?, addIntoBom?, addIntoPcb?)` |
| `schematic.component.modify` | `sch_PrimitiveComponent.modify(primitiveId, patch)` |
| `schematic.component.delete` | `sch_PrimitiveComponent.delete(primitiveIds)` → `{deleted}` |
| `schematic.wire.create` | `sch_PrimitiveWire.create(points, net?, color?, lineWidth?, lineType?)` |
| `schematic.netflag.create` | branches on `kind` (see below) |
| `schematic.power.connect_pin` | composite: `sch_PrimitiveWire.create([pinX,pinY,endX,endY])` then `createNetFlag`/`createNetPort` at the far end. Default offset 30u; flag body oriented outward along the stub. |
| `schematic.library.search` | `lib_Device.search(query)` → first `limit` results (default 10), each mapped to `{libraryUuid, uuid, name, value, footprintName, lcsc, description}` |
| `schematic.select` | `sch_SelectControl.doSelectPrimitives(primitiveIds)` then `getAllSelectedPrimitives_PrimitiveId()` |
| `schematic.snapshot` | `dmt_EditorControl.getCurrentRenderedAreaImage(tabId?)` → Blob → artifact |
| `schematic.drc.check` | `sch_Drc.check(strict, false, includeVerboseError)` → normalized `{passed, fatal, summary, violations:[{level,rule,message,primitiveIds,designators,x,y,raw}]}` (per-violation, not aggregate) |
| `schematic.save` | `sch_Document.save()` → `{saved}` |
| `schematic.export.netlist` | `sch_ManufactureData.getNetlistFile(fileName?, netlistType?)` → File → artifact |
| `schematic.export.bom` | `sch_ManufactureData.getBomFile(fileName?, fileType, template?, filterOptions?, statistics?, property?, columns?)` → File → artifact |
| `debug.exec_js` | runs raw `eda.*` JS via an `AsyncFunction('eda', code)` — escape hatch, confirmation-gated |

> Note: the catalog has **20** typed actions (`make actions`). `system.health` is
> handled by the Go daemon itself (daemon/connector liveness, no window) and is not
> dispatched to the connector, so the connector's handler map is the **19** above.
>
> `schematic.library.search` returns EasyEDA's native result order truncated to
> `limit` — it does NOT rerank, despite the action description's "ranked list"
> wording. See [docs/FEATURES.md](../docs/FEATURES.md) (optimized-search roadmap).

### `schematic.netflag.create` — payload `kind` → API mapping

| `payload.kind` | API call | identification / direction |
| --- | --- | --- |
| `power` | `createNetFlag` | `'Power'` |
| `ground` | `createNetFlag` | `'Ground'` |
| `analog_ground` | `createNetFlag` | `'AnalogGround'` |
| `protective_ground` / `protect_ground` | `createNetFlag` | `'ProtectGround'` |
| `net_port_in` | `createNetPort` | `'IN'` |
| `net_port_out` | `createNetPort` | `'OUT'` |
| `net_port_bi` | `createNetPort` | `'BI'` |
| `short_circuit` | `createShortCircuitFlag` | — (no net) |

`net` is required for `createNetFlag` and `createNetPort`; `short_circuit` takes
only `x, y, rotation?, mirror?`.

---

## Artifact transfer (snapshot / netlist / bom)

`getCurrentRenderedAreaImage` returns a **Blob**; `getNetlistFile`/`getBomFile`
return a **File**. The connector cannot write to the daemon's disk, so it reads
the bytes, base64-encodes them (a manual `Uint8Array → base64` helper in
`util.ts`; no Node `Buffer`, no `btoa`), and returns them inline:

```json
{
  "id": "art_<uuid>",
  "kind": "schematic_snapshot | schematic_netlist | schematic_bom",
  "mimeType": "<blob.type or inferred>",
  "fileName": "<name.ext>",
  "inlineBase64": "<base64 bytes>"
}
```

The **daemon** decodes `inlineBase64`, writes the file, and fills
`path`/`size`/`sha256`. The connector only produces
`{id, kind, mimeType, fileName, inlineBase64}`.

---

## Error handling

Handlers throw `ActionError(code, message, detail)`; the original `eda.*` error
message is preserved in `error.detail`. Stable codes:

- `UNKNOWN_ACTION` — no handler for the action name.
- `MISSING_PAYLOAD_FIELD` — a required payload field is missing/invalid.
- `EDA_API_UNAVAILABLE` — the global `eda` object is not present.
- `EDA_CALL_FAILED` — an `eda.*` call threw or returned no result.
- `INTERNAL_ERROR` — an unexpected non-ActionError was thrown.

---

## Menu actions (`extension.json` → exported fns)

| Menu item | Exported fn |
| --- | --- |
| Reconnect | `reconnect` |
| Stop | `stopConnection` |
| Toggle Auto-Connect | `toggleAutoConnect` |
| About... | `about` |

`activate()` (auto-start on `onStartupFinished`) and `deactivate()` (cleanup)
are also exported. Auto-connect preference is stored via
`eda.sys_Storage.get/setExtensionUserConfig("autoConnectEnabled")`.

---

## What remains uncertain

- **Artifact transfer is an assumption.** The protocol carries bytes inline as
  base64 because the connector has no filesystem access to the daemon's machine.
  This works but is memory-heavy for large BOM/netlist/snapshot files; a future
  chunked/streamed transfer may be warranted. The daemon must implement the
  `inlineBase64` → file decode side.
- **Coordinate units.** Coordinates are passed through verbatim. Per the SDK,
  schematic canvas units span `0.01 inch`. Unit interpretation/conversion is the
  daemon/skill's responsibility, not the connector's.
- **DRC violation shape.** `sch_Drc.check(..., true)` returns `Array<any>` — the
  SDK does not type the violation objects, and the shape differs by domain
  (schematic ships flat aggregates `[{count, type}]`; PCB nests
  `[{name, list:[{name, list:[{errorType,…}]}]}]`). The handler **normalizes**
  both: `flattenDrcNodes` walks any `list` containers into per-violation leaves and
  projects each to `{level, rule, message, primitiveIds, designators, x, y}` while
  keeping the original under `raw`. The field projection is best-effort over the
  untyped shape — **verify the real per-item fields against a connected window**
  (run `easyeda sch drc --json` on a board with known violations) and widen the key
  lists in `flattenDrcNodes` if a build names them differently. An empty
  `violations` array (and `passed:true`) means DRC passed.
- **`easyedaVersion`** is read from `eda.sys_Environment.getEditorCurrentVersion()`
  (best-effort; falls back to `""`).
- **`eda.sch_PrimitiveComponent` is a union type** (`SCH_PrimitiveComponent |
  SCH_PrimitiveComponent3`) in the SDK; both members expose identical method
  shapes, so calls type-check cleanly. Component/pin primitive types are derived
  from the API return types (`Awaited<ReturnType<...>>`) rather than the
  internal `$1`-suffixed class names.
- **Net-flag library devices.** `createNetFlag`/`createNetPort` rely on the
  EasyEDA defaults; if a project needs custom flag symbols, the
  `setNetFlagComponentUuid_*` / `setNetPortComponentUuid_*` setters would need
  wiring (not exposed as actions in Phase 1).
