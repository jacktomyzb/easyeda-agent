#!/usr/bin/env python3
"""Generate api-index.json from @jlceda/pro-api-types index.d.ts.

The official type package (Apache-2.0, pinned in extension/node_modules) is the
authoritative `eda.*` API surface. This walks its `declare global { class X_Y {…} }`
blocks and emits one searchable record per method:

    { "ns": "eda.dmt_Schematic", "method": "createSchematic",
      "sig": "createSchematic(boardName?: string): Promise<string | undefined>",
      "summary": "创建原理图", "stability": "beta" }

`easyeda api search/ls` (cmd_api.go) embeds and searches the result. Re-run after
bumping pro-api-types:

    python3 internal/apidoc/gen.py            # writes internal/apidoc/api-index.json
    python3 internal/apidoc/gen.py --dts <path> --out <path>
"""
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DTS = os.path.join(
    HERE, '..', '..', 'extension', 'node_modules', '@jlceda', 'pro-api-types', 'index.d.ts')
DEFAULT_OUT = os.path.join(HERE, 'api-index.json')

CLASS_RE = re.compile(r'^\s*class\s+([A-Za-z0-9_]+)\s*\{')
# A member declaration: `name(...` (method) — capture the name; the rest may span lines.
METHOD_RE = re.compile(r'^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(')
STABILITY_RE = re.compile(r'@(alpha|beta|deprecated|internal)\b')


def ns_of(cls):
    """DMT_Schematic -> eda.dmt_Schematic ; LIB_Device -> eda.lib_Device."""
    if '_' in cls:
        prefix, rest = cls.split('_', 1)
        return f'eda.{prefix.lower()}_{rest}'
    return f'eda.{cls[:1].lower()}{cls[1:]}'


def main():
    dts = DEFAULT_DTS
    out = DEFAULT_OUT
    av = sys.argv[1:]
    if '--dts' in av:
        dts = av[av.index('--dts') + 1]
    if '--out' in av:
        out = av[av.index('--out') + 1]

    with open(dts, encoding='utf-8') as f:
        lines = f.readlines()

    records = []
    cur_ns = None
    # Pending JSDoc state for the next member.
    doc_summary = None
    doc_stability = None
    in_doc = False
    doc_lines = []
    # Reserved words that look like methods but aren't API surface.
    skip = {'constructor', 'if', 'for', 'while', 'switch', 'catch', 'function', 'return'}

    for raw in lines:
        line = raw.rstrip('\n')
        stripped = line.strip()

        # Class / namespace boundary.
        m = CLASS_RE.match(line)
        if m:
            cur_ns = ns_of(m.group(1))
            doc_summary, doc_stability, in_doc, doc_lines = None, None, False, []
            continue

        # JSDoc block.
        if stripped.startswith('/**'):
            in_doc = True
            doc_lines = []
            if '*/' in stripped:
                in_doc = False
            continue
        if in_doc:
            doc_lines.append(stripped)
            if '*/' in stripped:
                in_doc = False
                # First non-tag content line is the summary.
                summary = None
                joined = ' '.join(doc_lines)
                stab = STABILITY_RE.search(joined)
                doc_stability = stab.group(1) if stab else None
                for dl in doc_lines:
                    t = dl.lstrip('*').strip()
                    if t and not t.startswith('@') and t != '/' and not t.endswith('*/') and t != '*/':
                        summary = t
                        break
                doc_summary = summary
            continue

        # Member declaration inside a class.
        if cur_ns:
            mm = METHOD_RE.match(line)
            if mm and mm.group(1) not in skip:
                method = mm.group(1)
                # Signature: from this line to the first ';' (handle multi-line).
                sig = stripped
                # If the declaration doesn't end here, leave it as the opening — enough
                # for search; full multi-line sigs are rare and noisy.
                sig = re.sub(r'\s+', ' ', sig).rstrip()
                records.append({
                    'ns': cur_ns,
                    'method': method,
                    'sig': sig,
                    'summary': doc_summary or '',
                    'stability': doc_stability or '',
                })
            # Consume the pending doc whether or not it matched a method.
            if stripped and not stripped.startswith('*'):
                doc_summary, doc_stability = None, None

    # De-dup (overloads) by (ns, method, sig).
    seen = set()
    deduped = []
    for r in records:
        key = (r['ns'], r['method'], r['sig'])
        if key in seen:
            continue
        seen.add(key)
        deduped.append(r)

    namespaces = sorted({r['ns'] for r in deduped})
    payload = {
        'source': '@jlceda/pro-api-types',
        'namespaceCount': len(namespaces),
        'methodCount': len(deduped),
        'records': deduped,
    }
    with open(out, 'w', encoding='utf-8') as f:
        json.dump(payload, f, ensure_ascii=False, indent=0)
        f.write('\n')
    print(f'wrote {out}: {len(namespaces)} namespaces, {len(deduped)} methods')


if __name__ == '__main__':
    main()
