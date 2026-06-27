# Skills

Two kinds of skill, deliberately split so each can be maintained on its own while
staying tightly cross-referenced.

| Skill | Kind | Holds |
|---|---|---|
| [`easyeda-conventions`](easyeda-conventions/SKILL.md) | **Reference** (no actions) | The tool-agnostic EE design truth — schematic/PCB layout conventions, part-selection criteria, and the canonical data (`orientation.json`, `standard-parts.json`). |
| [`easyeda-design-flow`](easyeda-design-flow/SKILL.md) | **Orchestration** | The chief-EDA-engineer **process spine** for a whole board: the staged, gated pipeline (pre-analysis → paginate → group → place-by-group → route → DRC + layout-lint → adjust loop). Sequences and gates; delegates actions to the operational skills and rules to conventions — copies neither. |
| [`easyeda-schematic`](easyeda-schematic/SKILL.md) | **Operational** | How to drive `easyeda-agent` for **schematics**: the typed-action workflow, scripts (`lint`, `bom-enrich`, `parts-select`, `calibrate`), `sch layout-lint` (bbox overlap/spacing), and guardrails. |
| [`easyeda-pcb`](easyeda-pcb/SKILL.md) | **Operational** | How to drive `easyeda-agent` for **PCB**: switch to a PCB, read components/layers/nets/board, sync from the schematic (`import_changes`), and lay out components (move/rotate/align/distribute/grid-snap/cluster-arrange). |

## Why split

1. **Different change cadence & reviewer.** Conventions are EE domain knowledge
   (stable, reviewed by a hardware engineer). The operational skill changes with the
   tool (new actions, daemon/connector behavior). Separating them lets each evolve
   without churning the other.
2. **Shared across operational skills.** Schematic and (emerging) PCB both consume the
   same design conventions. One conventions skill keeps them DRY instead of letting
   `schematic-*` and `pcb-*` rules duplicate and drift.
3. **We paid the drift tax.** The flag-rotation truth once lived in `SKILL.md` context
   + four docs and had to be corrected in all of them. A single canonical home ends
   that.

## The boundary (what goes where)

- **Pure EE design knowledge** → `easyeda-conventions`. *Which way a flag points*,
  zone map, spacing, decoupling, selection criteria.
- **Tool mechanics & connector quirks** → `easyeda-schematic` / `CLAUDE.md` / memory.
  *That `createNetFlag` stores rotation negated and `connect_pin` compensates* is a
  connector quirk, not a design convention.

## The one rule that makes "fused but separate" work

**Single source — link, don't copy.** A convention or canonical datum lives in exactly
one place (`easyeda-conventions`); the operational skills `[link]` to it. The canonical
JSON is read across the skill boundary by the operational scripts
(`easyeda-schematic/scripts/{orient,bom-enrich}.py` →
`../../easyeda-conventions/references/…`), and `make lint-test` asserts the connector
and linter still derive the same orientation table.

## Authoring conventions (for these skills)

- **Language.** Write AI-facing routing metadata in **English** — skill `name` /
  `description` frontmatter, action names, navigational headers (this is what the
  agent matches on to find a skill). Keep substantive body content in its natural
  language; **Chinese prose stays Chinese**.
- **Reference files by bare name, not full path.** In prompts, action `Description`s,
  and code comments, cite `orientation.json` / `pcb-layout-conventions.md` — not the
  full repo path. The agent resolves by name, and the reference **survives file
  moves** (full paths are exactly what broke during this split). Only *clickable*
  markdown links use a short **relative** path.
- **One prompt, one home.** Guidance prose lives once — in the relevant skill /
  conventions doc. Code (action descriptions, comments) stays short and points to it
  by filename; don't grow a second copy of the prompt inside the code.
