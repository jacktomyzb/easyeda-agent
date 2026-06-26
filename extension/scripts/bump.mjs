// Bump the connector version in extension.json + package.json in lock-step.
//
// EasyEDA dedups installed extensions by (uuid, version): re-importing an .eext
// whose version equals the installed one is a no-op. So every connector change
// that the user needs to test must ship a NEW, higher version. Run this before
// building the .eext you hand to the user.
//
//   node scripts/bump.mjs            # patch: 0.2.0 -> 0.2.1
//   node scripts/bump.mjs minor      # minor: 0.2.1 -> 0.3.0
//   node scripts/bump.mjs major      # major: 0.3.0 -> 1.0.0
//   node scripts/bump.mjs 0.5.0      # set an explicit version

import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const extPath = path.join(here, '..', 'extension.json');
const pkgPath = path.join(here, '..', 'package.json');

function parse(v) {
	const m = /^(\d+)\.(\d+)\.(\d+)/.exec(v);
	if (!m) throw new Error(`unparseable version: ${v}`);
	return [Number(m[1]), Number(m[2]), Number(m[3])];
}

function nextVersion(current, mode) {
	if (/^\d+\.\d+\.\d+$/.test(mode)) return mode; // explicit version
	const [maj, min, pat] = parse(current);
	switch (mode) {
		case 'major': return `${maj + 1}.0.0`;
		case 'minor': return `${maj}.${min + 1}.0`;
		case 'patch':
		default: return `${maj}.${min}.${pat + 1}`;
	}
}

function writeJsonTabs(file, obj) {
	fs.writeFileSync(file, `${JSON.stringify(obj, null, '\t')}\n`, 'utf-8');
}

const args = process.argv.slice(2);
const mode = args.find((a) => !a.startsWith('--')) ?? 'patch';
const freshUuid = args.includes('--uuid'); // opt-in fallback: mint a new uuid

const ext = JSON.parse(fs.readFileSync(extPath, 'utf-8'));
const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf-8'));

const from = ext.version;
const to = nextVersion(from, mode);

// UUID policy: keep it STABLE by default so the build is the SAME extension you
// update in place (uninstall the old one in EasyEDA's 已安装 tab → import). Only
// the fallback path (--uuid) mints a fresh uuid, which imports as a NEW extension
// entry (no uninstall needed, but you must delete the stale one). EasyEDA dedups
// installed extensions by uuid, so a same-uuid re-import requires that uninstall.
// A minted uuid must match the connector's testUuid: 32 lowercase hex chars.
const fromUuid = ext.uuid;
const toUuid = freshUuid ? crypto.randomUUID().replaceAll('-', '') : fromUuid;

ext.version = to;
ext.uuid = toUuid;
pkg.version = to;

writeJsonTabs(extPath, ext);
writeJsonTabs(pkgPath, pkg);

console.log(`version ${from} -> ${to}  (extension.json + package.json)`);
console.log(freshUuid
	? `uuid    ${fromUuid} -> ${toUuid}  (FRESH uuid — imports as a new extension; delete the old one)`
	: `uuid    ${toUuid}  (unchanged — update in place: uninstall old in 已安装, then import)`);
