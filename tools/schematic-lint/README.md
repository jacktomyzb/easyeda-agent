# schematic-lint — data-only schematic checker

Find layout/connectivity problems in a live EasyEDA schematic **from data, not
screenshots**. One `getAll` + `wire.getAll` pull returns the entire global
layout (≈600ms even for a 53-part board); everything else is local analysis.

```bash
make build
bin/easyeda daemon &                 # connector must be connected
tools/schematic-lint/lint.sh ceshi   # project name (default: ceshi)
```

## Why data, not screenshots

`getCurrentRenderedAreaImage` is for final human eyeballing. For *finding*
problems it is slow and lossy. The connector can return the full primitive set
(components, pins, bboxes, netflags, netports, wires with coordinates) in a
single dispatch, and a geometry/union-find pass finds the issues deterministically
with exact coordinates. Screenshots are only used to confirm a fix at the end —
and for that, use `eda.dmt_EditorControl.zoomToAllPrimitives()` /
`zoomToSelectedPrimitives()` before snapshotting (NOT `navigateToRegion`, which
does not zoom the rendered-area image).

## Checks

| id | severity | what |
|---|---|---|
| `flag_on_pin` | 🔴 | netflag/netport at the exact pin coordinate → DRC fatal (EasyEDA needs a real wire, never overlap) |
| `zero_wire` | 🔴 | zero-length wire segment |
| `dangling_wire` | 🔴 | wire endpoint (degree 1) with no pin/flag — 空连 |
| `floating_pin` | 🟠 | pin with no wire and no flag |
| `single_pin_net` | 🟠 | a pin whose net has only itself and no power/ground/label |
| `flag_no_wire` | 🟠 | netflag/netport with no wire connected |
| `orientation` | 🟡 | flag rotation not 顺着导线 (body should point outward along the stub) |
| `bbox_overlap` | 🟠 | two parts whose pin-bboxes overlap |
| `dup_designator` | 🟠 | duplicate designator |
| `netport_hop` | 🟡 | same-net net-ports < 300u apart on one page (should be a wire/label) |
| `collinear_flags` | 🟡 | different-net flags collinear through a component (visual false-short) |
| `unnamed_net` | 🔵 | multi-pin signal net with no label/rail |
| `off_grid` | 🔵 | coordinate not on the 5-unit grid |

## How the orientation check works

The flag body must point outward along the stub direction. The whole rotation
table is **derived from four facts** — the `up → left → down → right` +90° cycle
and the body direction at rot 0 per family (power=up, ground=down, net_port=right).
Those four facts live in [`orientation.json`](orientation.json), the **single
source of truth**: `orient.py` derives the table for this linter, and the
connector's `connect_pin` ([actions.ts](../../extension/src/actions.ts)
`deriveBodyRotation()`) derives the *same* table for what it writes. They can't
drift — the harness asserts it. See
[docs/schematic-layout-conventions.md §3.5](../../docs/schematic-layout-conventions.md).

## Rule-trust harness — `make lint-test`

A data-driven linter is only as trustworthy as its rules; a wrong rule is itself
a bug. Two guards keep verdicts honest (run `make lint-test` or
`python3 tools/schematic-lint/tests/run.py`):

1. **Orientation consistency** — `orientation.json` must derive back to its own
   `frozenTable`, and the +90° cycle law must hold. This is what stops the
   Python check and the TS writer from silently diverging. To re-validate the
   anchors against *live* ground truth, run [`calibrate.js`](calibrate.js) via
   `debug.exec_js` against a connected window — it creates a flag at each
   rotation, reads the body direction from the bbox-center offset, and compares
   to `orientation.json` (do this after importing a new `.eext`).
2. **Fixture goldens** — every layout under `tests/fixtures/` is linted and
   diffed against `tests/golden/`. `clean_board.json` MUST stay clean (the
   false-positive net); each bad fixture MUST still fire its rule. After an
   intentional rule change, re-freeze with `tests/run.py --update`.

> Notes: `createNetFlag`/`createNetPort` rotation is **identity** (read back ===
> written; no negation — an earlier "negation" finding was wrong). EasyEDA is
> **y-up** (+y renders upward), so `direction()` treats `dy>0` as up. Ground-truth
> a flag's body direction with `sch_Primitive.getPrimitivesBBox([pid])` — the bbox
> center's offset from the placement point is the body direction (pure data).

## Files

- `probe.js` — the one-shot data pull (runs via `debug.exec_js`)
- `lint.py` — the analyzer (`lint.py <layout.json>`)
- `lint.sh` — resolves the live window, pulls, and lints
- `orientation.json` — canonical orientation facts (single source of truth)
- `orient.py` — derives the body-rotation table from the spec
- `calibrate.js` — live bbox ground-truth check for the orientation anchors
- `tests/run.py` + `tests/fixtures/` + `tests/golden/` — the rule-trust harness

This is a candidate to promote into a typed `schematic.lint` action.
