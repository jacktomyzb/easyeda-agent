# Changelog

All notable changes to the **EasyEDA Agent Connector** are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/); versions
follow [SemVer](https://semver.org/).

## [Unreleased]
### Fixed
- **`schematic.pin.set_no_connect` no longer reports a false success.** On EasyEDA
  Pro 3.2.x, `pin.setState_NoConnected` is a **no-op** — the pin primitive has no
  `noConnected` field (verified by re-pull, DRC re-run, and a canvas snapshot: no
  非连接标识 is ever placed and DRC still treats the pin as floating). The setter is
  typed `@public`, so the prior implementation compiled and returned `ok` while
  silently doing nothing. The handler now **verifies** the write and fails with
  `EDA_CALL_FAILED`, naming it as an EasyEDA platform limitation (not a connector
  defect) and returning `notApplied[]`. It auto-passes if a future build makes the
  setter real. There is no public `eda.*` API to place a 非连接标识 on this version —
  use `schematic.check` to enumerate floating pins.
- **`schematic.wire.create` now normalizes nested `points` (issue #5).** EDA's
  `eda.sch_PrimitiveWire.create` only accepts a **flat** `number[]`
  (`[x1,y1,x2,y2,…]`); a nested `[[x,y],…]` payload failed with
  `EDA_CALL_FAILED / "create failed!"`. The connector now flattens nested points
  at a single source of truth (`normalizeWirePoints` in `util.ts`), so CLI /
  `call` / sch.py / `debug.exec_js` all accept either form. Also validates the
  list is an even-length (`≥4`) run of finite numbers. CLI `sch wire --help` and
  `auto-layout-sop.md` updated to document both forms.

### Added
- **`schematic.check` — reconstructed per-item design check.** The EDA schematic DRC
  API (`eda.sch_Drc.check`) returns only an aggregate `{count,type}`; the per-item
  detail the UI DRC panel shows is not exposed by any public API. `schematic.check`
  recomputes the actionable findings from primitives. Rule 1: **floating pins** via
  geometric connectivity (a pin is connected iff a wire touches its coordinate;
  NC-marked pins excluded), grouped by component as `{designator, pins[]}` — the
  exact input `schematic.pin.set_no_connect` takes. Returns `{passed, summary,
  findings[]}`. CLI: `easyeda sch check` (`--json`, `--strict`, `--all-pages`).
  Verified live: 34 floating pins on an ESP32-S3, matching the EDA DRC panel.
- **`schematic.drc.check` now returns per-violation detail.** Normalizes the SDK
  result into `{passed, fatal, summary, violations[]}` — each violation projects
  `{level, rule, message, primitiveIds, designators, x, y}` (raw kept) plus a
  severity summary and a `fatal` count for the design-flow S5 gate. CLI `sch drc`
  prints one line per violation and exits non-zero only when `fatal > 0`. NOTE: the
  schematic SDK only provides an aggregate, so detail degrades honestly to
  "N issue(s) — EDA returned no per-item detail" (use `schematic.check` for the
  itemized floating-pin findings).
- **`schematic.snapshot` anti-stale metadata (issue #2).** The snapshot result now
  carries `primitiveCount` (live components + page primitives on the current page),
  `capturedAt` (ISO timestamp), and a `stale` advisory string. EasyEDA does not
  auto-redraw after `eda.*` edits, so `getCurrentRenderedAreaImage` can return a
  byte-identical STALE frame; callers compare `primitiveCount` across two snapshots
  to detect when the image didn't change but the page did. Judge state by data, use
  the screenshot for layout only.
- **`schematic.page.clear` — one-shot page reset.** Deletes every page-level
  primitive on the active page (components, net flags/ports/labels, wires, buses,
  and graphics — arcs/circles/rectangles/polygons/text), not just components.
  `preserveSheet` (default true) keeps the sheet/title block; `dryRun` reports
  per-type counts without deleting. Returns `{deleted:{...}, total, deletedIds}`.
  Fixes the trap where `schematic.component.delete` left wires/buses behind while
  `components.list` reported a clean page, forcing a fall back to raw
  `debug.exec_js`.
- **`schematic.primitives.delete` — generalized, any-type delete.** Routes each
  requested id to its owning `sch_Primitive*` class so wires/buses/graphics/flags
  can be deleted alongside components; omit `primitiveIds` to delete the current
  selection (select-all → delete). Reports `notFound` ids.

## [0.5.8] - 2026-06-27
### Changed
- **Version bump to pair with the daemon/CLI artifact-path change.** No connector
  behavior change vs 0.5.7. The CLI now sends its working directory and the daemon
  writes artifacts (snapshots, netlist/BOM exports) under `<cwd>/.easyeda/artifacts`
  with sortable timestamped names (`<YYYYMMDD-HHMMSS>-<kind>-<short>.ext`) instead
  of a flat `artifacts/art_<uuid>` in the daemon's cwd. Released together to keep
  CLI and connector on the same version.

## [0.5.7] - 2026-06-27
### Added
- **Heartbeat-carried context.** The connector now re-reads the active
  project/document on each heartbeat (~3s) and pushes it to the daemon only when
  it changed. `easyeda daemon health` (and project routing) now reflect a UI
  tab-switch within one interval — previously context refreshed only on connect
  or as a side effect of running an action, so it lagged the UI until the next
  command. The initial post-connect push is unconditional; reconnects reset the
  change-detection signature so they always re-push.

## [0.5.6] - 2026-06-27
### Changed
- **Rebuild to pair with the daemon's live-context + `doc` work.** No connector
  behavior change vs 0.5.5 — this build exists so a window stuck on a stale
  connector can be re-imported to pick up real version reporting + port-scan
  (49620–49629). The daemon now refreshes each window's context from every action
  response (so `health` no longer reads `home` forever) and `easyeda doc ls/switch`
  drives the discover→switch loop on top of the existing `document.open` action.

## [0.5.5] - 2026-06-27
### Fixed
- **Handshake reports the real connector version.** `connectorVersion` was a
  hardcoded `0.1.0`, so `easyeda daemon health` could not reveal which build a
  window was actually running — useless for spotting a stale open window. esbuild
  now injects `extension.json`'s version at build time (`__CONNECTOR_VERSION__`).

## [0.5.4] - 2026-06-27
### Added
- **Board (板子/组合) management** — `board.list`, `board.current`, `board.create`,
  `board.rename`, `board.copy`, `board.delete`. A Board binds one schematic + one
  PCB; these expose `eda.dmt_Board.*` so the schematic↔PCB grouping is editable
  (and a floating PCB can be linked before `import_changes`).

## [0.5.3] - 2026-06-27
### Added
- **Schematic page management** — `schematic.page.create`, `schematic.page.rename`,
  `schematic.page.delete`, `schematic.rename` (`eda.dmt_Schematic.*`).
- **明细表 (title block)** — `schematic.titleblock.get` / `schematic.titleblock.modify`
  to read and adjust the drawing-sheet title block (the editable "图纸" surface;
  EasyEDA Pro exposes no set-paper-size API).

## [0.5.2] - 2026-06-27
### Added
- **Editor view shortcuts** — `view.fit` (适应全部 / `K`), `view.fit_selection`
  (适应选中), `view.zoom`, `view.region` via `eda.dmt_EditorControl.*`; act on the
  focused canvas, shared by schematic and PCB.

## [0.5.1] - 2026-06-26
### Added
- **PCB layout intelligence** — `pcb.components.arrange` (cluster / grid auto-layout
  seed) and rendered bounding boxes in `pcb.components.list`.
- **PCB layout adjustment** — `pcb.align`, `pcb.distribute`, `pcb.grid_snap`,
  `pcb.components.move`.
- **Board outline (板框)** — `pcb.outline.set` / `pcb.outline.get` / `pcb.outline.clear`.
- **PCB DRC** — `pcb.drc.check`, normalized to `{passed, violations}`.

## [0.4.10] - 2026-06-26
### Added
- `homepage` pointing at the GitHub repository (open-source link for the listing),
  as a plain URL (no `#readme` fragment).

## [0.4.9] - 2026-06-26
### Fixed
- Marketplace manifest finalized: `repository.type` is `github` (per the official
  `eext-extension-demo`); removed the optional `bugs`/`homepage` fields — the
  marketplace flagged the `bugs` content and neither field is required. No email
  or other private data ships in the `.eext`.

## [0.4.5] - 2026-06-26
### Added
- `repository` field in the manifest and this `CHANGELOG.md` (marketplace
  submission requirements).

## [0.4.4] - 2026-06-26
### Changed
- Release tooling keeps a **stable UUID** by default, so a new version updates in
  place (uninstall the old entry, then import); a fresh-UUID build is now an
  explicit fallback. No change to the connector's runtime behaviour.

## [0.4.2] - 2026-06-26
### Fixed
- **Self-healing reconnection.** The connector no longer permanently gives up
  after a few failed retries. After the initial fast attempts it falls back to a
  quiet background poll, so a daemon that is started or restarted *after* the
  editor auto-connects with no manual **Reconnect**. A connection lost to a daemon
  restart also recovers on its own.

## [0.4.0] - 2026-06-26
### Fixed
- `.eext` packaging so the extension installs reliably. Bundled a JPEG logo.

## [0.3.0] - 2026-06-26
### Fixed
- Netflag / netport **orientation**: corrected for EasyEDA's y-up coordinate
  system and fixed rotation handling in `connect_pin` (reverted a wrong rotation
  negation).

## [0.2.0] - 2026-06-25
### Added
- Initial connector: a WebSocket bridge (port-scans 49620–49629) to the
  easyeda-agent Go daemon, dispatching typed schematic actions to the official
  `eda.*` API, with auto-reconnect and a heartbeat.
- `connect_pin` composite action and the netflag/netport orientation convention.
- Header menu: **Reconnect**, **Stop**, **Toggle Auto-Connect**, **About**.
