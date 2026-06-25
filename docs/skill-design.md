# Skill Design

The Phase 1 Skill should guide agents to call typed CLI actions rather than generating EasyEDA JavaScript.

## Skill Responsibilities

- Check connection before work.
- Read active project/document context.
- Decide whether the current document is a schematic page.
- Prefer additive operations unless the user asks for destructive changes.
- Ask for confirmation before deletion, save, or multi-step mutation plans.
- Verify mutations with readback and snapshots.
- Run DRC before claiming schematic work is complete.
- Export BOM/netlist when the user asks for deliverables.

## Example Workflow

```text
User: Add a 10k pull-up resistor from NET_A to 3V3.

Agent:
1. easyeda health
2. easyeda schematic context
3. easyeda schematic components list
4. resolve or ask for resistor library identity
5. easyeda schematic component place ...
6. easyeda schematic wire create ...
7. easyeda schematic netflag create ...
8. easyeda schematic snapshot
9. summarize the result and ask whether to save
```

## Missing Action Strategy

If a needed typed action is missing:

1. Check whether the work can be decomposed into existing typed actions.
2. If not, explain the missing action.
3. Use raw JavaScript only for exploration or a clearly bounded debug task.
4. Promote repeated raw JavaScript patterns into a typed action.
