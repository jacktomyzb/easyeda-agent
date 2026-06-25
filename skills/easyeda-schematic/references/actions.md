# EasyEDA Schematic Action Reference

This reference mirrors the Phase 1 action set. Prefer the CLI's `easyeda actions` output when available.

## Read Context

- `system.health`
- `project.current`
- `document.current`
- `schematic.pages.list`
- `schematic.page.open`

## Inspect Schematic

- `schematic.components.list`
- `schematic.select`
- `schematic.snapshot`

## Mutate Schematic

- `schematic.component.place`
- `schematic.component.modify`
- `schematic.component.delete`
- `schematic.wire.create`
- `schematic.netflag.create`
- `schematic.save`

## Verify and Export

- `schematic.drc.check`
- `schematic.export.netlist`
- `schematic.export.bom`

## Confirmation Required

- `schematic.component.delete`
- `schematic.save` when save was not explicit
- generated multi-action mutation plans
