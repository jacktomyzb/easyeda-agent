#!/bin/bash
# Data-only schematic linter: pull the full layout from a connected EasyEDA
# window in ONE call and report problem points — no screenshots needed.
#
#   tools/schematic-lint/lint.sh [projectName] [host] [portStart] [portEnd]
#
# Resolves the live windowId by project name (default "ceshi"), runs probe.js
# via debug.exec_js, and pipes the layout JSON through lint.py.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/../.." && pwd)"
BIN="$ROOT/bin/easyeda"
PROJ="${1:-ceshi}"
HOST="${2:-127.0.0.1}"
PS="${3:-49620}"; PE="${4:-49629}"

[ -x "$BIN" ] || { echo "build first: make build" >&2; exit 1; }

# 1. find the daemon port (first that reports service easyeda-agent)
PORT=""
for p in $(seq "$PS" "$PE"); do
  if curl -fsS --max-time 1 "http://$HOST:$p/health" 2>/dev/null | grep -q 'easyeda-agent'; then
    PORT="$p"; break
  fi
done
[ -n "$PORT" ] || { echo "no easyeda-agent daemon on $HOST:$PS-$PE (run: $BIN daemon)" >&2; exit 1; }

# 2. resolve the live windowId for the project
WIN="$("$BIN" health 2>/dev/null | python3 -c "
import json,sys
wins=json.load(sys.stdin).get('found',{}).get('raw',{}).get('windows',[])
for w in wins:
    if w.get('context',{}).get('projectName')=='$PROJ': print(w['windowId']); break
")"
[ -n "$WIN" ] || { echo "no connected window for project '$PROJ'" >&2; exit 1; }

# 3. pull full layout via debug.exec_js, then lint
PROBE="$(cat "$DIR/probe.js")"
python3 - "$DIR/lint.py" "$HOST" "$PORT" "$WIN" "$PROBE" <<'PY'
import json, sys, subprocess, urllib.request
lintpy, host, port, win, probe = sys.argv[1:6]
body = json.dumps({"action": "debug.exec_js", "windowId": win, "payload": {"code": probe}}).encode()
req = urllib.request.Request(f"http://{host}:{port}/action", data=body, headers={"Content-Type": "application/json"})
resp = json.load(urllib.request.urlopen(req, timeout=60))
if not resp.get("ok"):
    print("probe failed:", resp.get("error"), file=sys.stderr); sys.exit(1)
import tempfile, os
with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
    json.dump(resp["result"]["value"], f); path = f.name
subprocess.run([sys.executable, lintpy, path])
os.unlink(path)
PY
