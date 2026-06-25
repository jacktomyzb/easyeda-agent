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

The flag body must point outward along the stub direction. The rotation cycle is
`up → left → down → right` per +90° (anchored on ESP32 reference samples:
PWR stored-rot=90 → body left; GND stored-rot=270 → body left). Body at rot 0:
power=up, ground=down, net_port=right. See
[docs/schematic-layout-conventions.md §3.5](../../docs/schematic-layout-conventions.md).

> Note: `getState_Rotation()` reports the STORED rotation; `createNetFlag`/
> `createNetPort` NEGATE their input (`stored = (360 - input) % 360`). The linter
> works in stored space; `schematic.power.connect_pin` converts on write.

## Files

- `probe.js` — the one-shot data pull (runs via `debug.exec_js`)
- `lint.py` — the analyzer (`lint.py <layout.json>`)
- `lint.sh` — resolves the live window, pulls, and lints

This is a candidate to promote into a typed `schematic.lint` action.
