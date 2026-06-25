#!/usr/bin/env python3
"""Rule-trust harness for schematic-lint.

Two guards keep the linter's verdicts trustworthy:

1. Orientation-table consistency — the canonical spec (orientation.json) must
   derive back to its own frozenTable, and the +90° cycle law must hold. This is
   what keeps the Python check and the TS connect_pin writer from ever drifting.
   (Ground-truth of the 3 anchors vs. live bbox is calibrate.js, run on import.)

2. Fixture goldens — each layout in fixtures/ is linted and diffed against the
   frozen expected output in golden/. A rule that starts mis-firing on a
   known-good board, or stops catching a known-bad one, fails here.

    python3 tests/run.py            # check (exit 1 on any mismatch)
    python3 tests/run.py --update   # re-freeze goldens after an intended change
"""
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)  # tools/schematic-lint
LINT = os.path.join(ROOT, 'lint.py')
FIXTURES = os.path.join(HERE, 'fixtures')
GOLDEN = os.path.join(HERE, 'golden')

sys.path.insert(0, ROOT)
import orient  # noqa: E402

GREEN, RED, DIM, RESET = '\033[32m', '\033[31m', '\033[2m', '\033[0m'


def check_orientation(failures):
    spec = orient.load_spec()
    cycle, anchors, frozen = spec['rotationCycle'], spec['bodyAnchorAtRot0'], spec['frozenTable']
    derived = orient.derive(cycle, anchors)

    # The derived table must reproduce the human-readable frozenTable exactly.
    for fam in frozen:
        for d, want in frozen[fam].items():
            got = derived[fam][d]
            if got != want:
                failures.append(
                    f"orientation: {fam}.{d} derived={got} but frozenTable={want} "
                    f"(edit anchors/cycle, then --update — never hand-edit frozenTable)")

    # Structural law: every entry is a multiple of 90 in {0,90,180,270} and each
    # family is a bijection direction->rotation (a pure rotation of the cycle).
    for fam, table in derived.items():
        rots = sorted(table.values())
        if rots != [0, 90, 180, 270]:
            failures.append(f"orientation: {fam} rotations {rots} are not a clean 0/90/180/270 set")

    if not failures:
        print(f"{GREEN}✓{RESET} orientation table: spec derives to frozenTable; cycle law holds")


def run_lint(path):
    proc = subprocess.run([sys.executable, LINT, path], capture_output=True, text=True)
    if proc.returncode != 0:
        return f"<lint.py crashed: rc={proc.returncode}>\n{proc.stderr}"
    return proc.stdout


def check_fixtures(update, failures):
    os.makedirs(GOLDEN, exist_ok=True)
    fixtures = sorted(f for f in os.listdir(FIXTURES) if f.endswith('.json'))
    for fx in fixtures:
        name = fx[:-len('.json')]
        out = run_lint(os.path.join(FIXTURES, fx))
        gpath = os.path.join(GOLDEN, name + '.txt')
        if update:
            with open(gpath, 'w') as f:
                f.write(out)
            print(f"{DIM}↻ froze golden/{name}.txt{RESET}")
            continue
        if not os.path.exists(gpath):
            failures.append(f"fixture {name}: no golden (run --update)")
            continue
        with open(gpath) as f:
            want = f.read()
        if out != want:
            failures.append(f"fixture {name}: output differs from golden/{name}.txt")
            _print_diff(want, out)
        else:
            print(f"{GREEN}✓{RESET} fixture {name}")


def _print_diff(want, got):
    import difflib
    diff = difflib.unified_diff(
        want.splitlines(), got.splitlines(),
        fromfile='golden', tofile='actual', lineterm='')
    for line in diff:
        print(f"  {DIM}{line}{RESET}")


def main():
    update = '--update' in sys.argv
    failures = []
    check_orientation(failures)
    check_fixtures(update, failures)
    if update:
        print("goldens re-frozen.")
        return 0
    if failures:
        print(f"\n{RED}✗ {len(failures)} failure(s):{RESET}")
        for f in failures:
            print(f"  - {f}")
        return 1
    print(f"\n{GREEN}all rule-trust checks passed{RESET}")
    return 0


if __name__ == '__main__':
    sys.exit(main())
